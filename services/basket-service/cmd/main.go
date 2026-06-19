package main

import (
	"context"
	"database/sql"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	catalogv1 "github.com/mantrobuslawal/bfstore/gen/go/bfstore/catalog/v1"
	"github.com/mantrobuslawal/bfstore/pkg/platform/dbmetrics"
	"github.com/mantrobuslawal/bfstore/pkg/platform/healthcheck"
	"github.com/mantrobuslawal/bfstore/pkg/platform/telemetry"
	"github.com/mantrobuslawal/bfstore/services/basket-service/internal/basket"
	"github.com/mantrobuslawal/bfstore/services/basket-service/internal/config"
	"github.com/mantrobuslawal/bfstore/services/basket-service/internal/database"
	basketgrpc "github.com/mantrobuslawal/bfstore/services/basket-service/internal/grpcadapter"
	baskethealth "github.com/mantrobuslawal/bfstore/services/basket-service/internal/health"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/reflection"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	cfg, err := config.Load()
	if err != nil {
		logger.Error("failed to load configuration", "error", err)
		os.Exit(1)
	}

	db, err := database.Open(cfg.Database)
	if err != nil {
		logger.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer closeDatabase(logger, db)

	logger.Info("database connection opened")

	if err := dbmetrics.Register(db, dbmetrics.Config{
		MeterName: "github.com/mantrobuslawal/bfstore/services/basket-service",
		DBSystem:  "mysql",
		DBName:    cfg.Database.Name,
	}); err != nil {
		logger.Error("failed to register database metrics", "error", err)
		os.Exit(1)
	}

	var telemetryRuntime *telemetry.Runtime

	if cfg.Telemetry.TelemetryEnabled {
		telemetryConfig := telemetry.DefaultConfig("basket-service")
		telemetryConfig.Environment = cfg.Environment
		telemetryConfig.ServiceVersion = cfg.Telemetry.ServiceVersion
		telemetryConfig.OTLPEndpoint = cfg.Telemetry.OTLPEndpoint
		telemetryConfig.OTLPInsecure = cfg.Telemetry.OTLPInsecure
		telemetryConfig.TracesEnabled = cfg.Telemetry.TracesEnabled
		telemetryConfig.MetricsEnabled = cfg.Telemetry.MetricsEnabled
		telemetryConfig.MetricExportInterval = cfg.Telemetry.MetricsExportInterval

		telemetryRuntime, err = telemetry.Setup(ctx, telemetryConfig)
		if err != nil {
			logger.Error("failed to setup telemetry", "error", err)
			os.Exit(1)
		}
		logger.Info(
			"telemetry enabled",
			"service", telemetryConfig.ServiceName,
			"environment", telemetryConfig.Environment,
			"oltp_enpoint", telemetryConfig.OTLPEndpoint,
			"traces_enabled", telemetryConfig.TracesEnabled,
			"metrics_enabled", telemetryConfig.TracesEnabled,
		)

	}

	repository := basket.NewMySQLRepository(db, logger.With(
		"service", "basket-service",
		"component", "basket-repository",
	))

	catalogConn, err := grpc.NewClient(
		cfg.CatalogGRPCAddress,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		logger.Error("failed to create catalog grpc client",
			"error", err,
			"catalog_grpc_address", cfg.CatalogGRPCAddress,
		)
		os.Exit(1)
	}
	defer catalogConn.Close()

	catalogClient := basket.NewCatalogGRPCClient(
		catalogv1.NewCatalogServiceClient(catalogConn),
		logger.With(
			"service", "basket-service",
			"component", "catalog_grpc_client",
		),
		cfg.CatalogRequestTimeout,
	)

	basketService := basket.NewService(
		repository,
		*catalogClient,
		logger.With(
			"service", "basket-service",
			"component", "catalog_grpc_client",
		),
	)

	grpcServer, err := basketgrpc.NewServer(basketService, logger)
	if err != nil {
		logger.Error("failed to setup requestmetrics interceptor", "error", err)
		os.Exit(1)
	}

	if cfg.EnableGRPCReflection {
		reflection.Register(grpcServer)
		logger.Info("grpc reflection enabled")
	}

	const basketServiceName = "bfstore.basket.v1.BasketService"

	healthManager := healthcheck.NewManager(grpcServer)
	healthManager.RegisterService(basketServiceName)

	basketHealthchecker := baskethealth.NewChecker(db)

	if err := basketHealthchecker.Ready(ctx); err != nil {
		logger.Error("service is not ready", "error", err)
		os.Exit(1)
	}

	logger.Info("database readiness check passed")

	healthManager.MarkServing()

	logger.Info("grpc health service is registered")

	listener, err := net.Listen("tcp", ":"+cfg.GRPCPort)
	if err != nil {
		logger.Error("failed to listen for gRPC", "port", cfg.GRPCPort, "error", err)
		os.Exit(1)
	}

	serverErr := make(chan error, 1)

	go func() {
		logger.Info("basket-service started", "grpc_port", cfg.GRPCPort)
		serverErr <- grpcServer.Serve(listener)
	}()

	go func() { monitorReadiness(ctx, logger, basketHealthchecker, healthManager) }()

	select {
	case <-ctx.Done():
		logger.Info("shutdown signal received")

	case err := <-serverErr:
		if err != nil {
			logger.Error("gRPC server failed", "error", err)
		}
		stop()
	}

	logger.Info("marking service not serving")
	healthManager.MarkNotServing()
	healthManager.Shutdown()

	if telemetryRuntime != nil {
		telemetryShutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err = telemetryRuntime.Shutdown(telemetryShutdownCtx); err != nil {
			logger.Error("failed to shutdown telemetry", "error", err)
		}
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	done := make(chan struct{})
	go func() {
		grpcServer.GracefulStop()
		close(done)
	}()

	select {
	case <-done:
		logger.Info("basket-service stopped gracefully")
	case <-shutdownCtx.Done():
		logger.Warn("graceful shutdown timed out; forcing stop")
		grpcServer.Stop()
	}
}

func closeDatabase(logger *slog.Logger, db *sql.DB) {
	if err := db.Close(); err != nil {
		logger.Error("failed to close database", "error", err)
	}
}

func monitorReadiness(
	ctx context.Context,
	logger *slog.Logger,
	checker *baskethealth.Checker,
	healthServer *healthcheck.Manager,
) {

	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			healthServer.MarkNotServing()
			return

		case <-ticker.C:
			if err := checker.Ready(ctx); err != nil {
				logger.Warn("readiness check failed", "error", err)
				healthServer.MarkNotServing()
				continue
			}

			healthServer.MarkServing()
		}
	}

}
