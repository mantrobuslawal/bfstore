package grpcadapter

import (
	"log/slog"

	basketv1 "github.com/mantrobuslawal/bfstore/gen/go/bfstore/basket/v1"
	platforminterceptors "github.com/mantrobuslawal/bfstore/pkg/platform/grpc/interceptors"
	"github.com/mantrobuslawal/bfstore/pkg/platform/grpc/requestmetrics"
	"github.com/mantrobuslawal/bfstore/services/basket-service/internal/basket"

	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
)

// NewServer creates the Basket Service gRPC server.
func NewServer(basketService *basket.Service, logger *slog.Logger) (*grpc.Server, error) {
	if logger == nil {
		logger = slog.Default()
	}

	requestmetricsInterceptor, err := requestmetrics.UnaryServerInterceptor(requestmetrics.Config{
		MeterName:   "github.com/mantrobuslawal/bfstore/services/basket-service",
		ServiceName: "basket-service",
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

	handler := NewBasketHandler(basketService, logger)

	basketv1.RegisterBasketServiceServer(server, handler)

	return server, nil
}
