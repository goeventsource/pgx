# pgx

**PostgreSQL plugin** for [goeventsource](https://github.com/goeventsource/goeventsource): production-style implementations of `goeventsource.Store`, `goeventsource.Repository`, and `goeventsource.Snapshotter` using [jackc/pgx](https://github.com/jackc/pgx) and a `*pgxpool.Pool`.

## Install

```bash
go get github.com/goeventsource/pgx@latest
```

The module path and root **package name** are both **`pgx`**. The driver’s root import `github.com/jackc/pgx/v5` is aliased inside this module (for example `jackc`) so it does not clash with this package name.

```go
import (
	"github.com/goeventsource/goeventsource"
	"github.com/goeventsource/pgx"
)

store := pgx.NewStore[goeventsource.ID](pool, codec, tableName, primaryKeyName, opts...)
```

## Role in the ecosystem

| Layer | Responsibility |
|-------|----------------|
| **goeventsource** | Domain model and interfaces |
| **pgx** (this module) | SQL persistence + transactions aligned with those interfaces |
| **Your service** | Schema migrations, pool lifecycle, domain codecs |

You keep aggregates and events in the core style; this module only **stores and loads** them.

## Packages

| Import | Package | Purpose |
|--------|---------|---------|
| `github.com/goeventsource/pgx` | **`pgx`** | `NewStore`, `NewRepository`, `NewSnapshotter`, `WithSnapshotterOpt`, `WithProjectorOpt`, context transaction helpers (`WithValueTx`, `ValueTx`) |
| `github.com/goeventsource/pgx/pgxtest` | **`pgxtest`** | Testcontainers-backed pool, seeded schema, helpers aligned with production constructors |

## Quick start: production-shaped repository

You must supply:

- A **`pgxpool.Pool`** (your lifecycle).
- A **`goeventsource.DomainEventEncodeDecoder`** (usually a wrapper map of `NewJSONDomainEventEncodeDecoder[T]` per event).
- Table name and **primary key column name** used as stream id in your schema.
- A **`FactoryFunc`** to rebuild the aggregate root when reading.

```go
factory := func(id uuid.UUID, ver goeventsource.Version) *MyRoot {
	return &MyRoot{BaseRoot: goeventsource.NewBase(id, MyAggregateName, ver)}
}

codec := goeventsource.NewDomainEventEncodeDecoderWrapper(map[goeventsource.DomainEventName]goeventsource.DomainEventEncodeDecoder{
	MyEvent{}.DomainEventName(): goeventsource.NewJSONDomainEventEncodeDecoder[MyEvent](),
})

store := pgx.NewStore[uuid.UUID](pool, codec, "events", "stream_id")
repo := pgx.NewRepository(pool, store, factory,
	pgx.WithSnapshotterOpt(snap),
	pgx.WithProjectorOpt(proj),
)
```

Align `INSERT`/`SELECT` column sets with your migrations—the store uses parameterized SQL built from the table and PK names you pass in.

## Tests and demos: `pgxtest`

For integration tests or a local demo without hand-rolling Docker:

```go
import "github.com/goeventsource/pgx/pgxtest"

db := pgxtest.NewUniqueSeededDatabaseConnection(t) // *testing.T
// or, for a long-lived process:
// pool, cleanup, err := pgxtest.NewDemoPool(ctx)

cfg := pgxtest.NewRepositoryConfig(db, factory)
cfg.StoreConfig.Codec = codec // set like in example-banking/cmd/main.go
repo, _ := pgxtest.NewRepository(cfg)
```

See [`../example-banking`](../example-banking/README.md) for a full wiring including snapshot strategy and HTTP.

## Optional: observability plugin

Wrap the concrete `Store` or `Repository` with [`github.com/goeventsource/opentelemetry`](../opentelemetry/README.md) **outside** your domain so spans cover the storage boundary:

```go
import otes "github.com/goeventsource/opentelemetry"

repo = otes.NewRepository(repo, "payments-api")
```

## Running tests in this repo

Integration tests use **testcontainers** (Docker required):

```bash
go test ./...
```

Use `go test -short ./...` if you add short-mode skips in your own fork.

## Unpublished core module

Use `replace` for `github.com/goeventsource/goeventsource` or the [`new_org/`](../README.md) workspace until tagged releases exist.
