# Platform DB Metrics

The `dbmetrics` package provides reusable OpenTelemetry metrics for Go `database/sql` connection pools.

It is designed for bfstore services that use `*sql.DB`, such as the catalog service.

## Package location

```text
pkg/platform/dbmetrics
```

Expected package structure:

```text
pkg/platform/dbmetrics/
├── README.md
├── dbmetrics.go
└── dbmetrics_test.go
```

## Purpose

Go's `*sql.DB` is a connection pool, not a single database connection.

That pool can become a production bottleneck if:

```text
too many requests need database connections
queries are slow
the pool is too small
connections are churned too often
the database is saturated
```

This package exposes database pool state as OpenTelemetry metrics so operators can understand the database access pattern over time.

## Mental model

Traces answer:

```text
What happened to this request?
```

Database spans answer:

```text
Which SQL operations happened inside this request?
```

Database pool metrics answer:

```text
How healthy is the database connection pool over time?
```

## Current metric source

Metrics are read from:

```go
db.Stats()
```

That returns Go `database/sql` pool statistics.

The package registers OpenTelemetry observable instruments that read those stats during metric collection.

## Metrics

### `db.client.connections.max`

Maximum number of open connections configured for the database pool.

Maps to:

```go
stats.MaxOpenConnections
```

Useful for understanding the configured pool limit.

### `db.client.connections.open`

Number of established database connections, both in use and idle.

Maps to:

```go
stats.OpenConnections
```

If this regularly sits close to `db.client.connections.max`, the pool may be under pressure.

### `db.client.connections.in_use`

Number of database connections currently in use.

Maps to:

```go
stats.InUse
```

High values during load are expected. Constantly high values may indicate slow queries, connection starvation, or insufficient pool capacity.

### `db.client.connections.idle`

Number of idle database connections.

Maps to:

```go
stats.Idle
```

Idle connections are available for reuse.

### `db.client.connections.wait_count`

Total number of times callers waited for a database connection.

Maps to:

```go
stats.WaitCount
```

A rising value means callers are waiting because the pool has no immediately available connection.

### `db.client.connections.wait_duration`

Total time callers spent waiting for a database connection, reported in milliseconds.

Maps to:

```go
stats.WaitDuration.Milliseconds()
```

Rising wait duration can indicate database pool pressure.

### `db.client.connections.max_idle_closed`

Total number of connections closed because the max idle connections limit was exceeded.

Maps to:

```go
stats.MaxIdleClosed
```

### `db.client.connections.max_idle_time_closed`

Total number of connections closed because they were idle for too long.

Maps to:

```go
stats.MaxIdleTimeClosed
```

### `db.client.connections.max_lifetime_closed`

Total number of connections closed because they exceeded the configured connection lifetime.

Maps to:

```go
stats.MaxLifetimeClosed
```

## Attributes

The package attaches database identity attributes such as:

```text
db.system=mysql
db.name=bfstore_catalog
```

These attributes make it easier to distinguish metrics when multiple services or databases exist.

## Usage

After opening a database connection pool:

```go
db, err := database.Open(cfg.Database)
if err != nil {
    logger.Error("failed to open database", "error", err)
    os.Exit(1)
}
```

Register database metrics:

```go
if err := dbmetrics.Register(db, dbmetrics.Config{
    MeterName: "github.com/mantrobuslawal/bfstore/services/catalog-service",
    DBSystem:  "mysql",
    DBName:    cfg.Database.Name,
}); err != nil {
    logger.Error("failed to register database metrics", "error", err)
    os.Exit(1)
}
```

## Why this lives in `pkg/platform`

Connection pool metrics are platform-level runtime plumbing.

The same pattern can later be reused by:

```text
inventory-service
basket-service
order-service
payment-service
shipping-service
notification-service
```

Each service can provide its own database system and database name.

## Testing

Run package tests:

```bash
go test ./pkg/platform/dbmetrics -v
```

The unit tests should not require a live database.

They should verify:

```text
nil *sql.DB is rejected
a normal *sql.DB handle can register metrics
defaults are applied
```

Avoid `PingContext` in these unit tests because that would require a real MySQL instance.

## Verification

To verify locally:

```bash
make observability-up
make catalog-run-telemetry
```

Send catalog requests:

```bash
grpcurl -plaintext \
  -H 'x-correlation-id: local-dev-dbmetrics-123' \
  -d '{"page":{"page_size":5}}' \
  localhost:50051 \
  bfstore.catalog.v1.CatalogService/ListProducts
```

Watch Collector logs:

```bash
docker compose logs -f otel-collector
```

Look for metrics such as:

```text
db.client.connections.open
db.client.connections.in_use
db.client.connections.idle
db.client.connections.wait_count
db.client.connections.wait_duration
```

## Troubleshooting

### Metrics do not appear in Collector logs

Check that:

```text
TELEMETRY_ENABLED=true
MetricsEnabled is true
the Collector metrics pipeline includes the debug exporter
dbmetrics.Register is called after database.Open
the service was restarted after wiring dbmetrics
```

### Metrics appear but do not change

This may be normal under low local traffic.

Send repeated requests or run a small load test later to create more visible pool activity.

### Unit tests try to connect to MySQL

Unit tests should use `sql.Open`, not `PingContext`.

`sql.Open` creates a handle and validates the driver name. It does not establish a database connection by itself.

## Design rule

```text
Trace request paths.
Measure shared runtime resources.
Keep business logic clean.
```

Keep it boring where production matters.
