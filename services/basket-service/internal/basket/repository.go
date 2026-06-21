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
			&basketItem.ProductNameSnapshot,
			&basketItem.VariantNameSnapshot,
			&basketItem.Quantity,
			&basketItem.UnitPrice,
			&basketItem.LineTotal.AmountMinor,
			&basketItem.UnitPrice.CurrencyCode,
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

// AddItem adds a basket item to an existing basket.
//
// If successful it returns the updated basket and a nil error. If it fails it
// returns an empty basket and an error.
func (r *MySQLRepository) AddItem(ctx context.Context, cmd AddValidatedItemCommand) (Basket, error) {
	for attempt := 1; attempt <= maxIDGenerationAttempts; attempt++ {
		updatedBasket, err := r.addItemOnce(ctx, cmd)
		if err == nil {
			return updatedBasket, nil
		}

		if errors.Is(err, ErrCouldNotAllocateItemID) {
			r.logger.WarnContext(ctx, "basket item id collision detedted; retrying",
				"basket_id", cmd.BasketID,
				"product_id", cmd.ProductID,
				"variant_id", cmd.VariantID,
				"attempt", attempt,
			)

			continue
		}

		return Basket{}, err
	}

	return Basket{}, nil
}

// UpdateItemQuantity
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

// RemoveItem
func (r *MySQLRepository) RemoveItem(ctx context.Context, cmd RemoveItemCommand) (Basket, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		r.logger.ErrorContext(ctx, "failed to begin remove item transaction",
			"error", err,
			"basket_id", cmd.BasketID,
			"basket_item_id", cmd.BasketItemID,
		)
		return Basket{}, fmt.Errorf("begin remove item transaction; %w", err)
	}

	committed := false
	defer func() {
		if committed {
			return
		}

		if rollbackErr := tx.Rollback(); rollbackErr != nil {
			r.logger.ErrorContext(ctx, "failed to roll back remove item transaction",
				"error", rollbackErr,
				"basked_id", cmd.BasketID,
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

// ClearBasket
func (r *MySQLRepository) ClearBasket(ctx context.Context, cmd ClearBasketCommand) (Basket, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		r.logger.ErrorContext(ctx, "failed to begin clear basket transaction",
			"error", err,
			"basket_id", cmd.BasketID,
		)

		return Basket{}, fmt.Errorf("begin clear basket transaction: %w", err)
	}

	commited := false
	defer func() {
		if commited {
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

	commited = true

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

func (r *MySQLRepository) deleteBasketItems(
	ctx context.Context,
	tx *sql.Tx,
	basketID string,
) (int64, error) {
	const query = `
		DELETE FROM basket_items
		WHERE basket_id =?
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

func (r *MySQLRepository) updateBasketItemQuantity(
	ctx context.Context,
	tx *sql.Tx,
	query UpdateItemQuantityCommand,
	item lockedBasketItem,
) error {
	lineTotalMinorUnits := int64(query.Quantity) * item.UnitPriceMinorUnits

	const updateItemQuery = `
		UPDATE basket_items
		SET
			quantity = ?,
			line_total_minor_units = ?,
			updated_at = CURRENT_TIMESTAMP(6)
		WHERE basket_id = ?
			AND basket_item_id = ?
	`
	args := []any{
		query.Quantity,
		lineTotalMinorUnits,
		query.BasketID,
		query.BasketItemID,
	}

	result, err := tx.ExecContext(
		ctx,
		updateItemQuery,
		args...,
	)
	if err != nil {
		r.logger.ErrorContext(ctx, "failed to update basket item quantity",
			"error", err,
			"basket_id", query.BasketID,
			"basket_item_id", query.BasketItemID,
			"quantity", query.Quantity,
			"line_total_minor_units", lineTotalMinorUnits,
		)

		return fmt.Errorf("update basket item quantity: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("read update basket item quantity rows affected: %w", err)
	}

	if rowsAffected != 1 {
		return fmt.Errorf(
			"update basket item quantity: expected 1 row affected, got %d",
			rowsAffected,
		)
	}
	return nil
}

func (r *MySQLRepository) lockBasketItemForUpdate(
	ctx context.Context,
	tx *sql.Tx,
	basketID string,
	BasketItemID string,
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

	err := tx.QueryRowContext(
		ctx,
		query,
		basketID,
		BasketItemID,
	).Scan(
		&item.ID,
		&item.UnitPriceMinorUnits,
		&item.CurrencyCode,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return lockedBasketItem{}, ErrBasketItemNotFound
		}

		r.logger.ErrorContext(ctx, "failed to lock basket item for quantity update",
			"error", err,
			"basket_id", basketID,
			"basket_item_id", BasketItemID,
		)

		return lockedBasketItem{}, fmt.Errorf("lock basket item for quantity update: %w", err)
	}

	return item, nil

}

func (r *MySQLRepository) deleteBasketItem(
	ctx context.Context,
	tx *sql.Tx,
	query RemoveItemCommand,
) error {
	const deleteItemQuery = `
		DELETE FROM basket_items
		WHERE basket_id = ?
		  AND basket_item_id = ?
	`

	args := []any{query.BasketID, query.BasketItemID}

	result, err := tx.ExecContext(
		ctx,
		deleteItemQuery,
		args...,
	)
	if err != nil {
		r.logger.ErrorContext(ctx, "failed to delete basket item",
			"error", err,
			"basket_id", query.BasketID,
			"basket_item_id", query.BasketItemID,
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
		return fmt.Errorf(
			"delete basket item: expected 1 row affected, got %d",
			rowsAffected,
		)
	}

	return nil
}

func (r *MySQLRepository) addItemOnce(ctx context.Context, cmd AddValidatedItemCommand) (Basket, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		r.logger.ErrorContext(ctx, "failed to begin add item transaction",
			"error", err,
			"basked_id", cmd.BasketID,
		)

		return Basket{}, fmt.Errorf("begin add item transaction: %w", err)
	}

	commited := false
	defer func() {
		if commited {
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

	if basketCurrencyCode != CurrencyCode(cmd.UnitPrice.CurrencyCode) {
		r.logger.ErrorContext(ctx, "basket currency mismatch",
			"error", ErrBasketCurrencyMismatch,
			"basket_id", cmd.BasketID,
			"db_basket_currency_code", basketCurrencyCode,
			"basket_currency_code", cmd.UnitPrice.CurrencyCode,
		)

		return Basket{}, ErrBasketCurrencyMismatch
	}

	existingItem, found, err := r.findBasketItemForUpdate(ctx, tx, cmd)
	if err != nil {
		r.logger.ErrorContext(ctx, "basket currency mismatch",
			"error", ErrBasketCurrencyMismatch,
			"basket_id", cmd.BasketID,
			"db_basket_currency_code", basketCurrencyCode,
			"basket_currency_code", cmd.UnitPrice.CurrencyCode,
		)

		return Basket{}, err
	}

	if found {
		if err := r.updateExistingBasketItem(ctx, tx, cmd, existingItem); err != nil {
			return Basket{}, err
		}

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

	commited = true

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
	basketItemID, err := NewBasketID()
	if err != nil {
		r.logger.ErrorContext(ctx, "generate basket item id err",
			"error", err,
			"basket_id", cmd.BasketID,
		)
		return fmt.Errorf("generate basket item id: %w", err)
	}

	lineTotalMinorUnits := int64(cmd.Quantity) * cmd.UnitPrice.AmountMinor

	const insertNewItemQuery = `
		INSERT INTO basket_items(
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
		VALUES (?,?,?,?,?,?,?,?,?,?)
	`

	args := []any{
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
	}

	_, err = tx.ExecContext(
		ctx,
		insertNewItemQuery,
		args...,
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
			updated_at = CURRRENT_TIMESTAMP(6)
		WHERE basket_item_id = ?
	`

	args := []any{
		cmd.ProductNameSnapshot,
		cmd.VariantNameSnapshot,
		cmd.Quantity,
		cmd.UnitPrice.AmountMinor,
		lineTotalMinorUnits,
		cmd.UnitPrice.CurrencyCode,
		cmd.BasketID,
	}

	result, err := tx.ExecContext(
		ctx,
		query,
		args...,
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
		return fmt.Errorf("update existing basket items; expected 1 row affected, go %d", rowsAffected)
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

func (r *MySQLRepository) touchBasket(
	ctx context.Context,
	tx *sql.Tx,
	basketID string,
) error {
	const query = `
		UPDATE baskets
		SET updated_at = CURRENT_TIMESTAMP(6)
		WHERE basket_id = ?
	`

	result, err := tx.ExecContext(ctx, query, basketID)
	if err != nil {
		r.logger.ErrorContext(ctx, "failed to touch basket update_at",
			"error", err,
			"basket_id", basketID,
		)

		return fmt.Errorf("touch basket update_at: %w", err)
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

func isDuplicateKey(err error, keyName string) bool {
	var mysqlErr *mysql.MySQLError
	if !errors.As(err, &mysqlErr) {
		return false
	}

	if mysqlErr.Number != 1062 { // mysql Error 1062: Duplicate entry
		return false
	}

	return strings.Contains(mysqlErr.Message, keyName)
}

func (r *MySQLRepository) findBasketItemForUpdate(ctx context.Context, tx *sql.Tx, cmd AddValidatedItemCommand) (
	existingBasketItem,
	bool,
	error) {
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

	args := []any{cmd.BasketID, cmd.ProductID, cmd.VariantID}

	err := tx.QueryRowContext(
		ctx,
		query,
		args...,
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
			"basked_id", cmd.BasketID,
			"product_id", cmd.ProductID,
			"variant_id", cmd.VariantID,
		)

		return existingBasketItem{}, false, fmt.Errorf("find basket item for update: %w", err)
	}

	return item, true, nil
}

func (r *MySQLRepository) lockBasketForUpdate(ctx context.Context, tx *sql.Tx, basketID string) (
	CurrencyCode, error) {
	const query = `
			SELECT
				status,
				currency_code
			FROM baskets
			WHERE basket_id = ?
			FOR UPDATE
		`

	var status string
	var currencyCode string

	err := tx.QueryRowContext(ctx, query, basketID).Scan(
		&status,
		&currencyCode,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			r.logger.ErrorContext(ctx, "failed to lock basket for add item: basket not found.",
				"error", err,
				"basket_id", basketID,
			)

			return CurrencyCode(""), ErrBasketNotFound
		}

		r.logger.ErrorContext(ctx, "failed to lock basket for add item",
			"error", err,
			"basket_id", basketID,
		)

		return CurrencyCode(""), fmt.Errorf("lock basket for add item item: %w", err)
	}

	if BasketStatus(status) != BasketStatusActive {
		r.logger.ErrorContext(ctx, "failed to lock basket for add item",
			"error", err,
			"basket_id", basketID,
		)
		return CurrencyCode(""), ErrBasketNotModifiable
	}

	r.logger.InfoContext(ctx, "locked basket for add item",
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
