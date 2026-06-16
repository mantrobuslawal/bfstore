package main

/*
EXAMPLE OF PASSING LOGGER FROM MAIN.GO TO ALL APP LAYERS
gRPC, SERVICE AND REPO

package main

import (
	"log/slog"
	"os"

	"github.com/mantrobuslawal/bfstore/services/basket-service/internal/basket"
	"github.com/mantrobuslawal/bfstore/services/basket-service/internal/database"
	"github.com/mantrobuslawal/bfstore/services/basket-service/internal/grpcadapter"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	db, err := database.Open()
	if err != nil {
		logger.Error("failed to open database", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	repo := basket.NewRepository(
		db,
		logger.With(
			"service", "basket-service",
			"component", "basket_repository",
		),
	)

	svc := basket.NewService(
		repo,
		logger.With(
			"service", "basket-service",
			"component", "basket_service",
		),
	)

	handler := grpcadapter.NewBasketHandler(
		svc,
		logger.With(
			"service", "basket-service",
			"component", "basket_grpc_handler",
		),
	)

	_ = handler

	// Register gRPC server, health checks, reflection, interceptors, etc.
}
*/
