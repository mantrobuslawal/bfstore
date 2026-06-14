package config

import (
	"fmt"
	"os"
	"strings"
	"time"
)

// Config contains runtime configuration for Catalogue Service.
type Config struct {
	Environment          string
	LogLevel             string
	GRPCPort             string
	EnableGRPCReflection bool
	Database             DatabaseConfig
	Telemetry            TelemetryConfig
}

// DatabaseConfig contains MySQL connection configuration.
type DatabaseConfig struct {
	Host     string
	Port     string
	Name     string
	User     string
	Password string
}

// TelemetryConfig contains OTLP configuration for service
type TelemetryConfig struct {
	Environment           string
	ServiceVersion        string
	OTLPEndpoint          string
	OTLPInsecure          bool
	TelemetryEnabled      bool
	TracesEnabled         bool
	MetricsEnabled        bool
	MetricsExportInterval time.Duration
}

// Load reads configuration from environment variables.
func Load() (Config, error) {
	cfg := Config{
		Environment: getEnv("ENVIRONMENT", "local"),
		LogLevel:    getEnv("LOG_LEVEL", "info"),
		GRPCPort:    getEnv("CATALOG_SERVICE_GRPC_PORT", "50051"),
		Database: DatabaseConfig{
			Host:     getEnv("MYSQL_HOST", "localhost"),
			Port:     getEnv("MYSQL_PORT", "3306"),
			Name:     getEnv("MYSQL_DATABASE", "bfstore_catalog"),
			User:     getEnv("MYSQL_USER", "bfstore_catalog_user"),
			Password: getEnv("MYSQL_PASSWORD", "bfstore_catalog_password"),
		},
		EnableGRPCReflection: loadBoolEnv("GRPC_REFLECTION_ENABLED", false),
		Telemetry: TelemetryConfig{
			Environment:           getEnv("ENVIRONMENT", "local"),
			ServiceVersion:        getEnv("SERVICE_VERSION", ""),
			OTLPEndpoint:          getEnv("OTEL_EXPORTER_OLTP_ENDPOINT", ""), // telemetry package config will set if absent
			OTLPInsecure:          loadBoolEnv("OTEL_EXPORTER_OTLP_INSECURE", true),
			TelemetryEnabled:      loadBoolEnv("TELEMETRY_ENABLED", false),
			TracesEnabled:         loadBoolEnv("TRACES_ENABLED", true),
			MetricsEnabled:        loadBoolEnv("METRICS_ENABLED", true),
			MetricsExportInterval: loadTimeEnv("METRICS_EXPORT_INTERVAL"),
		},
	}

	if cfg.Database.Password == "" {
		return Config{}, fmt.Errorf("MYSQL_PASSWORD must be set")
	}

	return cfg, nil
}

// DSN returns a MySQL data source name suitable for database/sql.
func (d DatabaseConfig) DSN() string {
	return fmt.Sprintf(
		"%s:%s@tcp(%s:%s)/%s?parseTime=true&multiStatements=false&charset=utf8mb4,utf8",
		d.User,
		d.Password,
		d.Host,
		d.Port,
		d.Name,
	)
}

func getEnv(key string, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	return value
}

func loadBoolEnv(key string, fallback bool) bool {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}

	switch strings.ToLower(value) {
	case "true", "1", "yes", "y":
		return true
	case "false", "0", "no", "n":
		return false
	default:
		return fallback
	}
}

func loadTimeEnv(key string) time.Duration {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		fallback, _ := time.ParseDuration("0s")
		return fallback
	}

	// expecting a string number e.g. "10" or "5"
	newValue := []string{value, "s"}
	value = strings.Join(newValue, "")
	duration, err := time.ParseDuration(value)
	if err != nil {
		fallback, _ := time.ParseDuration("0s")
		return fallback
	}
	return duration
}
