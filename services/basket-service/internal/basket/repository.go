package basket

import (
	"context"
	"database/sql"
	"log/slog"
)

// Repository defines Basket Service persistence behaviour.
type Repository interface {
	CreateBasket(ctx context.Context, basket Basket) (Basket, error)
	GetBasket(ctx context.Context, basketID BasketID) (Basket, error)
	AddItem(ctx context.Context, query BasketQuery) (Basket, error)
	UpdateItemQuantity(ctx context.Context, query BasketQuery) (Basket, error)
	RemoveItem(ctx context.Context, query BasketQuery) (Basket, error)
	ClearBasket(ctx context.Context, query BasketQuery) (Basket, error)
}

type MySQLRepository struct {
	db     *sql.DB
	logger *slog.Logger
}

func NewMySQLRepository(db *sql.DB, logger *slog.Logger) *Repository {
	if logger == nil {
		logger = slog.Default()
	}

	return &MySQLRepository{ // implement  repo interface to remove error highlight
		db:     db,
		logger: logger,
	}
}
