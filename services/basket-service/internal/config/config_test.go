package config

import (
	"testing"
	"time"
)

func TestLoadDefaults(t *testing.T) {
	t.Setenv("MYSQL_PASSWORD", "bfstore_basket_password")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v, want nil", err)
	}

	if cfg.GRPCPort != "50052" {
		t.Fatalf("GRPCPort = %q, want 50052", cfg.GRPCPort)
	}

	if cfg.CatalogGRPCAddress != "localhost:50051" {
		t.Fatalf("CatalogGRPCAddress = %q, want localhost:50051", cfg.CatalogGRPCAddress)
	}

	if cfg.CatalogRequestTimeout != 2*time.Second {
		t.Fatalf("CatalogRequestTimeout = %v, want 2s", cfg.CatalogRequestTimeout)
	}

	if cfg.Telemetry.OTLPEndpoint != "" {
		t.Fatalf("OTLPEndpoint = %q, want empty default", cfg.Telemetry.OTLPEndpoint)
	}
}

func TestLoadReadsOTLPEndpoint(t *testing.T) {
	t.Setenv("MYSQL_PASSWORD", "bfstore_basket_password")
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "localhost:4317")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v, want nil", err)
	}

	if cfg.Telemetry.OTLPEndpoint != "localhost:4317" {
		t.Fatalf("OTLPEndpoint = %q, want localhost:4317", cfg.Telemetry.OTLPEndpoint)
	}
}

func TestDatabaseConfigDSN(t *testing.T) {
	cfg := DatabaseConfig{
		Host:     "localhost",
		Port:     "3306",
		Name:     "bfstore_basket",
		User:     "bfstore_basket",
		Password: "secret",
	}

	got := cfg.DSN()
	want := "bfstore_basket:secret@tcp(localhost:3306)/bfstore_basket?parseTime=true&multiStatements=false&charset=utf8mb4,utf8"
	if got != want {
		t.Fatalf("DSN() = %q, want %q", got, want)
	}
}
