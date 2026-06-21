package basket

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/go-sql-driver/mysql"
)

// Repository defines Basket Service persistence behaviour.
type Repository interface {
	CreateBasket(ctx context.Context, basket Basket) (Basket, error)
	GetBasket(ctx context.Context, basketID string) (Basket, error)
	AddItem(ctx context.Context, cmd AddValidatedItemCommand) (Basket, error)
	UpdateItemQuantity(ctx context.Context, cmd UpdateItemQuantityCommand) (Basket, error)
	RemoveItem(ctx context.Context, cmd RemoveItemCommand) (Basket, error)
	ClearBasket(ctx context.Context, cmd ClearBasketCommand) (Basket, error)
}

type MySQLRepository struct {
	db     *sql.DB
	logger *slog.Logger
}

var _ Repository = (*MySQLRepository)(nil)

func NewMySQLRepository(db *sql.DB, logger *slog.Logger) *MySQLRepository {
	if db == nil {
		panic("basket: nil database")
	}

	if logger == nil {
		logger = slog.Default()
	}

	return &MySQLRepository{
		db:     db,
		logger: logger.With("component", "basket_repository"),
	}
}

// CreateBasket persists and returns a new basket.
func (r *MySQLRepository) CreateBasket(ctx context.Context, basket Basket) (Basket, error) {
	const insertBasketQuery = `
		INSERT INTO baskets (
			basket_id,
			status,
			currency_code
		)
		VALUES (?, ?, ?)
	`

	_, err := r.db.ExecContext(
		ctx,
		insertBasketQuery,
		basket.BasketID,
		basket.Status,
		basket.Subtotal.CurrencyCode,
	)
	if err != nil {
		r.logger.ErrorContext(ctx, "failed to insert basket",
			"error", err,
			"basket_id", basket.BasketID,
			"basket_status", basket.Status,
			"currency_code", basket.Subtotal.CurrencyCode,
		)

		return Basket{}, fmt.Errorf("insert basket: %w", err)
	}

	createdBasket, err := r.GetBasket(ctx, string(basket.BasketID))
	if err != nil {
		r.logger.ErrorContext(ctx, "failed to load basket after insert",
			"error", err,
			"basket_id", basket.BasketID,
		)

		return Basket{}, fmt.Errorf("load basket after insert: %w", err)
	}

	r.logger.InfoContext(ctx, "basket inserted",
		"basket_id", createdBasket.BasketID,
		"basket_status", createdBasket.Status,
		"currency_code", createdBasket.Subtotal.CurrencyCode,
	)

	return createdBasket, nil
}

// GetBasket loads one basket and its items.
func (r *MySQLRepository) GetBasket(ctx context.Context, basketID string) (Basket, error) {
	basket, err := r.getBasketRow(ctx, basketID)
	if err != nil {
		return Basket{}, err
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
		ORDER BY created_at ASC, id ASC
	`

	rows, err := r.db.QueryContext(ctx, getBasketItemsQuery, basketID)
	if err != nil {
		r.logger.ErrorContext(ctx, "failed to query basket items",
			"error", err,
			"basket_id", basketID,
		)

		return Basket{}, fmt.Errorf("query basket items: %w", err)
	}
	defer rows.Close()

	basketItems := make([]*BasketItem, 0)
	var subtotal int64

	for rows.Next() {
		var item BasketItem
		var unitPriceMinorUnits int64
		var lineTotalMinorUnits int64
		var currencyCode string

		if err := rows.Scan(
			&item.BasketItemID,
			&item.BasketID,
			&item.ProductID,
			&item.VariantID,
			&item.ProductNameSnapshot,
			&item.VariantNameSnapshot,
			&item.Quantity,
			&unitPriceMinorUnits,
			&lineTotalMinorUnits,
			&currencyCode,
			&item.AddedAt,
			&item.UpdatedAt,
		); err != nil {
			r.logger.ErrorContext(ctx, "failed to scan basket item row",
				"error", err,
				"basket_id", basketID,
			)

			return Basket{}, fmt.Errorf("scan basket item row: %w", err)
		}

		item.UnitPrice = Money{
			AmountMinor:  unitPriceMinorUnits,
			CurrencyCode: CurrencyCode(currencyCode),
		}
		item.LineTotal = Money{
			AmountMinor:  lineTotalMinorUnits,
			CurrencyCode: CurrencyCode(currencyCode),
		}

		subtotal += item.LineTotal.AmountMinor
		basketItems = append(basketItems, &item)
	}

	if err := rows.Err(); err != nil {
		r.logger.ErrorContext(ctx, "failed to iterate basket item rows",
			"error", err,
			"basket_id", basketID,
		)

		return Basket{}, fmt.Errorf("iterate basket item rows: %w", err)
	}

	basket.BasketItems = basketItems
	basket.Subtotal.AmountMinor = subtotal

	r.logger.DebugContext(ctx, "basket loaded",
		"basket_id", basketID,
		"item_count", len(basketItems),
		"subtotal_minor_units", subtotal,
		"currency_code", basket.Subtotal.CurrencyCode,
	)

	return basket, nil
}

// AddItem adds a basket item to an existing basket, or increases the existing
// item quantity for the same product/variant pair.
func (r *MySQLRepository) AddItem(ctx context.Context, cmd AddValidatedItemCommand) (Basket, error) {
	for attempt := 1; attempt <= maxIDGenerationAttempts; attempt++ {
		updatedBasket, err := r.addItemOnce(ctx, cmd)
		if err == nil {
			return updatedBasket, nil
		}

		if errors.Is(err, ErrCouldNotAllocateItemID) {
			r.logger.WarnContext(ctx, "basket item id collision detected; retrying",
				"basket_id", cmd.BasketID,
				"product_id", cmd.ProductID,
				"variant_id", cmd.VariantID,
				"attempt", attempt,
			)

			continue
		}

		return Basket{}, err
	}

	return Basket{}, ErrCouldNotAllocateItemID
}

func (r *MySQLRepository) UpdateItemQuantity(ctx context.Context, cmd UpdateItemQuantityCommand) (Basket, error) {
	if cmd.Quantity < minBasketQuantity || cmd.Quantity > maxBasketQuantity {
		return Basket{}, ErrInvalidQuantity
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		r.logger.ErrorContext(ctx, "failed to begin update item quantity transaction",
			"error", err,
			"basket_id", cmd.BasketID,
			"basket_item_id", cmd.BasketItemID,
		)

		return Basket{}, fmt.Errorf("begin update item quantity transaction: %w", err)
	}

	committed := false
	defer func() {
		if committed {
			return
		}

		if rollbackErr := tx.Rollback(); rollbackErr != nil {
			r.logger.ErrorContext(ctx, "failed to roll back update item quantity transaction",
				"error", rollbackErr,
				"basket_id", cmd.BasketID,
				"basket_item_id", cmd.BasketItemID,
			)
		}
	}()

	if _, err := r.lockBasketForUpdate(ctx, tx, cmd.BasketID); err != nil {
		return Basket{}, err
	}

	item, err := r.lockBasketItemForUpdate(ctx, tx, cmd.BasketID, cmd.BasketItemID)
	if err != nil {
		return Basket{}, err
	}

	if err := r.updateBasketItemQuantity(ctx, tx, cmd, item); err != nil {
		return Basket{}, err
	}

	if err := r.touchBasket(ctx, tx, cmd.BasketID); err != nil {
		return Basket{}, err
	}

	if err := tx.Commit(); err != nil {
		r.logger.ErrorContext(ctx, "failed to commit update item quantity transaction",
			"error", err,
			"basket_id", cmd.BasketID,
			"basket_item_id", cmd.BasketItemID,
			"quantity", cmd.Quantity,
		)

		return Basket{}, fmt.Errorf("commit update item quantity transaction: %w", err)
	}

	committed = true

	r.logger.DebugContext(ctx, "basket item quantity updated",
		"basket_id", cmd.BasketID,
		"basket_item_id", cmd.BasketItemID,
		"quantity", cmd.Quantity,
	)

	updatedBasket, err := r.GetBasket(ctx, cmd.BasketID)
	if err != nil {
		return Basket{}, fmt.Errorf("load basket after updating item quantity: %w", err)
	}

	return updatedBasket, nil
}

func (r *MySQLRepository) RemoveItem(ctx context.Context, cmd RemoveItemCommand) (Basket, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		r.logger.ErrorContext(ctx, "failed to begin remove item transaction",
			"error", err,
			"basket_id", cmd.BasketID,
			"basket_item_id", cmd.BasketItemID,
		)

		return Basket{}, fmt.Errorf("begin remove item transaction: %w", err)
	}

	committed := false
	defer func() {
		if committed {
			return
		}

		if rollbackErr := tx.Rollback(); rollbackErr != nil {
			r.logger.ErrorContext(ctx, "failed to roll back remove item transaction",
				"error", rollbackErr,
				"basket_id", cmd.BasketID,
				"basket_item_id", cmd.BasketItemID,
			)
		}
	}()

	if _, err := r.lockBasketForUpdate(ctx, tx, cmd.BasketID); err != nil {
		return Basket{}, err
	}

	if err := r.deleteBasketItem(ctx, tx, cmd); err != nil {
		return Basket{}, err
	}

	if err := r.touchBasket(ctx, tx, cmd.BasketID); err != nil {
		return Basket{}, err
	}

	if err := tx.Commit(); err != nil {
		r.logger.ErrorContext(ctx, "failed to commit remove item transaction",
			"error", err,
			"basket_id", cmd.BasketID,
			"basket_item_id", cmd.BasketItemID,
		)

		return Basket{}, fmt.Errorf("commit remove item transaction: %w", err)
	}

	committed = true

	r.logger.DebugContext(ctx, "basket item removed",
		"basket_id", cmd.BasketID,
		"basket_item_id", cmd.BasketItemID,
	)

	updatedBasket, err := r.GetBasket(ctx, cmd.BasketID)
	if err != nil {
		return Basket{}, fmt.Errorf("load basket after removing item: %w", err)
	}

	return updatedBasket, nil
}

func (r *MySQLRepository) ClearBasket(ctx context.Context, cmd ClearBasketCommand) (Basket, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		r.logger.ErrorContext(ctx, "failed to begin clear basket transaction",
			"error", err,
			"basket_id", cmd.BasketID,
		)

		return Basket{}, fmt.Errorf("begin clear basket transaction: %w", err)
	}

	committed := false
	defer func() {
		if committed {
			return
		}

		if rollbackErr := tx.Rollback(); rollbackErr != nil {
			r.logger.ErrorContext(ctx, "failed to roll back clear basket transaction",
				"error", rollbackErr,
				"basket_id", cmd.BasketID,
			)
		}
	}()

	if _, err := r.lockBasketForUpdate(ctx, tx, cmd.BasketID); err != nil {
		return Basket{}, err
	}

	deleteCount, err := r.deleteBasketItems(ctx, tx, cmd.BasketID)
	if err != nil {
		return Basket{}, err
	}

	if err := r.touchBasket(ctx, tx, cmd.BasketID); err != nil {
		return Basket{}, err
	}

	if err := tx.Commit(); err != nil {
		r.logger.ErrorContext(ctx, "failed to commit clear basket transaction",
			"error", err,
			"basket_id", cmd.BasketID,
		)

		return Basket{}, fmt.Errorf("commit clear basket transaction: %w", err)
	}

	committed = true

	r.logger.DebugContext(ctx, "basket cleared",
		"basket_id", cmd.BasketID,
		"deleted_item_count", deleteCount,
	)

	updatedBasket, err := r.GetBasket(ctx, cmd.BasketID)
	if err != nil {
		return Basket{}, fmt.Errorf("load basket after clearing: %w", err)
	}

	return updatedBasket, nil
}

func (r *MySQLRepository) addItemOnce(ctx context.Context, cmd AddValidatedItemCommand) (Basket, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		r.logger.ErrorContext(ctx, "failed to begin add item transaction",
			"error", err,
			"basket_id", cmd.BasketID,
		)

		return Basket{}, fmt.Errorf("begin add item transaction: %w", err)
	}

	committed := false
	defer func() {
		if committed {
			return
		}

		if rollbackErr := tx.Rollback(); rollbackErr != nil {
			r.logger.ErrorContext(ctx, "failed to roll back add item transaction",
				"error", rollbackErr,
				"basket_id", cmd.BasketID,
			)
		}
	}()

	basketCurrencyCode, err := r.lockBasketForUpdate(ctx, tx, cmd.BasketID)
	if err != nil {
		return Basket{}, err
	}

	if basketCurrencyCode != cmd.UnitPrice.CurrencyCode {
		r.logger.WarnContext(ctx, "basket currency mismatch",
			"error", ErrBasketCurrencyMismatch,
			"basket_id", cmd.BasketID,
			"basket_currency_code", basketCurrencyCode,
			"item_currency_code", cmd.UnitPrice.CurrencyCode,
		)

		return Basket{}, ErrBasketCurrencyMismatch
	}

	existingItem, found, err := r.findBasketItemForUpdate(ctx, tx, cmd)
	if err != nil {
		return Basket{}, err
	}

	if found {
		if err := r.updateExistingBasketItem(ctx, tx, cmd, existingItem); err != nil {
			return Basket{}, err
		}
	} else {
		if err := r.insertNewBasketItem(ctx, tx, cmd); err != nil {
			return Basket{}, err
		}
	}

	if err := r.touchBasket(ctx, tx, cmd.BasketID); err != nil {
		return Basket{}, err
	}

	if err := tx.Commit(); err != nil {
		r.logger.ErrorContext(ctx, "failed to commit add item transaction",
			"error", err,
			"basket_id", cmd.BasketID,
			"product_id", cmd.ProductID,
			"variant_id", cmd.VariantID,
		)

		return Basket{}, fmt.Errorf("commit add item transaction: %w", err)
	}

	committed = true

	r.logger.DebugContext(ctx, "add item transaction committed",
		"basket_id", cmd.BasketID,
		"product_id", cmd.ProductID,
		"variant_id", cmd.VariantID,
		"quantity", cmd.Quantity,
	)

	updatedBasket, err := r.GetBasket(ctx, cmd.BasketID)
	if err != nil {
		return Basket{}, fmt.Errorf("load basket after add item: %w", err)
	}

	return updatedBasket, nil
}

func (r *MySQLRepository) insertNewBasketItem(
	ctx context.Context,
	tx *sql.Tx,
	cmd AddValidatedItemCommand,
) error {
	basketItemID, err := NewBasketItemID()
	if err != nil {
		r.logger.ErrorContext(ctx, "failed to generate basket item id",
			"error", err,
			"basket_id", cmd.BasketID,
		)

		return fmt.Errorf("generate basket item id: %w", err)
	}

	lineTotalMinorUnits := int64(cmd.Quantity) * cmd.UnitPrice.AmountMinor

	const insertNewItemQuery = `
		INSERT INTO basket_items (
			basket_item_id,
			basket_id,
			product_id,
			variant_id,
			product_name_snapshot,
			variant_name_snapshot,
			quantity,
			unit_price_minor_units,
			line_total_minor_units,
			currency_code
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err = tx.ExecContext(
		ctx,
		insertNewItemQuery,
		basketItemID,
		cmd.BasketID,
		cmd.ProductID,
		cmd.VariantID,
		cmd.ProductNameSnapshot,
		cmd.VariantNameSnapshot,
		cmd.Quantity,
		cmd.UnitPrice.AmountMinor,
		lineTotalMinorUnits,
		cmd.UnitPrice.CurrencyCode,
	)
	if err != nil {
		if isDuplicateKey(err, "uk_basket_items_basket_item_id") {
			return ErrCouldNotAllocateItemID
		}

		r.logger.ErrorContext(ctx, "failed to insert basket item",
			"error", err,
			"basket_id", cmd.BasketID,
			"basket_item_id", basketItemID,
			"product_id", cmd.ProductID,
			"variant_id", cmd.VariantID,
		)

		return fmt.Errorf("insert basket item: %w", err)
	}

	r.logger.DebugContext(ctx, "basket item inserted",
		"basket_id", cmd.BasketID,
		"basket_item_id", basketItemID,
		"product_id", cmd.ProductID,
		"variant_id", cmd.VariantID,
		"quantity", cmd.Quantity,
	)

	return nil
}

func (r *MySQLRepository) updateExistingBasketItem(
	ctx context.Context,
	tx *sql.Tx,
	cmd AddValidatedItemCommand,
	existingItem existingBasketItem,
) error {
	newQuantity := existingItem.Quantity + cmd.Quantity
	if newQuantity < minBasketQuantity || newQuantity > maxBasketQuantity {
		return ErrInvalidQuantity
	}

	lineTotalMinorUnits := int64(newQuantity) * cmd.UnitPrice.AmountMinor

	const query = `
		UPDATE basket_items
		SET
			product_name_snapshot = ?,
			variant_name_snapshot = ?,
			quantity = ?,
			unit_price_minor_units = ?,
			line_total_minor_units = ?,
			currency_code = ?,
			updated_at = CURRENT_TIMESTAMP(6)
		WHERE basket_id = ?
		  AND basket_item_id = ?
	`

	result, err := tx.ExecContext(
		ctx,
		query,
		cmd.ProductNameSnapshot,
		cmd.VariantNameSnapshot,
		newQuantity,
		cmd.UnitPrice.AmountMinor,
		lineTotalMinorUnits,
		cmd.UnitPrice.CurrencyCode,
		cmd.BasketID,
		existingItem.ID,
	)
	if err != nil {
		r.logger.ErrorContext(ctx, "failed to update existing basket item",
			"error", err,
			"basket_id", cmd.BasketID,
			"basket_item_id", existingItem.ID,
			"product_id", cmd.ProductID,
			"variant_id", cmd.VariantID,
		)

		return fmt.Errorf("update existing basket item: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("read basket item update rows affected: %w", err)
	}

	if rowsAffected != 1 {
		return fmt.Errorf("update existing basket item: expected 1 row affected, got %d", rowsAffected)
	}

	r.logger.DebugContext(ctx, "existing basket item quantity increased",
		"basket_id", cmd.BasketID,
		"basket_item_id", existingItem.ID,
		"product_id", cmd.ProductID,
		"variant_id", cmd.VariantID,
		"added_quantity", cmd.Quantity,
		"new_quantity", newQuantity,
	)

	return nil
}

func (r *MySQLRepository) updateBasketItemQuantity(
	ctx context.Context,
	tx *sql.Tx,
	cmd UpdateItemQuantityCommand,
	item lockedBasketItem,
) error {
	lineTotalMinorUnits := int64(cmd.Quantity) * item.UnitPriceMinorUnits

	const updateItemQuery = `
		UPDATE basket_items
		SET
			quantity = ?,
			line_total_minor_units = ?,
			updated_at = CURRENT_TIMESTAMP(6)
		WHERE basket_id = ?
		  AND basket_item_id = ?
	`

	result, err := tx.ExecContext(
		ctx,
		updateItemQuery,
		cmd.Quantity,
		lineTotalMinorUnits,
		cmd.BasketID,
		cmd.BasketItemID,
	)
	if err != nil {
		r.logger.ErrorContext(ctx, "failed to update basket item quantity",
			"error", err,
			"basket_id", cmd.BasketID,
			"basket_item_id", cmd.BasketItemID,
			"quantity", cmd.Quantity,
			"line_total_minor_units", lineTotalMinorUnits,
		)

		return fmt.Errorf("update basket item quantity: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("read update basket item quantity rows affected: %w", err)
	}

	if rowsAffected != 1 {
		return fmt.Errorf("update basket item quantity: expected 1 row affected, got %d", rowsAffected)
	}

	return nil
}

func (r *MySQLRepository) touchBasket(ctx context.Context, tx *sql.Tx, basketID string) error {
	const query = `
		UPDATE baskets
		SET updated_at = CURRENT_TIMESTAMP(6)
		WHERE basket_id = ?
	`

	result, err := tx.ExecContext(ctx, query, basketID)
	if err != nil {
		r.logger.ErrorContext(ctx, "failed to touch basket updated_at",
			"error", err,
			"basket_id", basketID,
		)

		return fmt.Errorf("touch basket updated_at: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("read touch basket rows affected: %w", err)
	}

	if rowsAffected != 1 {
		return fmt.Errorf("touch basket: expected 1 row affected, got %d", rowsAffected)
	}

	return nil
}

func (r *MySQLRepository) lockBasketItemForUpdate(
	ctx context.Context,
	tx *sql.Tx,
	basketID string,
	basketItemID string,
) (lockedBasketItem, error) {
	const query = `
		SELECT
			basket_item_id,
			unit_price_minor_units,
			currency_code
		FROM basket_items
		WHERE basket_id = ?
		  AND basket_item_id = ?
		FOR UPDATE
	`

	var item lockedBasketItem
	var currencyCode string

	err := tx.QueryRowContext(
		ctx,
		query,
		basketID,
		basketItemID,
	).Scan(
		&item.ID,
		&item.UnitPriceMinorUnits,
		&currencyCode,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return lockedBasketItem{}, ErrBasketItemNotFound
		}

		r.logger.ErrorContext(ctx, "failed to lock basket item",
			"error", err,
			"basket_id", basketID,
			"basket_item_id", basketItemID,
		)

		return lockedBasketItem{}, fmt.Errorf("lock basket item: %w", err)
	}

	item.CurrencyCode = CurrencyCode(currencyCode)

	return item, nil
}

func (r *MySQLRepository) deleteBasketItem(
	ctx context.Context,
	tx *sql.Tx,
	cmd RemoveItemCommand,
) error {
	const deleteItemQuery = `
		DELETE FROM basket_items
		WHERE basket_id = ?
		  AND basket_item_id = ?
	`

	result, err := tx.ExecContext(
		ctx,
		deleteItemQuery,
		cmd.BasketID,
		cmd.BasketItemID,
	)
	if err != nil {
		r.logger.ErrorContext(ctx, "failed to delete basket item",
			"error", err,
			"basket_id", cmd.BasketID,
			"basket_item_id", cmd.BasketItemID,
		)

		return fmt.Errorf("delete basket item: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("read delete basket item rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return ErrBasketItemNotFound
	}

	if rowsAffected != 1 {
		return fmt.Errorf("delete basket item: expected 1 row affected, got %d", rowsAffected)
	}

	return nil
}

func (r *MySQLRepository) deleteBasketItems(ctx context.Context, tx *sql.Tx, basketID string) (int64, error) {
	const query = `
		DELETE FROM basket_items
		WHERE basket_id = ?
	`

	result, err := tx.ExecContext(ctx, query, basketID)
	if err != nil {
		r.logger.ErrorContext(ctx, "failed to delete basket items",
			"error", err,
			"basket_id", basketID,
		)

		return 0, fmt.Errorf("delete basket items: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("read clear basket rows affected: %w", err)
	}

	return rowsAffected, nil
}

func (r *MySQLRepository) findBasketItemForUpdate(
	ctx context.Context,
	tx *sql.Tx,
	cmd AddValidatedItemCommand,
) (existingBasketItem, bool, error) {
	const query = `
		SELECT
			basket_item_id,
			quantity
		FROM basket_items
		WHERE basket_id = ?
		  AND product_id = ?
		  AND variant_id = ?
		FOR UPDATE
	`

	var item existingBasketItem

	err := tx.QueryRowContext(
		ctx,
		query,
		cmd.BasketID,
		cmd.ProductID,
		cmd.VariantID,
	).Scan(
		&item.ID,
		&item.Quantity,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return existingBasketItem{}, false, nil
		}

		r.logger.ErrorContext(ctx, "failed to find basket item for update",
			"error", err,
			"basket_id", cmd.BasketID,
			"product_id", cmd.ProductID,
			"variant_id", cmd.VariantID,
		)

		return existingBasketItem{}, false, fmt.Errorf("find basket item for update: %w", err)
	}

	return item, true, nil
}

func (r *MySQLRepository) lockBasketForUpdate(
	ctx context.Context,
	tx *sql.Tx,
	basketID string,
) (CurrencyCode, error) {
	const query = `
		SELECT
			status,
			currency_code
		FROM baskets
		WHERE basket_id = ?
		FOR UPDATE
	`

	var rawStatus string
	var currencyCode string

	err := tx.QueryRowContext(ctx, query, basketID).Scan(
		&rawStatus,
		&currencyCode,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			r.logger.DebugContext(ctx, "failed to lock basket: basket not found",
				"basket_id", basketID,
			)

			return "", ErrBasketNotFound
		}

		r.logger.ErrorContext(ctx, "failed to lock basket",
			"error", err,
			"basket_id", basketID,
		)

		return "", fmt.Errorf("lock basket: %w", err)
	}

	status, err := ParseToBasketStatus(rawStatus)
	if err != nil {
		return "", fmt.Errorf("parse basket status %q: %w", rawStatus, err)
	}

	if status != BasketStatusActive {
		r.logger.WarnContext(ctx, "basket is not modifiable",
			"basket_id", basketID,
			"status", status,
			"currency_code", currencyCode,
		)

		return "", ErrBasketNotModifiable
	}

	r.logger.DebugContext(ctx, "basket locked",
		"basket_id", basketID,
		"status", status,
		"currency_code", currencyCode,
	)

	return CurrencyCode(currencyCode), nil
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
		WHERE basket_id = ?
	`

	var basket Basket
	var rawStatus string
	var currencyCode string

	err := r.db.QueryRowContext(
		ctx,
		getBasketQuery,
		basketID,
	).Scan(
		&basket.BasketID,
		&rawStatus,
		&currencyCode,
		&basket.CreatedAt,
		&basket.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			r.logger.DebugContext(ctx, "basket not found",
				"basket_id", basketID,
			)

			return Basket{}, ErrBasketNotFound
		}

		r.logger.ErrorContext(ctx, "failed to query basket",
			"error", err,
			"basket_id", basketID,
		)

		return Basket{}, fmt.Errorf("query basket_id %q: %w", basketID, err)
	}

	status, err := ParseToBasketStatus(rawStatus)
	if err != nil {
		return Basket{}, fmt.Errorf("parse basket status %q: %w", rawStatus, err)
	}

	basket.Status = status
	basket.Subtotal.CurrencyCode = CurrencyCode(currencyCode)

	r.logger.DebugContext(ctx, "basket row loaded",
		"basket_id", basketID,
		"status", basket.Status,
		"created_at", basket.CreatedAt,
		"currency_code", basket.Subtotal.CurrencyCode,
	)

	return basket, nil
}

func isDuplicateKey(err error, keyName string) bool {
	var mysqlErr *mysql.MySQLError
	if !errors.As(err, &mysqlErr) {
		return false
	}

	if mysqlErr.Number != 1062 {
		return false
	}

	return strings.Contains(mysqlErr.Message, keyName)
}
