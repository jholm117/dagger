# This file generated by `dagger_codegen`. Please DO NOT EDIT.
defmodule Dagger.EngineCacheEntry do
  @moduledoc "An individual cache entry in a cache entry set"

  alias Dagger.Core.Client
  alias Dagger.Core.QueryBuilder, as: QB

  @derive Dagger.ID

  defstruct [:query_builder, :client]

  @type t() :: %__MODULE__{}

  @doc "Whether the cache entry is actively being used."
  @spec actively_used(t()) :: {:ok, boolean()} | {:error, term()}
  def actively_used(%__MODULE__{} = engine_cache_entry) do
    query_builder =
      engine_cache_entry.query_builder |> QB.select("activelyUsed")

    Client.execute(engine_cache_entry.client, query_builder)
  end

  @doc "The time the cache entry was created, in Unix nanoseconds."
  @spec created_time_unix_nano(t()) :: {:ok, integer()} | {:error, term()}
  def created_time_unix_nano(%__MODULE__{} = engine_cache_entry) do
    query_builder =
      engine_cache_entry.query_builder |> QB.select("createdTimeUnixNano")

    Client.execute(engine_cache_entry.client, query_builder)
  end

  @doc "The description of the cache entry."
  @spec description(t()) :: {:ok, String.t()} | {:error, term()}
  def description(%__MODULE__{} = engine_cache_entry) do
    query_builder =
      engine_cache_entry.query_builder |> QB.select("description")

    Client.execute(engine_cache_entry.client, query_builder)
  end

  @doc "The disk space used by the cache entry."
  @spec disk_space_bytes(t()) :: {:ok, integer()} | {:error, term()}
  def disk_space_bytes(%__MODULE__{} = engine_cache_entry) do
    query_builder =
      engine_cache_entry.query_builder |> QB.select("diskSpaceBytes")

    Client.execute(engine_cache_entry.client, query_builder)
  end

  @doc "A unique identifier for this EngineCacheEntry."
  @spec id(t()) :: {:ok, Dagger.EngineCacheEntryID.t()} | {:error, term()}
  def id(%__MODULE__{} = engine_cache_entry) do
    query_builder =
      engine_cache_entry.query_builder |> QB.select("id")

    Client.execute(engine_cache_entry.client, query_builder)
  end

  @doc "The most recent time the cache entry was used, in Unix nanoseconds."
  @spec most_recent_use_time_unix_nano(t()) :: {:ok, integer()} | {:error, term()}
  def most_recent_use_time_unix_nano(%__MODULE__{} = engine_cache_entry) do
    query_builder =
      engine_cache_entry.query_builder |> QB.select("mostRecentUseTimeUnixNano")

    Client.execute(engine_cache_entry.client, query_builder)
  end
end

defimpl Jason.Encoder, for: Dagger.EngineCacheEntry do
  def encode(engine_cache_entry, opts) do
    {:ok, id} = Dagger.EngineCacheEntry.id(engine_cache_entry)
    Jason.Encode.string(id, opts)
  end
end

defimpl Nestru.Decoder, for: Dagger.EngineCacheEntry do
  def decode_fields_hint(_struct, _context, id) do
    {:ok, Dagger.Client.load_engine_cache_entry_from_id(Dagger.Global.dag(), id)}
  end
end
