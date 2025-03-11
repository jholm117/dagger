// Code generated by dagger. DO NOT EDIT.

package dag

import (
	"context"
	"os"
	"sync"

	dagger "dagger.io/dagger"
)

var client *dagger.Client
var clientMu sync.Mutex

func initClient() *dagger.Client {
	clientMu.Lock()
	defer clientMu.Unlock()

	if client == nil {
		opts := []dagger.ClientOpt{
			dagger.WithLogOutput(os.Stdout),
		}

		var err error
		client, err = dagger.Connect(context.Background(), opts...)
		if err != nil {
			panic(err)
		}
	}
	return client
}

// Close the engine connection
func Close() error {
	clientMu.Lock()
	defer clientMu.Unlock()

	var err error
	if client != nil {
		err = client.Close()
		client = nil
	}
	return err
}

// Retrieves a container builtin to the engine.
func BuiltinContainer(digest string) *dagger.Container {
	client := initClient()
	return client.BuiltinContainer(digest)
}

// Constructs a cache volume for a given cache key.
func CacheVolume(key string, opts ...dagger.CacheVolumeOpts) *dagger.CacheVolume {
	client := initClient()
	return client.CacheVolume(key, opts...)
}

// Creates a scratch container.
//
// Optional platform argument initializes new containers to execute and publish as that platform. Platform defaults to that of the builder's host.
func Container(opts ...dagger.ContainerOpts) *dagger.Container {
	client := initClient()
	return client.Container(opts...)
}

// The FunctionCall context that the SDK caller is currently executing in.
//
// If the caller is not currently executing in a function, this will return an error.
func CurrentFunctionCall() *dagger.FunctionCall {
	client := initClient()
	return client.CurrentFunctionCall()
}

// The module currently being served in the session, if any.
func CurrentModule() *dagger.CurrentModule {
	client := initClient()
	return client.CurrentModule()
}

// The TypeDef representations of the objects currently being served in the session.
func CurrentTypeDefs(ctx context.Context) ([]dagger.TypeDef, error) {
	client := initClient()
	return client.CurrentTypeDefs(ctx)
}

// The default platform of the engine.
func DefaultPlatform(ctx context.Context) (dagger.Platform, error) {
	client := initClient()
	return client.DefaultPlatform(ctx)
}

// Creates an empty directory.
func Directory() *dagger.Directory {
	client := initClient()
	return client.Directory()
}

// The Dagger engine container configuration and state
func Engine() *dagger.Engine {
	client := initClient()
	return client.Engine()
}

// Create a new error.
func Error(message string) *dagger.Error {
	client := initClient()
	return client.Error(message)
}

// Creates a function.
func Function(name string, returnType *dagger.TypeDef) *dagger.Function {
	client := initClient()
	return client.Function(name, returnType)
}

// Create a code generation result, given a directory containing the generated code.
func GeneratedCode(code *dagger.Directory) *dagger.GeneratedCode {
	client := initClient()
	return client.GeneratedCode(code)
}

// Queries a Git repository.
func Git(url string, opts ...dagger.GitOpts) *dagger.GitRepository {
	client := initClient()
	return client.Git(url, opts...)
}

// Queries the host environment.
func Host() *dagger.Host {
	client := initClient()
	return client.Host()
}

// Returns a file containing an http remote url content.
func HTTP(url string, opts ...dagger.HTTPOpts) *dagger.File {
	client := initClient()
	return client.HTTP(url, opts...)
}

// Load a CacheVolume from its ID.
func LoadCacheVolumeFromID(id dagger.CacheVolumeID) *dagger.CacheVolume {
	client := initClient()
	return client.LoadCacheVolumeFromID(id)
}

// Load a Container from its ID.
func LoadContainerFromID(id dagger.ContainerID) *dagger.Container {
	client := initClient()
	return client.LoadContainerFromID(id)
}

// Load a CurrentModule from its ID.
func LoadCurrentModuleFromID(id dagger.CurrentModuleID) *dagger.CurrentModule {
	client := initClient()
	return client.LoadCurrentModuleFromID(id)
}

// Load a Directory from its ID.
func LoadDirectoryFromID(id dagger.DirectoryID) *dagger.Directory {
	client := initClient()
	return client.LoadDirectoryFromID(id)
}

// Load a EngineCacheEntry from its ID.
func LoadEngineCacheEntryFromID(id dagger.EngineCacheEntryID) *dagger.EngineCacheEntry {
	client := initClient()
	return client.LoadEngineCacheEntryFromID(id)
}

// Load a EngineCacheEntrySet from its ID.
func LoadEngineCacheEntrySetFromID(id dagger.EngineCacheEntrySetID) *dagger.EngineCacheEntrySet {
	client := initClient()
	return client.LoadEngineCacheEntrySetFromID(id)
}

// Load a EngineCache from its ID.
func LoadEngineCacheFromID(id dagger.EngineCacheID) *dagger.EngineCache {
	client := initClient()
	return client.LoadEngineCacheFromID(id)
}

// Load a Engine from its ID.
func LoadEngineFromID(id dagger.EngineID) *dagger.Engine {
	client := initClient()
	return client.LoadEngineFromID(id)
}

// Load a EnumTypeDef from its ID.
func LoadEnumTypeDefFromID(id dagger.EnumTypeDefID) *dagger.EnumTypeDef {
	client := initClient()
	return client.LoadEnumTypeDefFromID(id)
}

// Load a EnumValueTypeDef from its ID.
func LoadEnumValueTypeDefFromID(id dagger.EnumValueTypeDefID) *dagger.EnumValueTypeDef {
	client := initClient()
	return client.LoadEnumValueTypeDefFromID(id)
}

// Load a EnvVariable from its ID.
func LoadEnvVariableFromID(id dagger.EnvVariableID) *dagger.EnvVariable {
	client := initClient()
	return client.LoadEnvVariableFromID(id)
}

// Load a Error from its ID.
func LoadErrorFromID(id dagger.ErrorID) *dagger.Error {
	client := initClient()
	return client.LoadErrorFromID(id)
}

// Load a FieldTypeDef from its ID.
func LoadFieldTypeDefFromID(id dagger.FieldTypeDefID) *dagger.FieldTypeDef {
	client := initClient()
	return client.LoadFieldTypeDefFromID(id)
}

// Load a File from its ID.
func LoadFileFromID(id dagger.FileID) *dagger.File {
	client := initClient()
	return client.LoadFileFromID(id)
}

// Load a FunctionArg from its ID.
func LoadFunctionArgFromID(id dagger.FunctionArgID) *dagger.FunctionArg {
	client := initClient()
	return client.LoadFunctionArgFromID(id)
}

// Load a FunctionCallArgValue from its ID.
func LoadFunctionCallArgValueFromID(id dagger.FunctionCallArgValueID) *dagger.FunctionCallArgValue {
	client := initClient()
	return client.LoadFunctionCallArgValueFromID(id)
}

// Load a FunctionCall from its ID.
func LoadFunctionCallFromID(id dagger.FunctionCallID) *dagger.FunctionCall {
	client := initClient()
	return client.LoadFunctionCallFromID(id)
}

// Load a Function from its ID.
func LoadFunctionFromID(id dagger.FunctionID) *dagger.Function {
	client := initClient()
	return client.LoadFunctionFromID(id)
}

// Load a GeneratedCode from its ID.
func LoadGeneratedCodeFromID(id dagger.GeneratedCodeID) *dagger.GeneratedCode {
	client := initClient()
	return client.LoadGeneratedCodeFromID(id)
}

// Load a GitRef from its ID.
func LoadGitRefFromID(id dagger.GitRefID) *dagger.GitRef {
	client := initClient()
	return client.LoadGitRefFromID(id)
}

// Load a GitRepository from its ID.
func LoadGitRepositoryFromID(id dagger.GitRepositoryID) *dagger.GitRepository {
	client := initClient()
	return client.LoadGitRepositoryFromID(id)
}

// Load a Host from its ID.
func LoadHostFromID(id dagger.HostID) *dagger.Host {
	client := initClient()
	return client.LoadHostFromID(id)
}

// Load a InputTypeDef from its ID.
func LoadInputTypeDefFromID(id dagger.InputTypeDefID) *dagger.InputTypeDef {
	client := initClient()
	return client.LoadInputTypeDefFromID(id)
}

// Load a InterfaceTypeDef from its ID.
func LoadInterfaceTypeDefFromID(id dagger.InterfaceTypeDefID) *dagger.InterfaceTypeDef {
	client := initClient()
	return client.LoadInterfaceTypeDefFromID(id)
}

// Load a Label from its ID.
func LoadLabelFromID(id dagger.LabelID) *dagger.Label {
	client := initClient()
	return client.LoadLabelFromID(id)
}

// Load a ListTypeDef from its ID.
func LoadListTypeDefFromID(id dagger.ListTypeDefID) *dagger.ListTypeDef {
	client := initClient()
	return client.LoadListTypeDefFromID(id)
}

// Load a ModuleConfigClient from its ID.
func LoadModuleConfigClientFromID(id dagger.ModuleConfigClientID) *dagger.ModuleConfigClient {
	client := initClient()
	return client.LoadModuleConfigClientFromID(id)
}

// Load a Module from its ID.
func LoadModuleFromID(id dagger.ModuleID) *dagger.Module {
	client := initClient()
	return client.LoadModuleFromID(id)
}

// Load a ModuleSource from its ID.
func LoadModuleSourceFromID(id dagger.ModuleSourceID) *dagger.ModuleSource {
	client := initClient()
	return client.LoadModuleSourceFromID(id)
}

// Load a ObjectTypeDef from its ID.
func LoadObjectTypeDefFromID(id dagger.ObjectTypeDefID) *dagger.ObjectTypeDef {
	client := initClient()
	return client.LoadObjectTypeDefFromID(id)
}

// Load a Port from its ID.
func LoadPortFromID(id dagger.PortID) *dagger.Port {
	client := initClient()
	return client.LoadPortFromID(id)
}

// Load a SDKConfig from its ID.
func LoadSDKConfigFromID(id dagger.SDKConfigID) *dagger.SDKConfig {
	client := initClient()
	return client.LoadSDKConfigFromID(id)
}

// Load a ScalarTypeDef from its ID.
func LoadScalarTypeDefFromID(id dagger.ScalarTypeDefID) *dagger.ScalarTypeDef {
	client := initClient()
	return client.LoadScalarTypeDefFromID(id)
}

// Load a Secret from its ID.
func LoadSecretFromID(id dagger.SecretID) *dagger.Secret {
	client := initClient()
	return client.LoadSecretFromID(id)
}

// Load a Secret from its Name.
func LoadSecretFromName(name string, opts ...dagger.LoadSecretFromNameOpts) *dagger.Secret {
	client := initClient()
	return client.LoadSecretFromName(name, opts...)
}

// Load a Service from its ID.
func LoadServiceFromID(id dagger.ServiceID) *dagger.Service {
	client := initClient()
	return client.LoadServiceFromID(id)
}

// Load a Socket from its ID.
func LoadSocketFromID(id dagger.SocketID) *dagger.Socket {
	client := initClient()
	return client.LoadSocketFromID(id)
}

// Load a SourceMap from its ID.
func LoadSourceMapFromID(id dagger.SourceMapID) *dagger.SourceMap {
	client := initClient()
	return client.LoadSourceMapFromID(id)
}

// Load a Terminal from its ID.
func LoadTerminalFromID(id dagger.TerminalID) *dagger.Terminal {
	client := initClient()
	return client.LoadTerminalFromID(id)
}

// Load a TypeDef from its ID.
func LoadTypeDefFromID(id dagger.TypeDefID) *dagger.TypeDef {
	client := initClient()
	return client.LoadTypeDefFromID(id)
}

// Create a new module.
func Module() *dagger.Module {
	client := initClient()
	return client.Module()
}

// Create a new module source instance from a source ref string
func ModuleSource(refString string, opts ...dagger.ModuleSourceOpts) *dagger.ModuleSource {
	client := initClient()
	return client.ModuleSource(refString, opts...)
}

// Creates a new secret.
func Secret(uri string) *dagger.Secret {
	client := initClient()
	return client.Secret(uri)
}

// Sets a secret given a user defined name to its plaintext and returns the secret.
//
// The plaintext value is limited to a size of 128000 bytes.
func SetSecret(name string, plaintext string) *dagger.Secret {
	client := initClient()
	return client.SetSecret(name, plaintext)
}

// Creates source map metadata.
func SourceMap(filename string, line int, column int) *dagger.SourceMap {
	client := initClient()
	return client.SourceMap(filename, line, column)
}

// Create a new TypeDef.
func TypeDef() *dagger.TypeDef {
	client := initClient()
	return client.TypeDef()
}

// Get the current Dagger Engine version.
func Version(ctx context.Context) (string, error) {
	client := initClient()
	return client.Version(ctx)
}
