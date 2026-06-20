package catalog

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestMySQLRepositoryValidateProductVariantReturnsSnapshot(t *testing.T) {
	t.Parallel()

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New() error = %v", err)
	}
	defer db.Close()

	repo := NewMySQLRepository(db, slog.New(slog.NewTextHandler(io.Discard, nil)))

	rows := sqlmock.NewRows([]string{
		"product_id",
		"name",
		"status",
		"variant_id",
		"variant_name",
		"status",
		"price_minor",
		"currency_code",
	}).AddRow(
		"prod_go_gopher_mug",
		"Go Gopher Mug",
		"active",
		"var_blue",
		"Blue",
		"active",
		int64(1299),
		"GBP",
	)

	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT
		  p.product_id,
		  p.name,
		  p.status,
		  v.variant_id,
		  v.variant_name,
		  v.status,
		  v.price_minor,
		  v.currency_code
		FROM products p
		INNER JOIN product_variants v
		  ON v.product_id = p.product_id
		WHERE p.product_id = ?
		  AND v.variant_id = ?
		LIMIT 1
	`)).
		WithArgs(ProductID("prod_go_gopher_mug"), VariantID("var_blue")).
		WillReturnRows(rows)

	got, err := repo.ValidateProductVariant(context.Background(), ValidateProductVariantQuery{
		ProductID: ProductID("prod_go_gopher_mug"),
		VariantID: VariantID("var_blue"),
	})
	if err != nil {
		t.Fatalf("ValidateProductVariant() error = %v, want nil", err)
	}

	if got.ProductID != ProductID("prod_go_gopher_mug") {
		t.Fatalf("ProductID = %q, want %q", got.ProductID, "prod_go_gopher_mug")
	}

	if got.VariantID != VariantID("var_blue") {
		t.Fatalf("VariantID = %q, want %q", got.VariantID, "var_blue")
	}

	if got.ProductName != "Go Gopher Mug" {
		t.Fatalf("ProductName = %q, want %q", got.ProductName, "Go Gopher Mug")
	}

	if got.VariantName != "Blue" {
		t.Fatalf("VariantName = %q, want %q", got.VariantName, "Blue")
	}

	if got.ProductStatus != ProductStatus("active") {
		t.Fatalf("ProductStatus = %q, want %q", got.ProductStatus, "active")
	}

	if got.VariantStatus != ProductVariantStatus("active") {
		t.Fatalf("VariantStatus = %q, want %q", got.VariantStatus, "active")
	}

	if got.UnitPrice.AmountMinor != 1299 {
		t.Fatalf("UnitPrice.AmountMinor = %d, want %d", got.UnitPrice.AmountMinor, 1299)
	}

	if got.UnitPrice.CurrencyCode != "GBP" {
		t.Fatalf("UnitPrice.CurrencyCode = %q, want %q", got.UnitPrice.CurrencyCode, "GBP")
	}

	if !got.Sellable {
		t.Fatal("Sellable = false, want true")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}

func TestMySQLRepositoryValidateProductVariantClassifiesMissingProduct(t *testing.T) {
	t.Parallel()

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New() error = %v", err)
	}
	defer db.Close()

	repo := NewMySQLRepository(db, slog.New(slog.NewTextHandler(io.Discard, nil)))

	mock.ExpectQuery("FROM products p").
		WithArgs(ProductID("prod_missing"), VariantID("var_blue")).
		WillReturnRows(sqlmock.NewRows([]string{
			"product_id",
			"name",
			"status",
			"variant_id",
			"variant_name",
			"status",
			"price_minor",
			"currency_code",
		}))

	mock.ExpectQuery("SELECT EXISTS").
		WithArgs(ProductID("prod_missing")).
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))

	_, err = repo.ValidateProductVariant(context.Background(), ValidateProductVariantQuery{
		ProductID: ProductID("prod_missing"),
		VariantID: VariantID("var_blue"),
	})
	if !errors.Is(err, ErrProductNotFound) {
		t.Fatalf("ValidateProductVariant() error = %v, want %v", err, ErrProductNotFound)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}

func TestMySQLRepositoryValidateProductVariantClassifiesMissingVariant(t *testing.T) {
	t.Parallel()

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New() error = %v", err)
	}
	defer db.Close()

	repo := NewMySQLRepository(db, slog.New(slog.NewTextHandler(io.Discard, nil)))

	mock.ExpectQuery("FROM products p").
		WithArgs(ProductID("prod_go_gopher_mug"), VariantID("var_missing")).
		WillReturnRows(sqlmock.NewRows([]string{
			"product_id",
			"name",
			"status",
			"variant_id",
			"variant_name",
			"status",
			"price_minor",
			"currency_code",
		}))

	mock.ExpectQuery("SELECT EXISTS").
		WithArgs(ProductID("prod_go_gopher_mug")).
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))

	mock.ExpectQuery("SELECT EXISTS").
		WithArgs(VariantID("var_missing")).
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))

	_, err = repo.ValidateProductVariant(context.Background(), ValidateProductVariantQuery{
		ProductID: ProductID("prod_go_gopher_mug"),
		VariantID: VariantID("var_missing"),
	})
	if !errors.Is(err, ErrProductVariantNotFound) {
		t.Fatalf("ValidateProductVariant() error = %v, want %v", err, ErrProductVariantNotFound)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}

func TestMySQLRepositoryValidateProductVariantClassifiesMismatch(t *testing.T) {
	t.Parallel()

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New() error = %v", err)
	}
	defer db.Close()

	repo := NewMySQLRepository(db, slog.New(slog.NewTextHandler(io.Discard, nil)))

	mock.ExpectQuery("FROM products p").
		WithArgs(ProductID("prod_go_gopher_mug"), VariantID("var_other_product")).
		WillReturnRows(sqlmock.NewRows([]string{
			"product_id",
			"name",
			"status",
			"variant_id",
			"variant_name",
			"status",
			"price_minor",
			"currency_code",
		}))

	mock.ExpectQuery("SELECT EXISTS").
		WithArgs(ProductID("prod_go_gopher_mug")).
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))

	mock.ExpectQuery("SELECT EXISTS").
		WithArgs(VariantID("var_other_product")).
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))

	_, err = repo.ValidateProductVariant(context.Background(), ValidateProductVariantQuery{
		ProductID: ProductID("prod_go_gopher_mug"),
		VariantID: VariantID("var_other_product"),
	})
	if !errors.Is(err, ErrProductVariantProductMismatch) {
		t.Fatalf("ValidateProductVariant() error = %v, want %v", err, ErrProductVariantProductMismatch)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}
