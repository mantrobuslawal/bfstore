package grpcadapter

import (
	"log/slog"

	catalogv1 "github.com/mantrobuslawal/bfstore/gen/go/bfstore/catalog/v1"
	platforminterceptors "github.com/mantrobuslawal/bfstore/pkg/platform/grpc/interceptors"
	"github.com/mantrobuslawal/bfstore/pkg/platform/grpc/requestmetrics"
	"github.com/mantrobuslawal/bfstore/services/catalog-service/internal/catalog"

	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
)

// NewServer creates the Catalogue Service gRPC server.
func NewServer(catalogService *catalog.Service, logger *slog.Logger) (*grpc.Server, error) {
	requestmetricsInterceptor, err := requestmetrics.UnaryServerInterceptor(requestmetrics.Config{
		MeterName:   "github.com/mantrobuslawal/bfstore/services/catalog-service",
		ServiceName: "catalog-service",
	})
	if err != nil {
		return nil, err
	}

	server := grpc.NewServer(
		grpc.StatsHandler(otelgrpc.NewServerHandler()),
		grpc.ChainUnaryInterceptor(
			platforminterceptors.UnaryRecoveryInterceptor(logger),
			platforminterceptors.UnaryCorrelationIDInterceptor(),
			requestmetricsInterceptor,
			platforminterceptors.UnaryLoggingInterceptor(logger),
		),
	)

	handler := NewCatalogHandler(catalogService, logger)

	catalogv1.RegisterCatalogServiceServer(server, handler)

	return server, nil
}
