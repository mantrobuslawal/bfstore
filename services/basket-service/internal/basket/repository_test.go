package basket

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

func newTestRepository(t *testing.T) (*MySQLRepository, sqlmock.Sqlmock, func()) {
	t.Helper()

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New() error = %v", err)
	}

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	return NewMySQLRepository(db, logger), mock, func() { _ = db.Close() }
}

func TestMySQLRepositoryGetBasketScansMoneyCorrectly(t *testing.T) {
	t.Parallel()

	repo, mock, cleanup := newTestRepository(t)
	defer cleanup()

	now := time.Now().UTC()

	expectBasketRow(mock, "basket_test", "ACTIVE", "GBP", now)

	mock.ExpectQuery(regexp.QuoteMeta(`
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
	`)).
		WithArgs("basket_test").
		WillReturnRows(sqlmock.NewRows([]string{
			"basket_item_id",
			"basket_id",
			"product_id",
			"variant_id",
			"product_name_snapshot",
			"variant_name_snapshot",
			"quantity",
			"unit_price_minor_units",
			"line_total_minor_units",
			"currency_code",
			"created_at",
			"updated_at",
		}).AddRow(
			"bitem_test",
			"basket_test",
			"prod_01ARZ3NDEKTSV4RRFFQ69G5FAV",
			"var_01ARZ3NDEKTSV4RRFFQ69G5FAV",
			"Go Gopher Mug",
			"Blue",
			2,
			int64(1299),
			int64(2598),
			"GBP",
			now,
			now,
		))

	got, err := repo.GetBasket(context.Background(), "basket_test")
	if err != nil {
		t.Fatalf("GetBasket() error = %v, want nil", err)
	}

	if got.Status != BasketStatusActive {
		t.Fatalf("Status = %q, want %q", got.Status, BasketStatusActive)
	}

	if got.Subtotal.AmountMinor != 2598 {
		t.Fatalf("Subtotal.AmountMinor = %d, want 2598", got.Subtotal.AmountMinor)
	}

	if got.Subtotal.CurrencyCode != "GBP" {
		t.Fatalf("Subtotal.CurrencyCode = %q, want GBP", got.Subtotal.CurrencyCode)
	}

	if len(got.BasketItems) != 1 {
		t.Fatalf("len(BasketItems) = %d, want 1", len(got.BasketItems))
	}

	item := got.BasketItems[0]
	if item.UnitPrice.AmountMinor != 1299 {
		t.Fatalf("UnitPrice.AmountMinor = %d, want 1299", item.UnitPrice.AmountMinor)
	}

	if item.UnitPrice.CurrencyCode != "GBP" {
		t.Fatalf("UnitPrice.CurrencyCode = %q, want GBP", item.UnitPrice.CurrencyCode)
	}

	if item.LineTotal.AmountMinor != 2598 {
		t.Fatalf("LineTotal.AmountMinor = %d, want 2598", item.LineTotal.AmountMinor)
	}

	if item.LineTotal.CurrencyCode != "GBP" {
		t.Fatalf("LineTotal.CurrencyCode = %q, want GBP", item.LineTotal.CurrencyCode)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestMySQLRepositoryLockBasketForUpdateRejectsInactiveBasket(t *testing.T) {
	t.Parallel()

	repo, mock, cleanup := newTestRepository(t)
	defer cleanup()

	mock.ExpectBegin()
	tx, err := repo.db.BeginTx(context.Background(), nil)
	if err != nil {
		t.Fatalf("BeginTx() error = %v, want nil", err)
	}
	defer tx.Rollback()

	expectLockBasketForUpdate(mock, "basket_test", "CHECKED_OUT", "GBP")

	_, err = repo.lockBasketForUpdate(context.Background(), tx, "basket_test")
	if !errors.Is(err, ErrBasketNotModifiable) {
		t.Fatalf("lockBasketForUpdate() error = %v, want %v", err, ErrBasketNotModifiable)
	}

	mock.ExpectRollback()
	if err := tx.Rollback(); err != nil {
		t.Fatalf("Rollback() error = %v, want nil", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestMySQLRepositoryAddItemInsertsNewItemWhenNotFound(t *testing.T) {
	t.Parallel()

	repo, mock, cleanup := newTestRepository(t)
	defer cleanup()

	now := time.Now().UTC()

	mock.ExpectBegin()
	expectLockBasketForUpdate(mock, "basket_test", "ACTIVE", "GBP")
	expectFindBasketItemForUpdateNotFound(mock, "basket_test", "prod_01ARZ3NDEKTSV4RRFFQ69G5FAV", "var_01ARZ3NDEKTSV4RRFFQ69G5FAV")

	mock.ExpectExec(regexp.QuoteMeta(`
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
	`)).
		WithArgs(
			sqlmock.AnyArg(),
			"basket_test",
			"prod_01ARZ3NDEKTSV4RRFFQ69G5FAV",
			"var_01ARZ3NDEKTSV4RRFFQ69G5FAV",
			"Go Gopher Mug",
			"Blue",
			2,
			int64(1299),
			int64(2598),
			"GBP",
		).
		WillReturnResult(sqlmock.NewResult(1, 1))

	expectTouchBasket(mock, "basket_test")
	mock.ExpectCommit()

	expectBasketRow(mock, "basket_test", "ACTIVE", "GBP", now)
	expectBasketItems(mock, "basket_test", nil)

	got, err := repo.AddItem(context.Background(), AddValidatedItemCommand{
		BasketID:            "basket_test",
		ProductID:           "prod_01ARZ3NDEKTSV4RRFFQ69G5FAV",
		VariantID:           "var_01ARZ3NDEKTSV4RRFFQ69G5FAV",
		ProductNameSnapshot: "Go Gopher Mug",
		VariantNameSnapshot: "Blue",
		Quantity:            2,
		UnitPrice:           Money{AmountMinor: 1299, CurrencyCode: "GBP"},
	})
	if err != nil {
		t.Fatalf("AddItem() error = %v, want nil", err)
	}

	if got.BasketID != BasketID("basket_test") {
		t.Fatalf("BasketID = %q, want basket_test", got.BasketID)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestMySQLRepositoryAddItemUpdatesExistingItemWithoutInsert(t *testing.T) {
	t.Parallel()

	repo, mock, cleanup := newTestRepository(t)
	defer cleanup()

	now := time.Now().UTC()

	mock.ExpectBegin()
	expectLockBasketForUpdate(mock, "basket_test", "ACTIVE", "GBP")
	expectFindBasketItemForUpdateFound(mock, "basket_test", "prod_01ARZ3NDEKTSV4RRFFQ69G5FAV", "var_01ARZ3NDEKTSV4RRFFQ69G5FAV", "bitem_test", 3)

	mock.ExpectExec(regexp.QuoteMeta(`
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
	`)).
		WithArgs(
			"Go Gopher Mug",
			"Blue",
			5,
			int64(1299),
			int64(6495),
			"GBP",
			"basket_test",
			"bitem_test",
		).
		WillReturnResult(sqlmock.NewResult(0, 1))

	expectTouchBasket(mock, "basket_test")
	mock.ExpectCommit()

	expectBasketRow(mock, "basket_test", "ACTIVE", "GBP", now)
	expectBasketItems(mock, "basket_test", nil)

	_, err := repo.AddItem(context.Background(), AddValidatedItemCommand{
		BasketID:            "basket_test",
		ProductID:           "prod_01ARZ3NDEKTSV4RRFFQ69G5FAV",
		VariantID:           "var_01ARZ3NDEKTSV4RRFFQ69G5FAV",
		ProductNameSnapshot: "Go Gopher Mug",
		VariantNameSnapshot: "Blue",
		Quantity:            2,
		UnitPrice:           Money{AmountMinor: 1299, CurrencyCode: "GBP"},
	})
	if err != nil {
		t.Fatalf("AddItem() error = %v, want nil", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestMySQLRepositoryUpdateItemQuantityUsesStoredUnitPrice(t *testing.T) {
	t.Parallel()

	repo, mock, cleanup := newTestRepository(t)
	defer cleanup()

	now := time.Now().UTC()

	mock.ExpectBegin()
	expectLockBasketForUpdate(mock, "basket_test", "ACTIVE", "GBP")

	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT
			basket_item_id,
			unit_price_minor_units,
			currency_code
		FROM basket_items
		WHERE basket_id = ?
		  AND basket_item_id = ?
		FOR UPDATE
	`)).
		WithArgs("basket_test", "bitem_test").
		WillReturnRows(sqlmock.NewRows([]string{
			"basket_item_id",
			"unit_price_minor_units",
			"currency_code",
		}).AddRow("bitem_test", int64(1299), "GBP"))

	mock.ExpectExec(regexp.QuoteMeta(`
		UPDATE basket_items
		SET
			quantity = ?,
			line_total_minor_units = ?,
			updated_at = CURRENT_TIMESTAMP(6)
		WHERE basket_id = ?
		  AND basket_item_id = ?
	`)).
		WithArgs(4, int64(5196), "basket_test", "bitem_test").
		WillReturnResult(sqlmock.NewResult(0, 1))

	expectTouchBasket(mock, "basket_test")
	mock.ExpectCommit()

	expectBasketRow(mock, "basket_test", "ACTIVE", "GBP", now)
	expectBasketItems(mock, "basket_test", nil)

	_, err := repo.UpdateItemQuantity(context.Background(), UpdateItemQuantityCommand{
		BasketID:     "basket_test",
		BasketItemID: "bitem_test",
		Quantity:     4,
	})
	if err != nil {
		t.Fatalf("UpdateItemQuantity() error = %v, want nil", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func expectBasketRow(mock sqlmock.Sqlmock, basketID, status, currencyCode string, now time.Time) {
	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT
			basket_id,
			status,
			currency_code,
			created_at,
			updated_at
		FROM baskets
		WHERE basket_id = ?
	`)).
		WithArgs(basketID).
		WillReturnRows(sqlmock.NewRows([]string{
			"basket_id",
			"status",
			"currency_code",
			"created_at",
			"updated_at",
		}).AddRow(basketID, status, currencyCode, now, now))
}

func expectBasketItems(mock sqlmock.Sqlmock, basketID string, rows *sqlmock.Rows) {
	if rows == nil {
		rows = sqlmock.NewRows([]string{
			"basket_item_id",
			"basket_id",
			"product_id",
			"variant_id",
			"product_name_snapshot",
			"variant_name_snapshot",
			"quantity",
			"unit_price_minor_units",
			"line_total_minor_units",
			"currency_code",
			"created_at",
			"updated_at",
		})
	}

	mock.ExpectQuery(regexp.QuoteMeta(`
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
	`)).
		WithArgs(basketID).
		WillReturnRows(rows)
}

func expectLockBasketForUpdate(mock sqlmock.Sqlmock, basketID, status, currencyCode string) {
	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT
			status,
			currency_code
		FROM baskets
		WHERE basket_id = ?
		FOR UPDATE
	`)).
		WithArgs(basketID).
		WillReturnRows(sqlmock.NewRows([]string{"status", "currency_code"}).AddRow(status, currencyCode))
}

func expectFindBasketItemForUpdateNotFound(mock sqlmock.Sqlmock, basketID, productID, variantID string) {
	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT
			basket_item_id,
			quantity
		FROM basket_items
		WHERE basket_id = ?
		  AND product_id = ?
		  AND variant_id = ?
		FOR UPDATE
	`)).
		WithArgs(basketID, productID, variantID).
		WillReturnRows(sqlmock.NewRows([]string{"basket_item_id", "quantity"}))
}

func expectFindBasketItemForUpdateFound(
	mock sqlmock.Sqlmock,
	basketID string,
	productID string,
	variantID string,
	basketItemID string,
	quantity int,
) {
	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT
			basket_item_id,
			quantity
		FROM basket_items
		WHERE basket_id = ?
		  AND product_id = ?
		  AND variant_id = ?
		FOR UPDATE
	`)).
		WithArgs(basketID, productID, variantID).
		WillReturnRows(sqlmock.NewRows([]string{"basket_item_id", "quantity"}).AddRow(basketItemID, quantity))
}

func expectTouchBasket(mock sqlmock.Sqlmock, basketID string) {
	mock.ExpectExec(regexp.QuoteMeta(`
		UPDATE baskets
		SET updated_at = CURRENT_TIMESTAMP(6)
		WHERE basket_id = ?
	`)).
		WithArgs(basketID).
		WillReturnResult(sqlmock.NewResult(0, 1))
}
