package basket

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
)

// Repository defines Basket Service persistence behaviour.
type Repository interface {
	CreateBasket(ctx context.Context, basket Basket) (Basket, error)
	GetBasket(ctx context.Context, basketID string) (Basket, error)
	AddItem(ctx context.Context, query BasketQuery) (Basket, error)
	UpdateItemQuantity(ctx context.Context, query BasketQuery) (Basket, error)
	RemoveItem(ctx context.Context, query BasketQuery) (Basket, error)
	ClearBasket(ctx context.Context, query BasketQuery) (Basket, error)
}

type MySQLRepository struct {
	db     *sql.DB
	logger *slog.Logger
}

var _ Repository = (*MySQLRepository)(nil)

func NewMySQLRepository(db *sql.DB, logger *slog.Logger) *MySQLRepository {
	if logger == nil {
		logger = slog.Default()
	}

	return &MySQLRepository{
		logger: logger,
		db:     db,
	}
}

// CreateBasket returns basket from the basket database.
//
// Low-level database errors are wrapped to provide extra context.
func (r *MySQLRepository) CreateBasket(ctx context.Context, basket Basket) (Basket, error) {
	args := []any{basket.BasketID, basket.Status, basket.Subtotal.CurrencyCode}

	const insertBasketQuery = `
				INSERT INTO baskets (
				 basket_id,
				 status,
				 currency_code
				)
				VALUES (?,?,?)
	`

	_, err := r.db.ExecContext(
		ctx,
		insertBasketQuery,
		args...,
	)
	if err != nil {
		r.logger.ErrorContext(ctx, "failed to insert new basket",
			"error", err,
			"basket_id", basket.BasketID,
			"basket_status", basket.Status,
			"currency_code", basket.Subtotal.CurrencyCode,
		)
		return Basket{}, fmt.Errorf("insert basket: %w", err)
	}

	createdBasket, err := r.GetBasket(ctx, string(basket.BasketID))
	if err != nil {
		r.logger.ErrorContext(ctx, "failed to insert new basket into baskets table",
			"error", err,
			"basket_id", basket.BasketID,
			"basket_status", basket.Status,
			"currency_code", basket.Subtotal.CurrencyCode,
		)
		return Basket{}, fmt.Errorf("insert basket: %w", err)
	}

	r.logger.InfoContext(ctx, "inserted new basket into baskets table",
		"basket_id", basket.BasketID,
		"basket_status", basket.Status,
		"currency_code", basket.Subtotal.CurrencyCode,
	)

	return createdBasket, nil
}

// GetBasket
func (r *MySQLRepository) GetBasket(ctx context.Context, basketID string) (Basket, error) {
	basket, err := r.getBasketRow(ctx, basketID)
	if err != nil {
		return Basket{}, err // Error logged and wrapped in getBasketRow
	}

	const getBasketItemsQuery = `
	SELECT
		basket_item_id,
		basket_id,
		product_id,
		variant_id,
		product_name_snapshot,
		variant_name_snapshot,
		quantity,
		unit_price_minor_units,
		line_total_minor_units,
		currency_code,
		created_at,
		updated_at
	FROM basket_items
	WHERE basket_id = ?
	ORDER BY created_at ASC, id ASC;
	`

	rows, err := r.db.QueryContext(ctx, getBasketItemsQuery, basketID)
	if err != nil {
		r.logger.ErrorContext(ctx, "failed to get basket items",
			"error", err,
			"basket_id", basketID,
		)
		return Basket{}, fmt.Errorf("get basket items: %w", err)
	}
	defer rows.Close()

	basketItems := make([]*BasketItem, 0)
	var subtotal int64

	for rows.Next() {
		var basketItem BasketItem

		if err := rows.Scan(
			&basketItem.BasketItemID,
			&basketItem.BasketID,
			&basketItem.ProductID,
			&basketItem.VariantID,
			&basketItem.ProductNameSnapShot,
			&basketItem.VariantNameSnapShot,
			&basketItem.Quantity,
			&basketItem.UnitPrice,
			&basketItem.LineTotal.AmountMinor,
			&basketItem.CurrencyCode,
			&basketItem.AddedAt,
			&basketItem.UpdatedAt,
		); err != nil {
			r.logger.ErrorContext(ctx, "failed to get row basket item",
				"error", err,
				"basket_id", basketID,
			)
			return Basket{}, fmt.Errorf("get row basket item: %w", err)
		}

		subtotal += basketItem.LineTotal.AmountMinor

		basketItems = append(basketItems, &basketItem)
	}

	if err := rows.Err(); err != nil {
		r.logger.ErrorContext(ctx, "failed to get basket items",
			"error", err,
			"basket_id", basketID,
		)
		return Basket{}, fmt.Errorf("iterate basket item rows: %w", err)
	}

	basket.BasketItems = basketItems
	basket.Subtotal.AmountMinor = subtotal

	return basket, nil
}

// AddItem
func (r *MySQLRepository) AddItem(ctx context.Context, query BasketQuery) (Basket, error) {

	return Basket{}, nil
}

// UpdateItemQuantity
func (r *MySQLRepository) UpdateItemQuantity(ctx context.Context, query BasketQuery) (Basket, error) {

	return Basket{}, nil
}

// RemoveItem
func (r *MySQLRepository) RemoveItem(ctx context.Context, query BasketQuery) (Basket, error) {

	return Basket{}, nil
}

// ClearBasket
func (r *MySQLRepository) ClearBasket(ctx context.Context, query BasketQuery) (Basket, error) {

	return Basket{}, nil
}

func (r *MySQLRepository) getBasketRow(ctx context.Context, basketID string) (Basket, error) {
	const getBasketQuery = `
		SELECT 
		  basket_id,
		  status,
		  currency_code,
		  created_at,
		  updated_at
		FROM baskets
		WHERE baskets_id = ?
	`
	var basket Basket

	err := r.db.QueryRowContext(
		ctx,
		getBasketQuery,
		basketID,
	).Scan(
		&basket.BasketID,
		&basket.Status,
		&basket.Subtotal.CurrencyCode,
		&basket.CreatedAt,
		&basket.UpdatedAt,
	)

	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			r.logger.ErrorContext(ctx, "failed to get basket",
				"error", ErrBasketNotFound,
				"basket_id", basketID,
			)
			return Basket{}, ErrBasketNotFound

		default:
			r.logger.ErrorContext(ctx, "failed to get basket",
				"error", err,
				"basket_id", basketID,
			)
			return basket, fmt.Errorf("query basket_id %q: %w", basketID, err)

		}
	}

	r.logger.InfoContext(ctx, "retrieved basket",
		"basket_id", basketID,
		"status", basket.Status,
		"created_at", basket.CreatedAt,
		"subtotal", basket.Subtotal.AmountMinor,
		"currencycode", basket.Subtotal.CurrencyCode,
	)

	return basket, nil
}
