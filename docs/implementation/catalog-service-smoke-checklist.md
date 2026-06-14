# Catalog Service Smoke Checklist

This checklist verifies that the bfstore catalog service works end-to-end in local development.

It covers:

```text
database
gRPC API
health checks
telemetry
traces
metrics
dashboards
load script
tests
graceful shutdown
```

The goal is not to prove production readiness. The goal is to prove that the local catalog-service slice is complete, repeatable, and observable.

## Expected local architecture

```text
catalog-service
  -> MySQL
  -> OpenTelemetry Collector
  -> Jaeger
  -> Prometheus
  -> Grafana
```

## Local UIs

```text
Jaeger:     http://localhost:16686
Prometheus: http://localhost:9090
Grafana:    http://localhost:3000
```

## Prerequisites

Install or confirm:

```text
Go 1.25
Docker
Docker Compose
grpcurl
make
```

Optional but useful:

```text
migrate CLI later when golang-migrate is introduced
```

## 1. Start local dependencies

Start MySQL and the local observability stack:

```bash
make observability-up
```

Or directly:

```bash
docker compose up -d mysql otel-collector jaeger prometheus grafana
```

Check containers:

```bash
docker compose ps
```

Expected containers include:

```text
bfstore-mysql
bfstore-otel-collector
bfstore-jaeger
bfstore-prometheus
bfstore-grafana
```

## 2. Confirm MySQL is available

Check MySQL logs:

```bash
docker compose logs -f mysql
```

The database should start cleanly with the catalog database and service user available.

If this is a fresh environment, confirm the catalog schema and seed data are applied according to the current local database setup.

## 3. Start catalog-service with telemetry

From the service directory:

```bash
cd services/catalog-service

TELEMETRY_ENABLED=true \
OTEL_EXPORTER_OTLP_ENDPOINT=localhost:4317 \
OTEL_EXPORTER_OTLP_INSECURE=true \
GRPC_REFLECTION_ENABLED=true \
go run ./cmd/catalog-service
```

Expected startup behaviour:

```text
configuration loads
telemetry initialises
database connection opens
database readiness check passes
gRPC server starts
health service is registered
reflection is enabled for local development
service is marked SERVING
```

## 4. Check gRPC health

Use `grpcurl`:

```bash
grpcurl -plaintext localhost:50051 grpc.health.v1.Health/Check
```

Expected response:

```json
{
  "status": "SERVING"
}
```

Check catalog-service specific health if required:

```bash
grpcurl -plaintext \
  -d '{"service":"bfstore.catalog.v1.CatalogService"}' \
  localhost:50051 \
  grpc.health.v1.Health/Check
```

Expected response:

```json
{
  "status": "SERVING"
}
```

## 5. List gRPC services

Confirm reflection works:

```bash
grpcurl -plaintext localhost:50051 list
```

Expected services include:

```text
bfstore.catalog.v1.CatalogService
grpc.health.v1.Health
grpc.reflection.v1.ServerReflection
```

## 6. Send a catalog request

Call `ListProducts`:

```bash
grpcurl -plaintext \
  -H 'x-correlation-id: local-dev-smoke-001' \
  -d '{"page":{"page_size":5}}' \
  localhost:50051 \
  bfstore.catalog.v1.CatalogService/ListProducts
```

Expected result:

```text
request succeeds
products are returned
catalog-service logs include the supplied correlation ID
```

## 7. Run local traffic generator

Run the local load script:

```bash
REQUESTS=100 SLEEP_SECONDS=0.05 ./scripts/local/catalog-load.sh
```

Expected result:

```text
requests complete
catalog-service stays healthy
no panic recovery logs
no unexpected database errors
Grafana request metrics move
DB pool metrics show activity
```

## 8. Check Jaeger traces

Open:

```text
http://localhost:16686
```

Search for:

```text
catalog-service
```

Expected trace shape:

```text
/bfstore.catalog.v1.CatalogService/ListProducts
  -> database/sql span
```

A good trace should show:

```text
gRPC server span
database child span
service.name=catalog-service
method/status metadata
```

## 9. Check Prometheus targets

Open:

```text
http://localhost:9090/targets
```

Expected target:

```text
otel-collector
```

Expected state:

```text
UP
```

If the target is down, Prometheus cannot scrape metrics from the Collector.

## 10. Check request metrics in Prometheus

Open:

```text
http://localhost:9090
```

Try:

```promql
bfstore_rpc_server_requests_total
```

Request rate:

```promql
sum(rate(bfstore_rpc_server_requests_total[5m]))
```

Request rate by method:

```promql
sum by (rpc_method) (rate(bfstore_rpc_server_requests_total[5m]))
```

Error rate:

```promql
sum(rate(bfstore_rpc_server_requests_total{rpc_grpc_status_code!="OK"}[5m]))
```

Average request duration:

```promql
sum(rate(bfstore_rpc_server_request_duration_sum[5m]))
/
clamp_min(sum(rate(bfstore_rpc_server_request_duration_count[5m])), 1)
```

p95 request duration if histogram buckets are available:

```promql
histogram_quantile(
  0.95,
  sum(rate(bfstore_rpc_server_request_duration_bucket[5m])) by (le)
)
```

## 11. Check database pool metrics in Prometheus

Try:

```promql
db_client_connections_open
```

```promql
db_client_connections_in_use
```

```promql
db_client_connections_idle
```

```promql
rate(db_client_connections_wait_count[5m])
```

Healthy local behaviour usually looks like:

```text
low open connection count
low in-use connection count
some idle connections
near-zero wait rate
near-zero wait duration
```

## 12. Check Grafana dashboard

Open:

```text
http://localhost:3000
```

Login:

```text
username: admin
password: admin
```

Expected folder:

```text
bfstore
```

Expected dashboard:

```text
Catalog Service Overview
```

Expected dashboard sections:

```text
Catalog request overview
Catalog database pool overview
```

Confirm panels show data after traffic has been sent.

## 13. Run unit and package tests

From the repo root:

```bash
go test ./pkg/platform/grpc/requestmetrics -v
go test ./pkg/platform/dbmetrics -v
go test ./pkg/platform/telemetry -v
go test ./services/catalog-service/... -v
```

Then run all tests:

```bash
go test ./...
```

Expected result:

```text
all tests pass
```

## 14. Check graceful shutdown

With catalog-service running, press:

```text
Ctrl+C
```

Expected shutdown behaviour:

```text
shutdown signal received
service marked NOT_SERVING
gRPC server begins graceful stop
in-flight requests are allowed to complete
database connection pool closes
telemetry providers shut down
process exits cleanly
```

There should be no panic and no forced termination under normal local conditions.

## 15. Troubleshooting

### gRPC health is not SERVING

Check:

```text
database is running
database credentials are correct
catalog readiness check passes
service registered with health manager
```

### grpcurl cannot list services

Check:

```text
GRPC_REFLECTION_ENABLED=true
catalog-service restarted after enabling reflection
correct port is used
```

### Jaeger has no traces

Check:

```text
TELEMETRY_ENABLED=true
OTEL_EXPORTER_OTLP_ENDPOINT=localhost:4317
otel-collector is running
jaeger is running
```

### Prometheus has no metrics

Check:

```text
otel-collector metrics pipeline exports to prometheus
Prometheus target is UP
traffic has been sent after service startup
```

### Grafana panels are empty

Check Prometheus first.

If Prometheus has no data, Grafana will have no data.

### Grafana provisioning behaves strangely

A stale Grafana volume may be holding old state.

For local development only, remove the Grafana volume and restart:

```bash
docker compose down
docker volume ls
docker volume rm <project>_bfstore-grafana-data
docker compose up -d grafana
```

Use the actual volume name shown by `docker volume ls`.

## Completion criteria

The catalog-service iteration is complete when:

```text
local dependencies start
catalog-service starts with telemetry
gRPC health returns SERVING
ListProducts works with grpcurl
load script generates successful traffic
Jaeger shows request traces
Prometheus shows request and DB metrics
Grafana dashboard shows request and DB panels
tests pass
graceful shutdown works
README and implementation docs are up to date
```

## Practical rule

```text
A service is not done when it compiles.
A service is done when another engineer can run it, test it, observe it, and troubleshoot it.
```

Keep it boring where production matters.
