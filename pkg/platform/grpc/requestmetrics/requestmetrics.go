package requestmetrics

import (
	"context"
	"errors"
	"strings"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
)

const defaultMeterName = "github.com/mantrobuslawal/bfstore/pkg/platform/grpc/requestmetrics"

// Config describes gRPC metric registration.
type Config struct {
	MeterName   string
	ServiceName string
}

// UnaryServerInterceptor returns a gRPC unary server interceptor that records request metrics.
func UnaryServerInterceptor(cfg Config) (grpc.UnaryServerInterceptor, error) {
	meterName := strings.TrimSpace(cfg.MeterName)
	if meterName == "" {
		meterName = defaultMeterName
	}

	serviceName := strings.TrimSpace(cfg.ServiceName)
	if serviceName == "" {
		return nil, errors.New("requestmetics: service name is required")
	}

	meter := otel.Meter(meterName)

	requestsTotal, err := meter.Int64Counter(
		"bfstore.rpc.server.requests.total",
		metric.WithDescription("Total number of gRPC server requests handled by the service."),
		metric.WithUnit("{request}"),
	)
	if err != nil {
		return nil, err
	}

	requestDuration, err := meter.Float64Histogram(
		"bfstore.rpc.server.request.duration",
		metric.WithDescription("Duration of gRPC server requests handled by the service."),
		metric.WithUnit("ms"),
	)
	if err != nil {
		return nil, err
	}

	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		start := time.Now()

		resp, err := handler(ctx, req)

		durationMS := float64(time.Since(start).Microseconds()) / 1000.0
		code := status.Code(err)

		grpcService, grpcMethod := splitFullMethod(info.FullMethod)

		attrs := []attribute.KeyValue{
			attribute.String("service.name", serviceName),
			attribute.String("rpc.system", "grpc"),
			attribute.String("rpc.service", grpcService),
			attribute.String("rpc.method", grpcMethod),
			attribute.String("rpc.grpc.status_code", code.String()),
		}

		options := metric.WithAttributes(attrs...)

		requestsTotal.Add(ctx, 1, options)
		requestDuration.Record(ctx, durationMS, options)

		return resp, err
	}, nil
}

func splitFullMethod(fullmethod string) (string, string) {
	fullmethod = strings.TrimSpace(fullmethod)
	fullmethod = strings.TrimPrefix(fullmethod, "/")

	if fullmethod == "" {
		return "unknown", "unknown"
	}

	parts := strings.Split(fullmethod, "/")
	if len(parts) != 2 {
		return fullmethod, "unknown"
	}

	service := strings.TrimSpace(parts[0])
	method := strings.TrimSpace(parts[1])

	if service == "" {
		service = "unknown"
	}

	if method == "" {
		method = "unknown"
	}

	return service, method
}
