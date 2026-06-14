# Prometheus Metrics

This document explains the local Prometheus metrics integration for bfstore.

Current observability flow:

```text
catalog-service
  -> OpenTelemetry Collector
  -> Jaeger for traces
  -> Prometheus for metrics
```

Prometheus is the local metrics backend. It scrapes metrics exposed by the OpenTelemetry Collector.

## Goal

Before Prometheus, database pool metrics were visible in Collector debug logs. That proved metrics were emitted, but logs are not a metrics backend.

Prometheus gives bfstore:

```text
queryable metrics
time-series history
PromQL expressions
target health checks
a future source for Grafana dashboards
```

## Architecture

The service does not expose Prometheus metrics directly.

Preferred flow:

```text
catalog-service
  -> emits OTLP metrics
  -> OpenTelemetry Collector receives OTLP
  -> Collector exposes Prometheus scrape endpoint
  -> Prometheus scrapes Collector
```

Avoid direct service-to-Prometheus coupling:

```text
catalog-service
  -> Prometheus directly
```

The Collector remains the routing layer for telemetry.

## Local ports

```text
OpenTelemetry Collector OTLP gRPC: 4317
OpenTelemetry Collector OTLP HTTP: 4318
OpenTelemetry Collector Prometheus exporter: 9464
Jaeger UI: 16686
Prometheus UI: 9090
```

Prometheus UI:

```text
http://localhost:9090
```

Prometheus target health:

```text
http://localhost:9090/targets
```

## Prometheus config

Recommended path:

```text
deployments/local/prometheus/prometheus.yml
```

```yaml
global:
  scrape_interval: 15s
  evaluation_interval: 15s

scrape_configs:
  - job_name: otel-collector
    static_configs:
      - targets: ["otel-collector:9464"]
```

Prometheus uses the Docker Compose service name `otel-collector` because Prometheus and the Collector run on the same Compose network.

## OpenTelemetry Collector config

Recommended path:

```text
deployments/local/otel-collector/config.yaml
```

Prometheus exporter:

```yaml
exporters:
  prometheus:
    endpoint: 0.0.0.0:9464
```

Metrics pipeline:

```yaml
service:
  pipelines:
    metrics:
      receivers: [otlp]
      processors: [batch]
      exporters: [debug, prometheus]
```

Full local shape:

```yaml
receivers:
  otlp:
    protocols:
      grpc:
        endpoint: 0.0.0.0:4317
      http:
        endpoint: 0.0.0.0:4318

processors:
  batch:

exporters:
  debug:
    verbosity: detailed

  otlp/jaeger:
    endpoint: jaeger:4317
    tls:
      insecure: true

  prometheus:
    endpoint: 0.0.0.0:9464

service:
  pipelines:
    traces:
      receivers: [otlp]
      processors: [batch]
      exporters: [debug, otlp/jaeger]

    metrics:
      receivers: [otlp]
      processors: [batch]
      exporters: [debug, prometheus]
```

## Docker Compose

Prometheus service:

```yaml
prometheus:
  image: prom/prometheus:latest
  container_name: bfstore-prometheus
  command:
    - "--config.file=/etc/prometheus/prometheus.yml"
  volumes:
    - ./deployments/local/prometheus/prometheus.yml:/etc/prometheus/prometheus.yml:ro
    - bfstore-prometheus-data:/prometheus
  ports:
    - "9090:9090"
  depends_on:
    - otel-collector
  networks:
    - bfstore-local
```

The Collector should expose:

```yaml
ports:
  - "4317:4317"
  - "4318:4318"
  - "9464:9464"
```

Add the Prometheus volume:

```yaml
volumes:
  bfstore-prometheus-data:
```

If MySQL already has a volume:

```yaml
volumes:
  bfstore-mysql-data:
  bfstore-prometheus-data:
```

For repeatable portfolio builds, pin image versions later instead of using `latest`.

## Running locally

Start observability services:

```bash
make observability-up
```

Or directly:

```bash
docker compose up -d otel-collector jaeger prometheus
```

Check status:

```bash
docker compose ps
```

Expected containers:

```text
bfstore-otel-collector
bfstore-jaeger
bfstore-prometheus
```

## Run catalog-service with telemetry

From the host:

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
  -H 'x-correlation-id: local-dev-prometheus-123' \
  -d '{"page":{"page_size":5}}' \
  localhost:50051 \
  bfstore.catalog.v1.CatalogService/ListProducts
```

## Check Prometheus target health

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

## Metric name normalisation

OpenTelemetry metric names use dots:

```text
db.client.connections.open
```

Prometheus commonly exposes them with underscores:

```text
db_client_connections_open
```

Use the underscore form when querying in Prometheus.

## Starter PromQL queries

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
db_client_connections_wait_count
```

```promql
db_client_connections_wait_duration
```

## Operational PromQL queries

Connection wait rate:

```promql
rate(db_client_connections_wait_count[5m])
```

Connection wait duration rate:

```promql
rate(db_client_connections_wait_duration[5m])
```

Open versus in-use connections:

```promql
db_client_connections_open
```

```promql
db_client_connections_in_use
```

Idle connections:

```promql
db_client_connections_idle
```

## Interpreting results

Healthy local development usually shows:

```text
low open connections
low in-use connections
some idle connections
zero or low wait count
zero or low wait duration
```

Possible pool pressure:

```text
in-use connections close to open connections
open connections close to max connections
wait count increasing
wait duration increasing
```

Possible causes:

```text
slow queries
pool too small
traffic spike
database server saturation
connections held too long
```

## Troubleshooting

### Prometheus target is down

Check:

```bash
docker compose ps
docker compose logs -f prometheus
docker compose logs -f otel-collector
```

Confirm Prometheus config:

```yaml
targets: ["otel-collector:9464"]
```

Confirm Collector exporter:

```yaml
prometheus:
  endpoint: 0.0.0.0:9464
```

Confirm both services are on the same Compose network.

### Prometheus target is up but DB metrics are missing

Check:

```text
catalog-service is running
TELEMETRY_ENABLED=true
MetricsEnabled is true
dbmetrics.Register is called after database.Open
a fresh catalog request has been sent
```

Check Collector logs:

```bash
docker compose logs -f otel-collector
```

### Metric names do not work with dots

Use underscores:

```promql
db_client_connections_open
```

instead of:

```promql
db.client.connections.open
```

### Metrics stay flat

That may be normal in local development. Send repeated requests or later run a small load test.

## Current milestone

This Prometheus integration proves:

```text
catalog-service emits OTLP metrics
Collector receives metrics
Collector exposes metrics in Prometheus format
Prometheus scrapes the Collector
DB pool metrics are queryable with PromQL
```

## Next step

After Prometheus, add Grafana:

```text
catalog-service
  -> OpenTelemetry Collector
  -> Jaeger for traces
  -> Prometheus for metrics
  -> Grafana for dashboards
```

Keep it boring where production matters.
