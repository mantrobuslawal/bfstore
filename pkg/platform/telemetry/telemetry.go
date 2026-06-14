package telemetry

import (
	"context"
	"errors"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

// Runtime represents the telemetry providers created for a service process.
//
// Call Shutdown during service shutdown so traces and metrics can be flushed
// before the process exits.
type Runtime struct {
	tracerProvider *sdktrace.TracerProvider
	meterProvider  *metric.MeterProvider
}

// Setup initialises OpenTelemetry providers for a bfstore service.
//
// It configures resource attributes, trace context propagation, and optional
// OTLP trace and metric exporters. Service code should call Setup once during
// startup and call Runtime.Shutdown during graceful shutdown.
func Setup(ctx context.Context, cfg Config) (*Runtime, error) {
	cfg = cfg.WithDefaults()
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	res, err := newResource(ctx, cfg)
	if err != nil {
		return nil, err
	}

	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	runtime := &Runtime{}

	if cfg.TracesEnabled {
		tracerProvider, err := newTracerProvider(ctx, cfg, res)
		if err != nil {
			return nil, err
		}

		otel.SetTracerProvider(tracerProvider)
		runtime.tracerProvider = tracerProvider
	}

	if cfg.MetricsEnabled {
		meterProvider, err := newMeterProvider(ctx, cfg, res)
		if err != nil {
			shutdownErr := runtime.Shutdown(ctx)
			return nil, errors.Join(err, shutdownErr)
		}

		otel.SetMeterProvider(meterProvider)
		runtime.meterProvider = meterProvider
	}

	return runtime, nil
}

// Shutdown flushes and closes the telemetry providers created by Setup.
//
// It is safe to call Shutdown on a nil Runtime.
func (r *Runtime) Shutdown(ctx context.Context) error {
	if r == nil {
		return nil
	}

	var shutdownErr error

	if r.meterProvider != nil {
		shutdownErr = errors.Join(shutdownErr, r.meterProvider.Shutdown(ctx))
	}

	if r.tracerProvider != nil {
		shutdownErr = errors.Join(shutdownErr, r.tracerProvider.Shutdown(ctx))
	}

	return shutdownErr
}

func newResource(ctx context.Context, cfg Config) (*resource.Resource, error) {
	attrs := []attribute.KeyValue{
		attribute.String("service.name", cfg.ServiceName),
		attribute.String("deployment.environment.name", cfg.Environment),
	}

	if cfg.ServiceVersion != "" {
		attrs = append(attrs, attribute.String("service.version", cfg.ServiceVersion))
	}

	return resource.New(
		ctx,
		resource.WithAttributes(attrs...),
	)
}

func newTracerProvider(
	ctx context.Context,
	cfg Config,
	res *resource.Resource,
) (*sdktrace.TracerProvider, error) {
	options := []otlptracegrpc.Option{
		otlptracegrpc.WithEndpoint(cfg.OTLPEndpoint),
	}

	if cfg.OTLPInsecure {
		options = append(options, otlptracegrpc.WithInsecure())
	}

	exporter, err := otlptracegrpc.New(ctx, options...)
	if err != nil {
		return nil, err
	}

	return sdktrace.NewTracerProvider(
		sdktrace.WithResource(res),
		sdktrace.WithBatcher(exporter),
	), nil
}

func newMeterProvider(
	ctx context.Context,
	cfg Config,
	res *resource.Resource,
) (*metric.MeterProvider, error) {
	options := []otlpmetricgrpc.Option{
		otlpmetricgrpc.WithEndpoint(cfg.OTLPEndpoint),
	}

	if cfg.OTLPInsecure {
		options = append(options, otlpmetricgrpc.WithInsecure())
	}

	exporter, err := otlpmetricgrpc.New(ctx, options...)
	if err != nil {
		return nil, err
	}

	reader := metric.NewPeriodicReader(
		exporter,
		metric.WithInterval(cfg.MetricExportInterval),
	)

	return metric.NewMeterProvider(
		metric.WithResource(res),
		metric.WithReader(reader),
	), nil
}
