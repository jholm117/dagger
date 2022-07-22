package api

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	tools "github.com/bhoriuchi/graphql-go-tools"
	"github.com/containerd/containerd/platforms"
	"github.com/graphql-go/graphql"
	"github.com/graphql-go/graphql/language/ast"
	"github.com/graphql-go/graphql/language/parser"
	"github.com/graphql-go/graphql/language/printer"
	"github.com/moby/buildkit/client/llb"
	dockerfilebuilder "github.com/moby/buildkit/frontend/dockerfile/builder"
	bkgw "github.com/moby/buildkit/frontend/gateway/client"
	"github.com/moby/buildkit/solver/pb"
	specs "github.com/opencontainers/image-spec/specs-go/v1"
)

const (
	daggerSockName = "dagger-sock"
)

// Mirrors handler.RequestOptions, but includes omitempty for better compatibility
// with other servers like apollo (which don't seem to like "operationName": "").
type GraphQLRequest struct {
	Query         string                 `json:"query,omitempty"`
	Variables     map[string]interface{} `json:"variables,omitempty"`
	OperationName string                 `json:"operationName,omitempty"`
}

// FS is either llb representing the filesystem or a graphql query for obtaining that llb
// This is opaque to clients; to them FS is a scalar.
type FS struct {
	PB *pb.Definition
	GraphQLRequest
}

// FS encodes to the base64 encoding of its JSON representation
func (fs FS) MarshalText() ([]byte, error) {
	type marshalFS FS
	jsonBytes, err := json.Marshal(marshalFS(fs))
	if err != nil {
		return nil, err
	}
	b64Bytes := make([]byte, base64.StdEncoding.EncodedLen(len(jsonBytes)))
	base64.StdEncoding.Encode(b64Bytes, jsonBytes)
	return b64Bytes, nil
}

func (fs *FS) UnmarshalText(b64Bytes []byte) error {
	type marshalFS FS
	jsonBytes := make([]byte, base64.StdEncoding.DecodedLen(len(b64Bytes)))
	n, err := base64.StdEncoding.Decode(jsonBytes, b64Bytes)
	if err != nil {
		return fmt.Errorf("failed to unmarshal fs bytes: %v", err)
	}
	jsonBytes = jsonBytes[:n]
	var result marshalFS
	if err := json.Unmarshal(jsonBytes, &result); err != nil {
		return fmt.Errorf("failed to unmarshal result: %v", err)
	}
	fs.PB = result.PB
	fs.GraphQLRequest = result.GraphQLRequest
	return nil
}

func (fs FS) ToState() (llb.State, error) {
	if fs.PB == nil {
		return llb.State{}, fmt.Errorf("FS is not evaluated")
	}
	defop, err := llb.NewDefinitionOp(fs.PB)
	if err != nil {
		return llb.State{}, err
	}
	return llb.NewState(defop), nil
}

/*
	type AlpineBuild {
		fs: FS!
	}
	type Query {
		build(pkgs: [String]!): AlpineBuild
	}

	converts to:

	type AlpineBuild {
		fs: FS!
	}
	type Alpine {
		build(pkgs: [String]!): AlpineBuild
	}
	type Query {
		alpine: Alpine!
	}
*/
func parseSchema(pkgName string, typeDefs string) (*tools.ExecutableSchema, error) {
	capName := strings.ToUpper(string(pkgName[0])) + pkgName[1:]
	resolverMap := tools.ResolverMap{
		"Query": &tools.ObjectResolver{
			Fields: tools.FieldResolveMap{
				pkgName: &tools.FieldResolve{
					Resolve: func(p graphql.ResolveParams) (interface{}, error) {
						return struct{}{}, nil
					},
				},
			},
		},
		capName: &tools.ObjectResolver{
			Fields: tools.FieldResolveMap{},
		},
	}

	doc, err := parser.Parse(parser.ParseParams{Source: typeDefs})
	if err != nil {
		return nil, err
	}

	var actions []string
	var otherObjects []string
	for _, def := range doc.Definitions {
		if def, ok := def.(*ast.ObjectDefinition); ok {
			if def.Name.Value == "Query" {
				for _, field := range def.Fields {
					actions = append(actions, printer.Print(field).(string))
					resolverMap[capName].(*tools.ObjectResolver).Fields[field.Name.Value] = &tools.FieldResolve{
						Resolve: actionFieldToResolver(pkgName, field.Name.Value),
					}
				}
			} else {
				otherObjects = append(otherObjects, printer.Print(def).(string))
			}
		}
	}

	return &tools.ExecutableSchema{
		TypeDefs: fmt.Sprintf(`
%s
type %s {
	%s
}
type Query {
	%s: %s!
}
	`, strings.Join(otherObjects, "\n"), capName, strings.Join(actions, "\n"), pkgName, capName),
		Resolvers: resolverMap,
	}, nil
}

func actionFieldToResolver(pkgName, actionName string) graphql.FieldResolveFn {
	return func(p graphql.ResolveParams) (interface{}, error) {
		if !shouldEval(p.Context) {
			return lazyResolve(p)
		}

		// the action doesn't know we stitch its queries under the package name, patch the query we send to here
		queryOp := p.Info.Operation.(*ast.OperationDefinition)
		packageSelect := queryOp.SelectionSet.Selections[0].(*ast.Field)
		queryOp.SelectionSet.Selections = packageSelect.SelectionSet.Selections

		inputBytes, err := json.Marshal(GraphQLRequest{
			Query:         printer.Print(queryOp).(string),
			Variables:     p.Info.VariableValues,
			OperationName: getOperationName(p),
		})
		if err != nil {
			return nil, err
		}
		// fmt.Printf("requesting %s\n", string(inputBytes))

		input := llb.Scratch().File(llb.Mkfile("/dagger.json", 0644, inputBytes))

		fsState, err := daggerPackages[pkgName].FS.ToState()
		if err != nil {
			return nil, err
		}
		st := fsState.Run(
			llb.Args([]string{"/entrypoint"}),
			llb.AddSSHSocket(
				llb.SSHID(daggerSockName),
				llb.SSHSocketTarget("/dagger.sock"),
			),
			llb.AddMount("/inputs", input, llb.Readonly),
			llb.AddMount("/tmp", llb.Scratch(), llb.Tmpfs()),
			llb.ReadonlyRootFS(),
		)

		// TODO: /mnt should maybe be configurable?
		for path, fs := range collectFSPaths(p.Args, "/mnt", make(map[string]FS)) {
			fsState, err := fs.ToState()
			if err != nil {
				return nil, err
			}
			// TODO: it should be possible for this to be outputtable by the action; the only question
			// is how to expose that ability in a non-confusing way, just needs more thought
			st.AddMount(path, fsState, llb.ForceNoOutput)
		}

		outputMnt := st.AddMount("/outputs", llb.Scratch())
		outputDef, err := outputMnt.Marshal(p.Context, llb.Platform(getPlatform(p)), llb.WithCustomName(fmt.Sprintf("%s.%s", pkgName, actionName)))
		if err != nil {
			return nil, err
		}
		gw, err := getGatewayClient(p)
		if err != nil {
			return nil, err
		}
		res, err := gw.Solve(context.Background(), bkgw.SolveRequest{
			Evaluate:   true,
			Definition: outputDef.ToPB(),
		})
		if err != nil {
			return nil, err
		}
		ref, err := res.SingleRef()
		if err != nil {
			return nil, err
		}
		outputBytes, err := ref.ReadFile(p.Context, bkgw.ReadRequest{
			Filename: "/dagger.json",
		})
		if err != nil {
			return nil, err
		}
		fmt.Printf("%s.%s output: %s\n", pkgName, actionName, string(outputBytes))
		var output interface{}
		if err := json.Unmarshal(outputBytes, &output); err != nil {
			return nil, fmt.Errorf("failed to unmarshal output: %w", err)
		}
		for _, parentField := range append([]any{"data"}, p.Info.Path.AsArray()[1:]...) { // skip first field, which is the package name
			outputMap, ok := output.(map[string]interface{})
			if !ok {
				return nil, fmt.Errorf("output is not a map: %+v", output)
			}
			output = outputMap[parentField.(string)]
		}
		return output, nil
	}
}

func collectFSPaths(arg interface{}, curPath string, fsPaths map[string]FS) map[string]FS {
	switch arg := arg.(type) {
	case FS:
		// TODO: make sure there can't be any shenanigans with args named e.g. ../../../foo/bar
		fsPaths[curPath] = arg
	case map[string]interface{}:
		for k, v := range arg {
			fsPaths = collectFSPaths(v, filepath.Join(curPath, k), fsPaths)
		}
	case []interface{}:
		for i, v := range arg {
			// TODO: path format technically works but weird as hell, gotta be a better way
			fsPaths = collectFSPaths(v, fmt.Sprintf("%s/%d", curPath, i), fsPaths)
		}
	}
	return fsPaths
}

type daggerPackage struct {
	Name   string
	FS     FS
	Schema tools.ExecutableSchema
}

// TODO: shouldn't be global vars, pass through resolve context, make sure synchronization is handled, etc.
var schema graphql.Schema
var daggerPackages map[string]daggerPackage

func reloadSchemas() error {
	// tools.MakeExecutableSchema doesn't actually merge multiple schemas, it just overwrites any
	// overlapping types This is fine for the moment except for the Query type, which we need to be an
	// actual merge, so we do that ourselves here by merging the Query resolvers and appending a merged
	// Query type to the typeDefs.
	var queryFields []string
	var otherObjects []string
	for _, daggerPkg := range daggerPackages {
		doc, err := parser.Parse(parser.ParseParams{Source: daggerPkg.Schema.TypeDefs})
		if err != nil {
			return err
		}
		for _, def := range doc.Definitions {
			if def, ok := def.(*ast.ObjectDefinition); ok {
				if def.Name.Value == "Query" {
					for _, field := range def.Fields {
						queryFields = append(queryFields, printer.Print(field).(string))
					}
					continue
				}
			}
			otherObjects = append(otherObjects, printer.Print(def).(string))
		}
	}

	resolvers := make(map[string]interface{})
	for _, daggerPkg := range daggerPackages {
		for k, v := range daggerPkg.Schema.Resolvers {
			// TODO: need more general solution, verification that overwrites aren't happening, etc.
			if k == "Query" {
				if existing, ok := resolvers[k]; ok {
					existing := existing.(*tools.ObjectResolver)
					for field, fieldResolver := range v.(*tools.ObjectResolver).Fields {
						existing.Fields[field] = fieldResolver
					}
					v = existing
				}
			}
			resolvers[k] = v
		}
	}

	typeDefs := fmt.Sprintf(`
%s
type Query {
  %s
}
	`, strings.Join(otherObjects, "\n"), strings.Join(queryFields, "\n "))

	var err error
	schema, err = tools.MakeExecutableSchema(tools.ExecutableSchema{
		TypeDefs:  typeDefs,
		Resolvers: resolvers,
	})
	if err != nil {
		return err
	}

	return nil
}

func init() {
	daggerPackages = make(map[string]daggerPackage)
	daggerPackages["core"] = daggerPackage{
		Name: "core",
		Schema: tools.ExecutableSchema{
			TypeDefs: `
scalar FS

type CoreImage {
	fs: FS!
}

input CoreMount {
	path: String!
	fs: FS!
}
input CoreExecInput {
	mounts: [CoreMount!]!
	args: [String!]!
}
type CoreExecOutput {
	mount(path: String!): FS
}
type CoreExec {
	fs: FS!
}

type Core {
	image(ref: String!): CoreImage
	# exec(input: CoreExecInput!): CoreExecOutput
	exec(fs: FS!, args: [String]!): CoreExec
	dockerfile(context: FS!, dockerfileName: String): FS!
}

type Query {
	core: Core!
}

type Package {
	name: String!
	fs: FS!
}

type Mutation {
	import(name: String!, fs: FS!): Package
	readfile(fs: FS!, path: String!): String
	clientdir(id: String!): FS
	readsecret(id: String!): String
}
		`,
			Resolvers: tools.ResolverMap{
				"Query": &tools.ObjectResolver{
					Fields: tools.FieldResolveMap{
						"core": &tools.FieldResolve{
							Resolve: func(p graphql.ResolveParams) (interface{}, error) {
								return struct{}{}, nil
							},
						},
					},
				},
				"CoreExecOutput": &tools.ObjectResolver{
					Fields: tools.FieldResolveMap{
						"mount": &tools.FieldResolve{
							Resolve: func(p graphql.ResolveParams) (interface{}, error) {
								if lazyOutput, ok := p.Source.(map[string]interface{}); ok {
									return lazyOutput["mount"], nil
								}
								fsOutputs, ok := p.Source.(map[string]string)
								if !ok {
									return nil, fmt.Errorf("unexpected core exec source type %T", p.Source)
								}
								rawPath, ok := p.Args["path"]
								if !ok {
									return nil, fmt.Errorf("missing path argument")
								}
								path, ok := rawPath.(string)
								if !ok {
									return nil, fmt.Errorf("path argument is not a string")
								}
								fsstr, ok := fsOutputs[path]
								if !ok {
									return nil, fmt.Errorf("mount at path %q not found", path)
								}
								return fsstr, nil
							},
						},
					},
				},
				"Core": &tools.ObjectResolver{
					Fields: tools.FieldResolveMap{
						"image": &tools.FieldResolve{
							Resolve: func(p graphql.ResolveParams) (interface{}, error) {
								if !shouldEval(p.Context) {
									return lazyResolve(p)
								}
								rawRef, ok := p.Args["ref"]
								if !ok {
									return nil, fmt.Errorf("missing ref")
								}
								/* TODO: switch back to DaggerString once re-integrated with generated clients
								var ref DaggerString
								if err := ref.UnmarshalAny(rawRef); err != nil {
									return nil, err
								}
								ref, err := ref.Evaluate(p.Context)
								if err != nil {
									return nil, fmt.Errorf("error evaluating image ref: %v", err)
								}
								*/
								ref, ok := rawRef.(string)
								if !ok {
									return nil, fmt.Errorf("ref is not a string")
								}
								// llbdef, err := llb.Image(*ref.Value).Marshal(p.Context, llb.Platform(getPlatform(p)))
								llbdef, err := llb.Image(ref).Marshal(p.Context, llb.Platform(getPlatform(p)))
								if err != nil {
									return nil, err
								}
								return map[string]interface{}{
									"fs": FS{PB: llbdef.ToPB()},
								}, nil
							},
						},
						"exec": &tools.FieldResolve{
							Resolve: func(p graphql.ResolveParams) (interface{}, error) {
								if !shouldEval(p.Context) {
									return lazyResolve(p)
								}
								fs, ok := p.Args["fs"].(FS)
								if !ok {
									return nil, fmt.Errorf("invalid fs")
								}
								rawArgs, ok := p.Args["args"].([]interface{})
								if !ok {
									return nil, fmt.Errorf("invalid args")
								}
								var args []string
								for _, arg := range rawArgs {
									if arg, ok := arg.(string); ok {
										args = append(args, arg)
									} else {
										return nil, fmt.Errorf("invalid arg")
									}
								}
								fs, err := fs.Evaluate(p.Context)
								if err != nil {
									return nil, err
								}
								fsState, err := fs.ToState()
								if err != nil {
									return nil, err
								}
								llbdef, err := fsState.Run(llb.Args(args)).Root().Marshal(p.Context, llb.Platform(getPlatform(p)))
								if err != nil {
									return nil, err
								}
								return map[string]interface{}{
									"fs": FS{PB: llbdef.ToPB()},
								}, nil
							},
						},
						"dockerfile": &tools.FieldResolve{
							Resolve: func(p graphql.ResolveParams) (interface{}, error) {
								if !shouldEval(p.Context) {
									return lazyResolve(p)
								}

								fs, ok := p.Args["context"].(FS)
								if !ok {
									return nil, fmt.Errorf("invalid context")
								}

								var dockerfileName string
								rawDockerfileName, ok := p.Args["dockerfileName"]
								if ok {
									dockerfileName, ok = rawDockerfileName.(string)
									if !ok {
										return nil, fmt.Errorf("invalid dockerfile name: %+v", rawDockerfileName)
									}
								}

								fs, err := fs.Evaluate(p.Context)
								if err != nil {
									return nil, err
								}
								gw, err := getGatewayClient(p)
								if err != nil {
									return nil, err
								}

								opts := map[string]string{
									"platform": platforms.Format(getPlatform(p)),
								}
								inputs := map[string]*pb.Definition{
									dockerfilebuilder.DefaultLocalNameContext:    fs.PB,
									dockerfilebuilder.DefaultLocalNameDockerfile: fs.PB,
								}
								if dockerfileName != "" {
									opts["filename"] = dockerfileName
								}
								res, err := gw.Solve(p.Context, bkgw.SolveRequest{
									Frontend:       "dockerfile.v0",
									FrontendOpt:    opts,
									FrontendInputs: inputs,
								})
								if err != nil {
									return nil, err
								}

								bkref, err := res.SingleRef()
								if err != nil {
									return nil, err
								}
								st, err := bkref.ToState()
								if err != nil {
									return nil, err
								}
								llbdef, err := st.Marshal(p.Context, llb.Platform(getPlatform(p)))
								if err != nil {
									return nil, err
								}
								return FS{PB: llbdef.ToPB()}, nil
							},
						},
					},
					// 		Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					// 			if !shouldEval(p.Context) {
					// 				return lazyResolve(p)
					// 			}
					// 			input, ok := p.Args["input"].(map[string]interface{})
					// 			if !ok {
					// 				return nil, fmt.Errorf("invalid fs")
					// 			}

					// 			rawMounts, ok := input["mounts"].([]interface{})
					// 			if !ok {
					// 				return nil, fmt.Errorf("invalid mounts")
					// 			}
					// 			inputMounts := make(map[string]FS)
					// 			for _, rawMount := range rawMounts {
					// 				mount, ok := rawMount.(map[string]interface{})
					// 				if !ok {
					// 					return nil, fmt.Errorf("invalid mount: %T", rawMount)
					// 				}
					// 				path, ok := mount["path"].(string)
					// 				if !ok {
					// 					return nil, fmt.Errorf("invalid mount path")
					// 				}
					// 				path = filepath.Clean(path)
					// 				fsstr, ok := mount["fs"].(string)
					// 				if !ok {
					// 					return nil, fmt.Errorf("invalid mount fs")
					// 				}
					// 				var fs FS
					// 				if err := fs.UnmarshalText([]byte(fsstr)); err != nil {
					// 					return nil, err
					// 				}
					// 				inputMounts[path] = fs
					// 			}
					// 			rootFS, ok := inputMounts["/"]
					// 			if !ok {
					// 				return nil, fmt.Errorf("missing root fs")
					// 			}

					// 			rawArgs, ok := input["args"].([]interface{})
					// 			if !ok {
					// 				return nil, fmt.Errorf("invalid args")
					// 			}
					// 			var args []string
					// 			for _, rawArg := range rawArgs {
					// 				/* TODO: switch back to DaggerString once re-integrated with generated clients
					// 				var arg DaggerString
					// 				if err := arg.UnmarshalAny(rawArg); err != nil {
					// 					return nil, fmt.Errorf("invalid arg: %w", err)
					// 				}
					// 				arg, err := arg.Evaluate(p.Context)
					// 				if err != nil {
					// 					return nil, fmt.Errorf("error evaluating arg: %v", err)
					// 				}
					// 				args = append(args, *arg.Value)
					// 				*/
					// 				arg, ok := rawArg.(string)
					// 				if !ok {
					// 					return nil, fmt.Errorf("invalid arg: %T", rawArg)
					// 				}
					// 				args = append(args, arg)
					// 			}

					// 			rootFS, err := rootFS.Evaluate(p.Context)
					// 			if err != nil {
					// 				return nil, err
					// 			}
					// 			state, err := rootFS.ToState()
					// 			if err != nil {
					// 				return nil, err
					// 			}
					// 			execState := state.Run(llb.Args(args))
					// 			outputStates := map[string]llb.State{
					// 				"/": execState.Root(),
					// 			}
					// 			for path, inputFS := range inputMounts {
					// 				if path == "/" {
					// 					continue
					// 				}
					// 				inputFS, err := inputFS.Evaluate(p.Context)
					// 				if err != nil {
					// 					return nil, err
					// 				}
					// 				inputState, err := inputFS.ToState()
					// 				if err != nil {
					// 					return nil, err
					// 				}
					// 				outputStates[path] = execState.AddMount(path, inputState)
					// 			}
					// 			fsOutputs := make(map[string]string)
					// 			for path, outputState := range outputStates {
					// 				llbdef, err := outputState.Marshal(p.Context, llb.Platform(getPlatform(p)))
					// 				if err != nil {
					// 					return nil, err
					// 				}
					// 				fsbytes, err := FS{PB: llbdef.ToPB()}.MarshalText()
					// 				if err != nil {
					// 					return nil, err
					// 				}
					// 				fsOutputs[path] = string(fsbytes)
					// 			}
					// 			return fsOutputs, nil
					// 		},
					// 	},
					// },
				},

				"Mutation": &tools.ObjectResolver{
					Fields: tools.FieldResolveMap{
						"import": &tools.FieldResolve{
							Resolve: func(p graphql.ResolveParams) (interface{}, error) {
								// TODO: make sure not duped
								pkgName, ok := p.Args["name"].(string)
								if !ok {
									return nil, fmt.Errorf("invalid package name")
								}

								fs, ok := p.Args["fs"].(FS)
								if !ok {
									return nil, fmt.Errorf("invalid fs")
								}
								fs, err := fs.Evaluate(p.Context)
								if err != nil {
									return nil, fmt.Errorf("failed to evaluate fs: %v", err)
								}

								gw, err := getGatewayClient(p)
								if err != nil {
									return nil, err
								}
								res, err := gw.Solve(context.Background(), bkgw.SolveRequest{
									Evaluate:   true,
									Definition: fs.PB,
								})
								if err != nil {
									return nil, err
								}
								bkref, err := res.SingleRef()
								if err != nil {
									return nil, err
								}
								outputBytes, err := bkref.ReadFile(p.Context, bkgw.ReadRequest{
									Filename: "/dagger.graphql",
								})
								if err != nil {
									return nil, err
								}
								parsed, err := parseSchema(pkgName, string(outputBytes))
								if err != nil {
									return nil, err
								}
								daggerPackages[pkgName] = daggerPackage{
									Name:   pkgName,
									FS:     fs,
									Schema: *parsed,
								}

								if err := reloadSchemas(); err != nil {
									return nil, err
								}

								// TODO: also return schema probably
								return map[string]interface{}{
									"name": pkgName,
									"fs":   fs,
								}, nil
							},
						},
						"evaluate": &tools.FieldResolve{
							Resolve: func(p graphql.ResolveParams) (interface{}, error) {
								fs, ok := p.Args["fs"].(FS)
								if !ok {
									return nil, fmt.Errorf("invalid fs")
								}
								fs, err := fs.Evaluate(p.Context)
								if err != nil {
									return nil, fmt.Errorf("failed to evaluate fs: %v", err)
								}
								gw, err := getGatewayClient(p)
								if err != nil {
									return nil, err
								}
								_, err = gw.Solve(context.Background(), bkgw.SolveRequest{
									Evaluate:   true,
									Definition: fs.PB,
								})
								if err != nil {
									return nil, err
								}
								return fs, nil
							},
						},
						"readfile": &tools.FieldResolve{
							Resolve: func(p graphql.ResolveParams) (interface{}, error) {
								fs, ok := p.Args["fs"].(FS)
								if !ok {
									return nil, fmt.Errorf("invalid fs")
								}
								path, ok := p.Args["path"].(string)
								if !ok {
									return nil, fmt.Errorf("invalid path")
								}
								fs, err := fs.Evaluate(p.Context)
								if err != nil {
									return nil, err
								}
								gw, err := getGatewayClient(p)
								if err != nil {
									return nil, err
								}
								res, err := gw.Solve(context.Background(), bkgw.SolveRequest{
									Evaluate:   true,
									Definition: fs.PB,
								})
								if err != nil {
									return nil, err
								}
								ref, err := res.SingleRef()
								if err != nil {
									return nil, err
								}
								outputBytes, err := ref.ReadFile(p.Context, bkgw.ReadRequest{
									Filename: path,
								})
								if err != nil {
									return nil, err
								}
								return string(outputBytes), nil
							},
						},
						"readsecret": &tools.FieldResolve{
							Resolve: func(p graphql.ResolveParams) (interface{}, error) {
								id, ok := p.Args["id"].(string)
								if !ok {
									return nil, fmt.Errorf("invalid secret id")
								}
								secrets := getSecrets(p)
								if secrets == nil {
									return nil, fmt.Errorf("no secrets")
								}
								secret, ok := secrets[id]
								if !ok {
									return nil, fmt.Errorf("no secret with id %s", id)
								}
								return secret, nil
							},
						},
						"clientdir": &tools.FieldResolve{
							Resolve: func(p graphql.ResolveParams) (interface{}, error) {
								id, ok := p.Args["id"].(string)
								if !ok {
									return nil, fmt.Errorf("invalid clientdir id")
								}
								llbdef, err := llb.Local(id).Marshal(p.Context)
								if err != nil {
									return nil, err
								}
								return FS{PB: llbdef.ToPB()}, nil
							},
						},
					},
				},
				"FS": &tools.ScalarResolver{
					Serialize: func(value interface{}) interface{} {
						switch v := value.(type) {
						case FS:
							fsbytes, err := v.MarshalText()
							if err != nil {
								panic(err)
							}
							return string(fsbytes)
						case string:
							return v
						default:
							panic(fmt.Sprintf("unexpected fs type %T", v))
						}
					},
					ParseValue: func(value interface{}) interface{} {
						switch v := value.(type) {
						case string:
							var fs FS
							if err := fs.UnmarshalText([]byte(v)); err != nil {
								panic(err)
							}
							return fs
						default:
							panic(fmt.Sprintf("unexpected fs value type %T", v))
						}
					},
					ParseLiteral: func(valueAST ast.Value) interface{} {
						switch valueAST := valueAST.(type) {
						case *ast.StringValue:
							var fs FS
							if err := fs.UnmarshalText([]byte(valueAST.Value)); err != nil {
								panic(err)
							}
							return fs
						default:
							panic(fmt.Sprintf("unexpected fs literal type: %T", valueAST))
						}
					},
				},
				"DaggerString": &tools.ScalarResolver{
					Serialize: func(value interface{}) interface{} {
						return value
					},
					ParseValue: func(value interface{}) interface{} {
						return value
					},
					ParseLiteral: func(valueAST ast.Value) interface{} {
						switch valueAST := valueAST.(type) {
						case *ast.StringValue:
							return valueAST.Value
						case *ast.ListValue:
							if len(valueAST.Values) != 1 {
								panic(fmt.Sprintf("invalid dagger string: %+v", valueAST.Values))
							}
							elem, ok := valueAST.Values[0].(*ast.StringValue)
							if !ok {
								panic(fmt.Sprintf("invalid dagger string: %+v", valueAST.Values))
							}
							return []any{elem.Value}
						default:
							panic(fmt.Sprintf("unsupported fs type: %T", valueAST))
						}
					},
				},
			},
		},
	}

	if err := reloadSchemas(); err != nil {
		panic(err)
	}
}

type gatewayClientKey struct{}

func withGatewayClient(ctx context.Context, gw bkgw.Client) context.Context {
	return context.WithValue(ctx, gatewayClientKey{}, gw)
}

func getGatewayClient(p graphql.ResolveParams) (bkgw.Client, error) {
	v := p.Context.Value(gatewayClientKey{})
	if v == nil {
		return nil, fmt.Errorf("no gateway client")
	}
	return v.(bkgw.Client), nil
}

type platformKey struct{}

func withPlatform(ctx context.Context, platform *specs.Platform) context.Context {
	return context.WithValue(ctx, platformKey{}, platform)
}

func getPlatform(p graphql.ResolveParams) specs.Platform {
	v := p.Context.Value(platformKey{})
	if v == nil {
		return platforms.DefaultSpec()
	}
	return *v.(*specs.Platform)
}

type secretsKey struct{}

func withSecrets(ctx context.Context, secrets map[string]string) context.Context {
	return context.WithValue(ctx, secretsKey{}, secrets)
}

func getSecrets(p graphql.ResolveParams) map[string]string {
	v := p.Context.Value(secretsKey{})
	if v == nil {
		return nil
	}
	return v.(map[string]string)
}

func getQuery(p graphql.ResolveParams) string {
	return printer.Print(p.Info.Operation).(string)
}

func getOperationName(p graphql.ResolveParams) string {
	name := p.Info.Operation.(*ast.OperationDefinition).Name
	if name == nil {
		return ""
	}
	return name.Value
}
