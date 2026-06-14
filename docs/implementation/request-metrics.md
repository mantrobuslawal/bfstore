# Request Metrics

This document explains the request metrics added to the bfstore catalog service.

## Current flow

```text
catalog-service
  -> gRPC request metrics interceptor
  -> OpenTelemetry metrics SDK
  -> OpenTelemetry Collector
  -> Prometheus
  -> Grafana
```

## Why request metrics matter

Request metrics answer questions that traces alone do not answer easily:

```text
How many requests is the service handling?
Which methods are busiest?
How many requests are failing?
What is the request duration trend?
Is p95 duration moving in the wrong direction?
```

## Metrics emitted

```text
bfstore.rpc.server.requests.total
bfstore.rpc.server.request.duration
```

Prometheus-normalised names:

```text
bfstore_rpc_server_requests_total
bfstore_rpc_server_request_duration_count
bfstore_rpc_server_request_duration_sum
bfstore_rpc_server_request_duration_bucket
```

## Useful PromQL

Request rate:

```promql
sum(rate(bfstore_rpc_server_requests_total[5m]))
```

Request rate by method:

```promql
sum by (rpc_method) (rate(bfstore_rpc_server_requests_total[5m]))
```

Request rate by status:

```promql
sum by (rpc_grpc_status_code) (rate(bfstore_rpc_server_requests_total[5m]))
```

Error rate:

```promql
sum(rate(bfstore_rpc_server_requests_total{rpc_grpc_status_code!="OK"}[5m]))
```

Average duration:

```promql
sum(rate(bfstore_rpc_server_request_duration_sum[5m]))
/
clamp_min(sum(rate(bfstore_rpc_server_request_duration_count[5m])), 1)
```

p95 duration:

```promql
histogram_quantile(
  0.95,
  sum(rate(bfstore_rpc_server_request_duration_bucket[5m])) by (le)
)
```

p95 duration by method:

```promql
histogram_quantile(
  0.95,
  sum(rate(bfstore_rpc_server_request_duration_bucket[5m])) by (le, rpc_method)
)
```

## Grafana dashboard

The updated dashboard is:

```text
Catalog Service Overview
```

Dashboard file:

```text
deployments/local/grafana/dashboards/catalog-service-overview.json
```

It includes:

```text
Catalog request overview
Catalog database pool overview
```

## Verification

Start observability:

```bash
make observability-up
```

Run catalog-service with telemetry:

```bash
make catalog-run-telemetry
```

Generate traffic:

```bash
REQUESTS=100 SLEEP_SECONDS=0.05 ./scripts/local/catalog-load.sh
```

Check:

```text
Prometheus: http://localhost:9090
Grafana:    http://localhost:3000
```

## Troubleshooting

### Request metrics are missing in Prometheus

Check that:

```text
catalog-service was restarted after adding the interceptor
TELEMETRY_ENABLED=true
request metrics interceptor is in the gRPC chain
Collector metrics pipeline exports to Prometheus
traffic has been sent after startup
```

### Duration buckets are missing

Depending on Collector/exporter configuration, histogram output may differ.

Search Prometheus for:

```text
bfstore_rpc_server_request_duration
```

If `_bucket` series are unavailable, average duration using `_sum` and `_count` may still work.

### Grafana panels show no data

Check Prometheus first.

If Prometheus has no request metrics, Grafana cannot display them.

## Practical rule

```text
Do not start with alerts.
Start with trustworthy measurements.
```

Keep it boring where production matters.
