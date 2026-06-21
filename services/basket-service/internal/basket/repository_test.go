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
		WithArgs("basket_test").
		WillReturnRows(sqlmock.NewRows([]string{
			"basket_id", "status", "currency_code", "created_at", "updated_at",
		}).AddRow("basket_test", "ACTIVE", "GBP", now, now))

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
			"prod_1",
			"var_1",
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

	if got.Subtotal.AmountMinor != 2598 {
		t.Fatalf("Subtotal.AmountMinor = %d, want 2598", got.Subtotal.AmountMinor)
	}

	if len(got.BasketItems) != 1 {
		t.Fatalf("len(BasketItems) = %d, want 1", len(got.BasketItems))
	}

	item := got.BasketItems[0]
	if item.UnitPrice.AmountMinor != 1299 {
		t.Fatalf("UnitPrice.AmountMinor = %d, want 1299", item.UnitPrice.AmountMinor)
	}

	if item.LineTotal.AmountMinor != 2598 {
		t.Fatalf("LineTotal.AmountMinor = %d, want 2598", item.LineTotal.AmountMinor)
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

	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT
			status,
			currency_code
		FROM baskets
		WHERE basket_id = ?
		FOR UPDATE
	`)).
		WithArgs("basket_test").
		WillReturnRows(sqlmock.NewRows([]string{"status", "currency_code"}).AddRow("CHECKED_OUT", "GBP"))

	_, err = repo.lockBasketForUpdate(context.Background(), tx, "basket_test")
	if !errors.Is(err, ErrBasketNotModifiable) {
		t.Fatalf("lockBasketForUpdate() error = %v, want %v", err, ErrBasketNotModifiable)
	}
}
