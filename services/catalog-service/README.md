# Catalog Service

The `catalog-service` owns the product catalog for bfstore.

It provides gRPC APIs for reading catalog information such as products, categories, variants, images, and product attributes.

This service is part of the bfstore platform: a cloud-native ecommerce system for developer-themed homeware.

## Status

Current iteration status:

```text
local production-shaped slice complete
```

The service currently demonstrates:

```text
gRPC API serving
MySQL persistence
service-owned database boundary
unit tests
repository tests
local smoke testing
standard gRPC health checks
gRPC reflection for local development
structured request logging
correlation ID propagation
panic recovery
graceful shutdown
OpenTelemetry bootstrap
gRPC request tracing
database/sql tracing
database pool metrics
request metrics
OpenTelemetry Collector integration
Jaeger trace visualisation
Prometheus metric querying
Grafana dashboard provisioning
local traffic generation
```

## Service ownership

The catalog service owns product catalog read models and catalog persistence.

It is responsible for:

```text
products
categories
product variants
product images
catalog attributes
catalog metadata required for browsing
```

It is not responsible for:

```text
basket state
inventory reservation
order placement
payment processing
shipping
notifications
reviews
recommendations
search indexing
```

Those responsibilities belong to separate services or later bfstore slices.

## Runtime architecture

At runtime, the service follows this shape:

```text
configuration
  -> logger
  -> telemetry
  -> database connection pool
  -> readiness checks
  -> repository
  -> service layer
  -> gRPC handlers
  -> gRPC server
  -> health manager
  -> graceful shutdown
```

Request path:

```text
grpc client
  -> gRPC server
  -> recovery interceptor
  -> correlation ID interceptor
  -> request metrics interceptor
  -> logging interceptor
  -> catalog gRPC handler
  -> catalog service
  -> catalog repository
  -> MySQL
```

Telemetry path:

```text
catalog-service
  -> OpenTelemetry Collector
  -> Jaeger for traces
  -> Prometheus for metrics
  -> Grafana for dashboards
```

## Local ports

Typical local ports:

```text
catalog-service gRPC: 50051
MySQL: 3306
OpenTelemetry Collector OTLP gRPC: 4317
OpenTelemetry Collector OTLP HTTP: 4318
OpenTelemetry Collector Prometheus exporter: 9464
Jaeger UI: 16686
Prometheus UI: 9090
Grafana UI: 3000
```

## Configuration

Common local environment variables:

```text
TELEMETRY_ENABLED=true
OTEL_EXPORTER_OTLP_ENDPOINT=localhost:4317
OTEL_EXPORTER_OTLP_INSECURE=true
GRPC_REFLECTION_ENABLED=true
```

Database configuration is provided through the service config package and local environment setup.

When running inside Docker Compose, the OTLP endpoint should normally use the Compose service name:

```text
otel-collector:4317
```

When running from the host with `go run`, use:

```text
localhost:4317
```

## Running local dependencies

Start MySQL and observability services:

```bash
make observability-up
```

Or directly:

```bash
docker compose up -d mysql otel-collector jaeger prometheus grafana
```

Check status:

```bash
docker compose ps
```

## Running the service locally

From the service directory:

```bash
cd services/catalog-service

TELEMETRY_ENABLED=true \
OTEL_EXPORTER_OTLP_ENDPOINT=localhost:4317 \
OTEL_EXPORTER_OTLP_INSECURE=true \
GRPC_REFLECTION_ENABLED=true \
go run ./cmd/catalog-service
```

## gRPC API

Reflection can be enabled locally with:

```text
GRPC_REFLECTION_ENABLED=true
```

List services:

```bash
grpcurl -plaintext localhost:50051 list
```

Expected service:

```text
bfstore.catalog.v1.CatalogService
```

Call `ListProducts`:

```bash
grpcurl -plaintext \
  -H 'x-correlation-id: local-dev-readme-001' \
  -d '{"page":{"page_size":5}}' \
  localhost:50051 \
  bfstore.catalog.v1.CatalogService/ListProducts
```

## Health checks

Check whole-server health:

```bash
grpcurl -plaintext localhost:50051 grpc.health.v1.Health/Check
```

Expected response:

```json
{
  "status": "SERVING"
}
```

Check catalog service health:

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

## Database

The service uses MySQL as its source of truth.

Current local database assets live under the repo database structure, including:

```text
db/mysql-init/
db/catalog/migrations/
db/catalog/seeds/
```

The current catalog-service iteration uses SQL migration files directly.

When the basket service begins, bfstore will introduce `golang-migrate` as the migration standard for service-owned database migrations.

After the fuller commerce slice is complete, including order placement and payment, bfstore will revisit whether to move from `golang-migrate` to Flyway for broader migration workflow consistency.

## Observability

The catalog service emits:

```text
traces
request metrics
database spans
database pool metrics
structured logs
correlation IDs
```

Local observability tools:

```text
Jaeger:     http://localhost:16686
Prometheus: http://localhost:9090
Grafana:    http://localhost:3000
```

## Traces

Jaeger should show traces for gRPC requests.

Expected trace shape:

```text
/bfstore.catalog.v1.CatalogService/ListProducts
  -> database/sql span
```

Search for:

```text
catalog-service
```

in Jaeger.

## Request metrics

Request metrics are emitted by the reusable gRPC request metrics interceptor.

Prometheus metric names:

```text
bfstore_rpc_server_requests_total
bfstore_rpc_server_request_duration_count
bfstore_rpc_server_request_duration_sum
bfstore_rpc_server_request_duration_bucket
```

Useful queries:

```promql
sum(rate(bfstore_rpc_server_requests_total[5m]))
```

```promql
sum by (rpc_method) (rate(bfstore_rpc_server_requests_total[5m]))
```

```promql
sum(rate(bfstore_rpc_server_requests_total{rpc_grpc_status_code!="OK"}[5m]))
```

```promql
sum(rate(bfstore_rpc_server_request_duration_sum[5m]))
/
clamp_min(sum(rate(bfstore_rpc_server_request_duration_count[5m])), 1)
```

## Database metrics

Database pool metrics are emitted from `db.Stats()`.

Prometheus metric names:

```text
db_client_connections_open
db_client_connections_in_use
db_client_connections_idle
db_client_connections_wait_count
db_client_connections_wait_duration
```

Useful queries:

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

## Grafana dashboard

Expected Grafana folder:

```text
bfstore
```

Expected dashboard:

```text
Catalog Service Overview
```

Dashboard file:

```text
deployments/local/grafana/dashboards/catalog-service-overview.json
```

Dashboard sections:

```text
Catalog request overview
Catalog database pool overview
```

## Local traffic generation

Use the local load script to generate repeatable traffic:

```bash
REQUESTS=100 SLEEP_SECONDS=0.05 ./scripts/local/catalog-load.sh
```

This should make request metrics and database pool metrics move in Prometheus and Grafana.

## Testing

Run catalog-service tests:

```bash
go test ./services/catalog-service/... -v
```

Run related platform package tests:

```bash
go test ./pkg/platform/grpc/requestmetrics -v
go test ./pkg/platform/dbmetrics -v
go test ./pkg/platform/telemetry -v
```

Run all tests:

```bash
go test ./...
```

## Smoke checklist

The full local smoke checklist is documented here:

```text
docs/implementation/catalog-service-smoke-checklist.md
```

Use this checklist before calling the catalog-service iteration complete.

## Graceful shutdown

The service should handle `SIGINT` and `SIGTERM`.

Expected shutdown flow:

```text
receive shutdown signal
mark gRPC health NOT_SERVING
gracefully stop gRPC server
allow in-flight requests to finish
close database connection pool
shutdown telemetry providers
exit cleanly
```

## Troubleshooting

### Service does not start

Check:

```text
database is running
database credentials are correct
catalog schema exists
required environment variables are set
```

### Health is NOT_SERVING

Check:

```text
database readiness check
health manager registration
startup logs
```

### grpcurl cannot list services

Check:

```text
GRPC_REFLECTION_ENABLED=true
service restarted after changing config
correct port is used
```

### Jaeger has no traces

Check:

```text
TELEMETRY_ENABLED=true
OTEL_EXPORTER_OTLP_ENDPOINT is correct
otel-collector is running
jaeger is running
```

### Prometheus has no metrics

Check:

```text
Prometheus target otel-collector is UP
Collector metrics pipeline exports to Prometheus
traffic has been sent after service startup
```

### Grafana has no data

Check Prometheus first.

If Prometheus has no data, Grafana will have no data.

### Grafana provisioning conflicts

A stale Grafana data volume can conflict with provisioned datasources or dashboards.

For local development only:

```bash
docker compose down
docker volume ls
docker volume rm <project>_bfstore-grafana-data
docker compose up -d grafana
```

Use the actual volume name shown by `docker volume ls`.

## Known limitations

This iteration intentionally does not yet include:

```text
catalog-service Docker Compose runtime service
CI workflow
golang-migrate migration runner
Flyway migration workflow
production Kubernetes manifests
authentication / authorisation
rate limiting
full ecommerce checkout flow
basket service integration
inventory reservation
order placement
payment integration
```

Some of these are planned later. Optional catalog-service extras will be revisited after the basket service is complete.

## Planned follow-ups

After basket service is complete, revisit:

```text
Dockerise catalog-service inside Docker Compose
Add CI workflow
Polish request metrics dashboard docs
Add ADR for local observability stack
```

When basket service starts:

```text
introduce golang-migrate
update database migration docs
standardise service-owned migration workflow
```

After the fuller commerce slice is complete:

```text
revisit Flyway as a broader migration workflow
```

## Completion statement

The catalog-service iteration is complete when another engineer can:

```text
start dependencies
run the service
call the gRPC API
verify health
run tests
generate traffic
inspect traces
query metrics
view dashboards
shut the service down cleanly
troubleshoot common local issues
```

## Practical rule

```text
A service is not complete because it runs on your machine.
A service is complete when its behaviour can be repeated, observed, and explained.
```

