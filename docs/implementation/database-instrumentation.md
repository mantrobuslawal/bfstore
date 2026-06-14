# Database Instrumentation

This document explains how bfstore instruments catalog database access with OpenTelemetry.

The current implementation instruments MySQL through Go's standard `database/sql` package using `otelsql`.

## Goal

The goal is to make catalog traces show database work underneath incoming gRPC requests.

Before database instrumentation, a trace may show only:

```text
/bfstore.catalog.v1.CatalogService/ListProducts
```

After database instrumentation, the trace should show:

```text
/bfstore.catalog.v1.CatalogService/ListProducts
  -> database/sql span
  -> database/sql span
```

This helps answer:

```text
Did the request spend time in the database?
Which database operations were slow?
Did a database operation error?
Is the repository call part of the same request trace?
```

## Current flow

```text
grpcurl request
  -> catalog-service gRPC server
  -> otelgrpc gRPC span
  -> catalog handler
  -> catalog service
  -> catalog repository
  -> database/sql context-aware query
  -> otelsql instrumented driver
  -> go-sql-driver/mysql
  -> MySQL
```

Telemetry export flow:

```text
catalog-service
  -> OpenTelemetry Collector
  -> Jaeger
```

## Implementation location

Database instrumentation is configured in:

```text
services/catalog-service/internal/database/mysql.go
```

This is the right first location because the database package owns opening the database connection pool.

Instrumentation should not be scattered across every repository method.

## Why instrument at the connection boundary?

The database connection boundary is stable.

Repository methods should remain focused on domain persistence logic:

```text
build query
execute query with context
scan rows
return domain model
```

The instrumentation wrapper belongs underneath `database/sql`, so repository code does not need to manually create spans for each query.

Practical rule:

```text
Instrument stable boundaries first.
Avoid sprinkling tracing code through business logic.
```

## Driver registration

The service uses `otelsql.Register("mysql")` to create an instrumented SQL driver name.

The underlying MySQL driver is registered by this blank import:

```go
_ "github.com/go-sql-driver/mysql"
```

Then `otelsql` wraps it.

Conceptual flow:

```text
mysql
  -> go-sql-driver/mysql

otelsql.Register("mysql")
  -> creates instrumented wrapper driver name

sql.Open(instrumentedDriverName, dsn)
  -> opens an instrumented *sql.DB connection pool
```

## Register once

SQL drivers are registered globally inside the Go process.

The instrumented driver should therefore be registered once per process.

Recommended pattern:

```go
var (
    registerInstrumentedDriverOnce sync.Once
    registerInstrumentedDriverName string
    registerInstrumentedDriverErr  error
)
```

Then:

```go
func instrumentedMySQLDriver() (string, error) {
    registerInstrumentedDriverOnce.Do(func() {
        registerInstrumentedDriverName, registerInstrumentedDriverErr = otelsql.Register(baseMySQLDriverName)
    })

    if registerInstrumentedDriverErr != nil {
        return "", registerInstrumentedDriverErr
    }

    return registerInstrumentedDriverName, nil
}
```

This prevents duplicate registration if `Open` is called multiple times.

## Context propagation

Database spans attach to the current request trace only when the request context reaches the database call.

Good:

```go
rows, err := db.QueryContext(ctx, query, args...)
row := db.QueryRowContext(ctx, query, args...)
result, err := db.ExecContext(ctx, query, args...)
```

Avoid in request paths:

```go
db.Query(query, args...)
db.QueryRow(query, args...)
db.Exec(query, args...)
```

The required call chain is:

```text
gRPC handler receives ctx
  -> application service receives ctx
  -> repository method receives ctx
  -> QueryContext / QueryRowContext / ExecContext receives ctx
```

If that chain is broken, database spans may appear as separate traces or may not attach to the gRPC request span.

## Testing

Database instrumentation unit tests should not need a live MySQL server.

Unit tests should verify:

```text
instrumentedMySQLDriver returns a non-empty driver name
instrumentedMySQLDriver can be called repeatedly
sql.Open recognises the instrumented driver name
```

Avoid `PingContext` in unit tests because it requires a real database.

Run:

```bash
go test ./services/catalog-service/internal/database -v
```

Integration or smoke tests can verify real database connectivity.

## Smoke test

Start observability:

```bash
make observability-up
```

Run the catalog service with telemetry enabled:

```bash
cd services/catalog-service

TELEMETRY_ENABLED=true \
OTEL_EXPORTER_OTLP_ENDPOINT=localhost:4317 \
OTEL_EXPORTER_OTLP_INSECURE=true \
GRPC_REFLECTION_ENABLED=true \
go run ./cmd/catalog-service
```

Send a request:

```bash
grpcurl -plaintext \
  -H 'x-correlation-id: local-dev-db-otel-123' \
  -d '{"page":{"page_size":5}}' \
  localhost:50051 \
  bfstore.catalog.v1.CatalogService/ListProducts
```

Open Jaeger:

```text
http://localhost:16686
```

Search for:

```text
catalog-service
```

Expected result:

```text
the gRPC ListProducts span has database child spans
```

## Troubleshooting

### Jaeger shows gRPC spans but no database spans

Check that repository code uses:

```go
QueryContext
QueryRowContext
ExecContext
```

Also confirm the catalog service was restarted after the `mysql.go` instrumentation change.

### Database spans appear but not under the gRPC span

This usually means the context was lost between the handler and repository.

Check the method signatures and make sure `ctx context.Context` is passed all the way down.

### Unit tests fail with duplicate driver registration

Make sure `otelsql.Register("mysql")` is protected by `sync.Once`.

### Unit tests try to connect to MySQL

Unit tests should use `sql.Open`, not `PingContext`.

`sql.Open` validates the driver name and creates a pool handle. It does not establish a network connection by itself.

## Security and privacy note

Telemetry is operational data, but it can still leak sensitive information if handled carelessly.

Avoid recording:

```text
customer names
email addresses
payment data
full addresses
raw tokens
full SQL with embedded values
```

Prefer:

```text
operation name
database system
database name
duration
error status
sanitised statement shape if needed later
```

## Current milestone

The catalog service now demonstrates:

```text
gRPC request tracing
database/sql tracing
trace context propagation from handler to repository
OpenTelemetry Collector export
Jaeger trace visualisation
```

This moves observability from:

```text
the service received a request
```

to:

```text
the request did database work, and we can see where time was spent
```

Keep it boring where production matters.
