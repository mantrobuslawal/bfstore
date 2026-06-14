# Database Metrics

This document explains the database connection pool metrics added to bfstore.

The current implementation exposes Go `database/sql` pool statistics through OpenTelemetry metrics.

## Goal

Database traces show what happened inside a single request.

Database metrics show how the database connection pool behaves over time.

Together:

```text
traces
  explain individual request behaviour

metrics
  explain service health and resource pressure over time
```

## Why database pool metrics matter

The catalog service uses MySQL through Go's `database/sql`.

In Go, `*sql.DB` is a connection pool. It manages database connections for the application.

That means the pool can become a bottleneck even when application code is correct.

Common production symptoms include:

```text
slow requests
requests waiting for connections
too many open connections
too many idle connections
connection churn
database saturation
```

Database pool metrics help detect those symptoms earlier.

## Implementation location

The reusable package lives at:

```text
pkg/platform/dbmetrics
```

Catalog service wiring happens after the database is opened:

```text
services/catalog-service/cmd/catalog-service/main.go
```

The database connection itself is opened in:

```text
services/catalog-service/internal/database/mysql.go
```

## Metric source

The metrics are read from:

```go
db.Stats()
```

The package registers OpenTelemetry observable instruments that observe the current connection pool state during metric collection.

## Current metric names

```text
db.client.connections.max
db.client.connections.open
db.client.connections.in_use
db.client.connections.idle
db.client.connections.wait_count
db.client.connections.wait_duration
db.client.connections.max_idle_closed
db.client.connections.max_idle_time_closed
db.client.connections.max_lifetime_closed
```

## Metric meanings

### `db.client.connections.max`

The configured maximum number of open connections.

This comes from:

```go
stats.MaxOpenConnections
```

If this is `0`, Go treats the pool as having no explicit maximum.

### `db.client.connections.open`

The number of established connections, including in-use and idle connections.

This comes from:

```go
stats.OpenConnections
```

### `db.client.connections.in_use`

The number of connections currently being used.

This comes from:

```go
stats.InUse
```

If this stays high, queries may be slow or traffic may be high.

### `db.client.connections.idle`

The number of idle connections available for reuse.

This comes from:

```go
stats.Idle
```

### `db.client.connections.wait_count`

The total number of times callers waited for a connection.

This comes from:

```go
stats.WaitCount
```

A rising wait count is a strong signal that callers are waiting for pool capacity.

### `db.client.connections.wait_duration`

The total amount of time callers spent waiting for a connection, reported in milliseconds.

This comes from:

```go
stats.WaitDuration.Milliseconds()
```

If this rises quickly, the database pool may be too small, the database may be slow, or queries may be holding connections too long.

### `db.client.connections.max_idle_closed`

The total number of connections closed because the max idle connection limit was exceeded.

This comes from:

```go
stats.MaxIdleClosed
```

### `db.client.connections.max_idle_time_closed`

The total number of connections closed because they were idle for longer than the configured idle time.

This comes from:

```go
stats.MaxIdleTimeClosed
```

### `db.client.connections.max_lifetime_closed`

The total number of connections closed because they exceeded the configured connection lifetime.

This comes from:

```go
stats.MaxLifetimeClosed
```

## Attributes

Metrics include attributes such as:

```text
db.system=mysql
db.name=bfstore_catalog
```

These help distinguish metrics when there are multiple services and databases.

## How catalog-service wires metrics

After opening the database:

```go
db, err := database.Open(cfg.Database)
if err != nil {
    logger.Error("failed to open database", "error", err)
    os.Exit(1)
}
```

Register metrics:

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

## Local verification

Start observability services:

```bash
make observability-up
```

Run catalog-service with telemetry enabled:

```bash
make catalog-run-telemetry
```

Send a request:

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

Expected metric names:

```text
db.client.connections.open
db.client.connections.in_use
db.client.connections.idle
db.client.connections.wait_count
db.client.connections.wait_duration
```

## Testing

Run:

```bash
go test ./pkg/platform/dbmetrics -v
go test ./...
```

Unit tests should not require MySQL to be running.

They should verify the metric registration behaviour without calling `PingContext`.

## Operational interpretation

### Healthy local development pattern

In local development, you may see:

```text
low open connection count
low in-use count
some idle connections
zero or low wait count
zero or low wait duration
```

That is expected.

### Possible pool pressure

Warning signs include:

```text
open connections close to max connections
in-use connections close to open connections
wait count increasing
wait duration increasing
```

This means requests are waiting for database connections.

Possible causes:

```text
database queries are slow
connection pool is too small
traffic has increased
connections are held too long
database server is saturated
```

### Connection churn

High values for closed connection metrics may indicate pool tuning issues:

```text
max_idle_closed
max_idle_time_closed
max_lifetime_closed
```

Some churn is normal. Excessive churn can add latency and load.

## Relationship to traces

Database traces and metrics answer different questions.

Database traces:

```text
Which SQL work happened inside this request?
How long did this query take?
Did this query error?
```

Database metrics:

```text
How many DB connections are open?
Are requests waiting for connections?
Is the pool under pressure?
Is connection churn increasing?
```

Use both together.

## Current milestone

The catalog service now has:

```text
gRPC request spans
database child spans
database connection pool metrics
Collector debug metric verification
Jaeger trace visualisation
```

The next metrics step is to add a real metrics backend such as Prometheus.

Keep it boring where production matters.
