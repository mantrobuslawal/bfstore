# Local Observability

This document explains how to run and verify the local bfstore observability stack.

## Current flow

```text
catalog-service
  -> OpenTelemetry SDK
  -> otelgrpc gRPC server instrumentation
  -> otelsql database/sql instrumentation
  -> dbmetrics database pool metrics
  -> OTLP exporter
  -> OpenTelemetry Collector
  -> Jaeger for traces
  -> Prometheus for metrics
  -> Grafana for dashboards
```

## Local UIs

```text
Jaeger:     http://localhost:16686
Prometheus: http://localhost:9090
Grafana:    http://localhost:3000
```

Grafana credentials:

```text
username: admin
password: admin
```

## Key config files

```text
deployments/local/otel-collector/config.yaml
deployments/local/prometheus/prometheus.yml
deployments/local/grafana/provisioning/datasources/prometheus.yml
deployments/local/grafana/provisioning/dashboards/dashboards.yml
deployments/local/grafana/dashboards/catalog-db-pool-overview.json
```

## Start the stack

```bash
docker compose up -d otel-collector jaeger prometheus grafana
```

Or:

```bash
make observability-up
```

## Run catalog-service with telemetry

```bash
cd services/catalog-service

TELEMETRY_ENABLED=true \
OTEL_EXPORTER_OTLP_ENDPOINT=localhost:4317 \
OTEL_EXPORTER_OTLP_INSECURE=true \
GRPC_REFLECTION_ENABLED=true \
go run ./cmd/catalog-service
```

## Send a request

```bash
grpcurl -plaintext \
  -H 'x-correlation-id: local-dev-grafana-123' \
  -d '{"page":{"page_size":5}}' \
  localhost:50051 \
  bfstore.catalog.v1.CatalogService/ListProducts
```

## Verify Jaeger

Open:

```text
http://localhost:16686
```

Expected trace shape:

```text
/bfstore.catalog.v1.CatalogService/ListProducts
  -> database/sql span
```

## Verify Prometheus

Open:

```text
http://localhost:9090
```

Useful query:

```promql
db_client_connections_open
```

## Verify Grafana

Open:

```text
http://localhost:3000
```

Expected folder:

```text
bfstore
```

Expected dashboard:

```text
Catalog DB Pool Overview
```

## Suggested Make targets

```makefile
.PHONY: observability-up
observability-up:
	docker compose -f $(COMPOSE_FILE) up -d otel-collector jaeger prometheus grafana

.PHONY: observability-logs
observability-logs:
	docker compose -f $(COMPOSE_FILE) logs -f otel-collector jaeger prometheus grafana

.PHONY: grafana-up
grafana-up:
	docker compose -f $(COMPOSE_FILE) up -d grafana

.PHONY: grafana-logs
grafana-logs:
	docker compose -f $(COMPOSE_FILE) logs -f grafana
```

## Troubleshooting

### Grafana dashboard does not appear

Check:

```bash
docker compose logs -f grafana
```

Then restart:

```bash
docker compose restart grafana
```

### Grafana panels show no data

Check Prometheus first:

```promql
db_client_connections_open
```

If Prometheus has no data, Grafana will have no data.

## Next step

The next useful slice is either:

```text
add a small load test to make metrics visibly move
```

or:

```text
add service-level request metrics
```

Keep it boring where production matters.
