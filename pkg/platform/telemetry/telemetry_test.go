package telemetry

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestDefaultConfig(t *testing.T) {
	t.Parallel()

	cfg := DefaultConfig("catalog-service")

	if cfg.ServiceName != "catalog-service" {
		t.Fatalf("ServiceName = %q, want %q", cfg.ServiceName, "catalog-service")
	}

	if cfg.Environment != defaultEnvironment {
		t.Fatalf("Environment = %q, want %q", cfg.Environment, defaultEnvironment)
	}

	if cfg.OTLPEndpoint != defaultOTLPEndpoint {
		t.Fatalf("OTLPEndpoint = %q, want %q", cfg.OTLPEndpoint, defaultOTLPEndpoint)
	}

	if !cfg.OTLPInsecure {
		t.Fatal("OTLPInsecure = false, want true")
	}

	if !cfg.TracesEnabled {
		t.Fatal("TracesEnabled = false, want true")
	}

	if !cfg.MetricsEnabled {
		t.Fatal("MetricsEnabled = false, want true")
	}

	if cfg.MetricExportInterval != defaultMetricExportInterval {
		t.Fatalf("MetricExportInterval = %s, want %s", cfg.MetricExportInterval, defaultMetricExportInterval)
	}
}

func TestConfigWithDefaultsTrimsAndFillsOptionalDefaults(t *testing.T) {
	t.Parallel()

	cfg := Config{
		ServiceName:          " catalog-service ",
		ServiceVersion:       " abc123 ",
		Environment:          " ",
		OTLPEndpoint:         " ",
		OTLPInsecure:         true,
		TracesEnabled:        false,
		MetricsEnabled:       false,
		MetricExportInterval: 0,
	}

	got := cfg.WithDefaults()

	if got.ServiceName != "catalog-service" {
		t.Fatalf("ServiceName = %q, want %q", got.ServiceName, "catalog-service")
	}

	if got.ServiceVersion != "abc123" {
		t.Fatalf("ServiceVersion = %q, want %q", got.ServiceVersion, "abc123")
	}

	if got.Environment != defaultEnvironment {
		t.Fatalf("Environment = %q, want %q", got.Environment, defaultEnvironment)
	}

	if got.OTLPEndpoint != defaultOTLPEndpoint {
		t.Fatalf("OTLPEndpoint = %q, want %q", got.OTLPEndpoint, defaultOTLPEndpoint)
	}

	if got.TracesEnabled {
		t.Fatal("TracesEnabled = true, want false")
	}

	if got.MetricsEnabled {
		t.Fatal("MetricsEnabled = true, want false")
	}

	if got.MetricExportInterval != defaultMetricExportInterval {
		t.Fatalf("MetricExportInterval = %s, want %s", got.MetricExportInterval, defaultMetricExportInterval)
	}
}

func TestConfigValidateRequiresServiceName(t *testing.T) {
	t.Parallel()

	cfg := DefaultConfig(" ")
	err := cfg.Validate()

	if err == nil {
		t.Fatal("Validate() error = nil, want non-nil")
	}

	if !strings.Contains(err.Error(), "service name") {
		t.Fatalf("Validate() error = %q, want service name error", err.Error())
	}
}

func TestConfigValidateRequiresEndpointWhenSignalsEnabled(t *testing.T) {
	t.Parallel()

	cfg := Config{
		ServiceName:    "catalog-service",
		OTLPEndpoint:   " ",
		TracesEnabled:  true,
		MetricsEnabled: false,
	}

	err := cfg.Validate()

	if err == nil {
		t.Fatal("Validate() error = nil, want non-nil")
	}

	if !strings.Contains(err.Error(), "OTLP endpoint") {
		t.Fatalf("Validate() error = %q, want OTLP endpoint error", err.Error())
	}
}

func TestConfigValidateAllowsEmptyEndpointWhenSignalsDisabled(t *testing.T) {
	t.Parallel()

	cfg := Config{
		ServiceName:    "catalog-service",
		OTLPEndpoint:   " ",
		TracesEnabled:  false,
		MetricsEnabled: false,
	}

	if err := cfg.Validate(); err != nil {
		t.Fatalf("Validate() error = %v, want nil", err)
	}
}

func TestConfigValidateRequiresPositiveMetricExportInterval(t *testing.T) {
	t.Parallel()

	cfg := Config{
		ServiceName:          "catalog-service",
		OTLPEndpoint:         "localhost:4317",
		TracesEnabled:        false,
		MetricsEnabled:       true,
		MetricExportInterval: -1 * time.Second,
	}

	err := cfg.Validate()

	if err == nil {
		t.Fatal("Validate() error = nil, want non-nil")
	}

	if !strings.Contains(err.Error(), "metric export interval") {
		t.Fatalf("Validate() error = %q, want metric export interval error", err.Error())
	}
}

func TestSetupWithSignalsDisabledReturnsRuntime(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	runtime, err := Setup(ctx, Config{
		ServiceName:    "catalog-service",
		Environment:    "test",
		TracesEnabled:  false,
		MetricsEnabled: false,
	})

	if err != nil {
		t.Fatalf("Setup() error = %v, want nil", err)
	}

	if runtime == nil {
		t.Fatal("Setup() runtime = nil, want non-nil")
	}

	if err := runtime.Shutdown(ctx); err != nil {
		t.Fatalf("Shutdown() error = %v, want nil", err)
	}
}

func TestNilRuntimeShutdown(t *testing.T) {
	t.Parallel()

	var runtime *Runtime

	if err := runtime.Shutdown(context.Background()); err != nil {
		t.Fatalf("Shutdown() error = %v, want nil", err)
	}
}
