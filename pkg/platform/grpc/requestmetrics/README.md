# gRPC Request Metrics

This package provides reusable gRPC server request metrics for bfstore services.

It is designed to be used as a unary gRPC server interceptor.

## Purpose

Request metrics answer service-level operational questions such as:

```text
How much traffic is the service handling?
Which gRPC methods are being called?
How many requests are failing?
What status codes are being returned?
How long are requests taking?
```

Traces explain what happened inside a single request.

Request metrics explain service behaviour over time.

## Metric names

The package emits bfstore-owned metric names:

```text
bfstore.rpc.server.requests.total
bfstore.rpc.server.request.duration
```

When exported to Prometheus, these usually appear as:

```text
bfstore_rpc_server_requests_total
bfstore_rpc_server_request_duration_count
bfstore_rpc_server_request_duration_sum
bfstore_rpc_server_request_duration_bucket
```

The exact Prometheus series depend on how the OpenTelemetry Collector exports histogram data.

## Request count

```text
bfstore.rpc.server.requests.total
```

Type:

```text
Int64 counter
```

Unit:

```text
{request}
```

Purpose:

```text
Counts completed gRPC unary server requests.
```

Useful PromQL:

```promql
sum(rate(bfstore_rpc_server_requests_total[5m]))
```

By method:

```promql
sum by (rpc_method) (rate(bfstore_rpc_server_requests_total[5m]))
```

By gRPC status code:

```promql
sum by (rpc_grpc_status_code) (rate(bfstore_rpc_server_requests_total[5m]))
```

Error rate:

```promql
sum(rate(bfstore_rpc_server_requests_total{rpc_grpc_status_code!="OK"}[5m]))
```

## Request duration

```text
bfstore.rpc.server.request.duration
```

Type:

```text
Float64 histogram
```

Unit:

```text
ms
```

Purpose:

```text
Records request duration for completed gRPC unary server requests.
```

Average duration:

```promql
sum(rate(bfstore_rpc_server_request_duration_sum[5m]))
/
clamp_min(sum(rate(bfstore_rpc_server_request_duration_count[5m])), 1)
```

Average duration by method:

```promql
sum by (rpc_method) (rate(bfstore_rpc_server_request_duration_sum[5m]))
/
clamp_min(sum by (rpc_method) (rate(bfstore_rpc_server_request_duration_count[5m])), 1)
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

## Attributes

Each measurement is recorded with attributes similar to:

```text
service.name
rpc.system
rpc.service
rpc.method
rpc.grpc.status_code
```

Example values:

```text
service.name=catalog-service
rpc.system=grpc
rpc.service=bfstore.catalog.v1.CatalogService
rpc.method=ListProducts
rpc.grpc.status_code=OK
```

## Usage

Create the interceptor:

```go
requestMetricsInterceptor, err := requestmetrics.UnaryServerInterceptor(requestmetrics.Config{
	MeterName:   "github.com/mantrobuslawal/bfstore/services/catalog-service",
	ServiceName: "catalog-service",
})
if err != nil {
	return nil, err
}
```

Add it to the gRPC unary interceptor chain:

```go
grpc.NewServer(
	grpc.StatsHandler(otelgrpc.NewServerHandler()),
	grpc.ChainUnaryInterceptor(
		platforminterceptors.UnaryRecoveryInterceptor(logger),
		platforminterceptors.UnaryCorrelationIDInterceptor(),
		requestMetricsInterceptor,
		platforminterceptors.UnaryLoggingInterceptor(logger),
	),
)
```

## Recommended interceptor order

```text
recovery
correlation ID
request metrics
logging
handler
```

Why:

```text
Recovery converts panics into errors.
Correlation ID ensures request identity exists.
Request metrics observe the final handler result.
Logging records the same request outcome.
```

## Local verification

Start observability:

```bash
make observability-up
```

Run catalog-service with telemetry enabled:

```bash
cd services/catalog-service

TELEMETRY_ENABLED=true \
OTEL_EXPORTER_OTLP_ENDPOINT=localhost:4317 \
OTEL_EXPORTER_OTLP_INSECURE=true \
GRPC_REFLECTION_ENABLED=true \
go run ./cmd/catalog-service
```

Send traffic:

```bash
REQUESTS=100 SLEEP_SECONDS=0.05 ./scripts/local/catalog-load.sh
```

Open Prometheus:

```text
http://localhost:9090
```

Try:

```promql
bfstore_rpc_server_requests_total
```

```promql
sum(rate(bfstore_rpc_server_requests_total[5m]))
```

Open Grafana:

```text
http://localhost:3000
```

Dashboard:

```text
Catalog Service Overview
```

## Testing

Run:

```bash
go test ./pkg/platform/grpc/requestmetrics -v
go test ./...
```

## Design notes

The package intentionally uses bfstore-owned metric names:

```text
bfstore.rpc.server.requests.total
bfstore.rpc.server.request.duration
```

This keeps the implementation honest while the platform layer is still growing.

Later, the package may be aligned more tightly with OpenTelemetry RPC semantic convention metric names.

## Practical rule

```text
Traces explain individual requests.
Metrics explain repeated behaviour.
Dashboards make the signal visible.
```

Keep it boring where production matters.
