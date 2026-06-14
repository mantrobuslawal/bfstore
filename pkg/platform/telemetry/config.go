package telemetry

import (
	"fmt"
	"strings"
	"time"
)

const (
	defaultEnvironment          = "local"
	defaultOTLPEndpoint         = "localhost:4317"
	defaultMetricExportInterval = 30 * time.Second
)

// Config controls platform telemetry initialisation for a bfstore service.
//
// Use DefaultConfig to get sensible local-development defaults, then override
// fields from service configuration or environment variables.
type Config struct {
	// ServiceName is the logical service name used in telemetry resource
	// attributes. Examples: "catalog-service", "basket-service".
	ServiceName string

	// ServiceVersion is an optional version string such as a git SHA, semantic
	// version, or build number.
	ServiceVersion string

	// Environment identifies where the service is running.
	// Examples: "local", "dev", "test", "prod".
	Environment string

	// OTLPEndpoint is the OTLP gRPC endpoint used for traces and metrics.
	// For local OpenTelemetry Collector defaults this is usually
	// "localhost:4317".
	OTLPEndpoint string

	// OTLPInsecure disables transport security for the OTLP gRPC exporter.
	// This is convenient for local development with a local Collector.
	OTLPInsecure bool

	// TracesEnabled controls whether a TracerProvider and trace exporter are
	// created.
	TracesEnabled bool

	// MetricsEnabled controls whether a MeterProvider and metric exporter are
	// created.
	MetricsEnabled bool

	// MetricExportInterval controls how often metrics are exported.
	// If zero, a default interval is used.
	MetricExportInterval time.Duration
}

// DefaultConfig returns a telemetry config with sensible local-development
// defaults for a service.
func DefaultConfig(serviceName string) Config {
	return Config{
		ServiceName:           serviceName,
		Environment:           defaultEnvironment,
		OTLPEndpoint:          defaultOTLPEndpoint,
		OTLPInsecure:          true,
		TracesEnabled:         true,
		MetricsEnabled:        true,
		MetricExportInterval:  defaultMetricExportInterval,
	}
}

// WithDefaults returns a copy of cfg with empty optional fields filled with
// default values.
//
// Boolean fields are intentionally not changed here. Use DefaultConfig when you
// want traces and metrics enabled by default. This allows tests and services to
// explicitly set TracesEnabled or MetricsEnabled to false.
func (cfg Config) WithDefaults() Config {
	cfg.ServiceName = strings.TrimSpace(cfg.ServiceName)
	cfg.ServiceVersion = strings.TrimSpace(cfg.ServiceVersion)
	cfg.Environment = strings.TrimSpace(cfg.Environment)
	cfg.OTLPEndpoint = strings.TrimSpace(cfg.OTLPEndpoint)

	if cfg.Environment == "" {
		cfg.Environment = defaultEnvironment
	}

	if cfg.OTLPEndpoint == "" {
		cfg.OTLPEndpoint = defaultOTLPEndpoint
	}

	if cfg.MetricExportInterval == 0 {
		cfg.MetricExportInterval = defaultMetricExportInterval
	}

	return cfg
}

// Validate checks whether cfg contains enough information to initialise
// telemetry safely.
func (cfg Config) Validate() error {
	if strings.TrimSpace(cfg.ServiceName) == "" {
		return fmt.Errorf("telemetry service name is required")
	}

	if cfg.TracesEnabled || cfg.MetricsEnabled {
		if strings.TrimSpace(cfg.OTLPEndpoint) == "" {
			return fmt.Errorf("telemetry OTLP endpoint is required when traces or metrics are enabled")
		}
	}

	if cfg.MetricsEnabled && cfg.MetricExportInterval <= 0 {
		return fmt.Errorf("telemetry metric export interval must be greater than zero")
	}

	return nil
}
