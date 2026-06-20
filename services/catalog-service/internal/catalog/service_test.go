package catalog

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
)

type fakeCatalogRepository struct {
	validateProductVariantResult ValidatedProductVariant
	validateProductVariantErr    error
	validateProductVariantQuery  ValidateProductVariantQuery
	validateProductVariantCalled bool
}

func (f *fakeCatalogRepository) ListProducts(ctx context.Context, query ListQuery) ([]Product, error) {
	return nil, nil
}

func (f *fakeCatalogRepository) GetProduct(ctx context.Context, productID ProductID) (Product, error) {
	return Product{}, nil
}

func (f *fakeCatalogRepository) ValidateProductVariant(
	ctx context.Context,
	query ValidateProductVariantQuery,
) (ValidatedProductVariant, error) {
	f.validateProductVariantCalled = true
	f.validateProductVariantQuery = query

	if f.validateProductVariantErr != nil {
		return ValidatedProductVariant{}, f.validateProductVariantErr
	}

	return f.validateProductVariantResult, nil
}

func (f *fakeCatalogRepository) ListCategories(ctx context.Context, query ListQuery) ([]Category, error) {
	return nil, nil
}

func (f *fakeCatalogRepository) ListProductAttributeDefinitions(
	ctx context.Context,
	query ListQuery,
) ([]ProductAttributeDefinition, error) {
	return nil, nil
}

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestServiceValidateProductVariantReturnsSellableSnapshot(t *testing.T) {
	t.Parallel()

	repo := &fakeCatalogRepository{
		validateProductVariantResult: ValidatedProductVariant{
			ProductID:     ProductID("prod_go_gopher_mug"),
			VariantID:     VariantID("var_blue"),
			ProductName:   "Go Gopher Mug",
			VariantName:   "Blue",
			ProductStatus: ProductStatus("active"),
			VariantStatus: ProductVariantStatus("active"),
			UnitPrice: Money{
				AmountMinor:  1299,
				CurrencyCode: "GBP",
			},
		},
	}

	service := NewService(repo, discardLogger())

	got, err := service.ValidateProductVariant(context.Background(), ValidateProductVariantQuery{
		ProductID: ProductID(" prod_go_gopher_mug "),
		VariantID: VariantID(" var_blue "),
	})
	if err != nil {
		t.Fatalf("ValidateProductVariant() error = %v, want nil", err)
	}

	if !repo.validateProductVariantCalled {
		t.Fatal("repository ValidateProductVariant was not called")
	}

	if repo.validateProductVariantQuery.ProductID != ProductID("prod_go_gopher_mug") {
		t.Fatalf("repo query ProductID = %q, want %q", repo.validateProductVariantQuery.ProductID, "prod_go_gopher_mug")
	}

	if repo.validateProductVariantQuery.VariantID != VariantID("var_blue") {
		t.Fatalf("repo query VariantID = %q, want %q", repo.validateProductVariantQuery.VariantID, "var_blue")
	}

	if !got.Sellable {
		t.Fatal("Sellable = false, want true")
	}

	if got.UnitPrice.AmountMinor != 1299 {
		t.Fatalf("UnitPrice.AmountMinor = %d, want %d", got.UnitPrice.AmountMinor, 1299)
	}

	if got.UnitPrice.CurrencyCode != "GBP" {
		t.Fatalf("UnitPrice.CurrencyCode = %q, want %q", got.UnitPrice.CurrencyCode, "GBP")
	}
}

func TestServiceValidateProductVariantRejectsInvalidInput(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		query   ValidateProductVariantQuery
		wantErr error
	}{
		{
			name: "missing product id",
			query: ValidateProductVariantQuery{
				VariantID: VariantID("var_blue"),
			},
			wantErr: ErrInvalidProductID,
		},
		{
			name: "missing variant id",
			query: ValidateProductVariantQuery{
				ProductID: ProductID("prod_go_gopher_mug"),
			},
			wantErr: ErrInvalidVariantID,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			repo := &fakeCatalogRepository{}
			service := NewService(repo, discardLogger())

			got, err := service.ValidateProductVariant(context.Background(), tt.query)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("ValidateProductVariant() error = %v, want %v", err, tt.wantErr)
			}

			if got != (ValidatedProductVariant{}) {
				t.Fatalf("ValidateProductVariant() result = %+v, want zero value", got)
			}

			if repo.validateProductVariantCalled {
				t.Fatal("repository was called, want validation to fail before repository call")
			}
		})
	}
}

func TestServiceValidateProductVariantRejectsUnsellableStatuses(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		result  ValidatedProductVariant
		wantErr error
	}{
		{
			name: "inactive product",
			result: ValidatedProductVariant{
				ProductID:     ProductID("prod_go_gopher_mug"),
				VariantID:     VariantID("var_blue"),
				ProductName:   "Go Gopher Mug",
				VariantName:   "Blue",
				ProductStatus: ProductStatus("inactive"),
				VariantStatus: ProductVariantStatus("active"),
				UnitPrice:     Money{AmountMinor: 1299, CurrencyCode: "GBP"},
			},
			wantErr: ErrProductNotSellable,
		},
		{
			name: "inactive variant",
			result: ValidatedProductVariant{
				ProductID:     ProductID("prod_go_gopher_mug"),
				VariantID:     VariantID("var_blue"),
				ProductName:   "Go Gopher Mug",
				VariantName:   "Blue",
				ProductStatus: ProductStatus("active"),
				VariantStatus: ProductVariantStatus("inactive"),
				UnitPrice:     Money{AmountMinor: 1299, CurrencyCode: "GBP"},
			},
			wantErr: ErrProductVariantNotSellable,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			repo := &fakeCatalogRepository{
				validateProductVariantResult: tt.result,
			}
			service := NewService(repo, discardLogger())

			got, err := service.ValidateProductVariant(context.Background(), ValidateProductVariantQuery{
				ProductID: ProductID("prod_go_gopher_mug"),
				VariantID: VariantID("var_blue"),
			})
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("ValidateProductVariant() error = %v, want %v", err, tt.wantErr)
			}

			if got != (ValidatedProductVariant{}) {
				t.Fatalf("ValidateProductVariant() result = %+v, want zero value", got)
			}
		})
	}
}

func TestServiceValidateProductVariantWrapsRepositoryError(t *testing.T) {
	t.Parallel()

	repo := &fakeCatalogRepository{
		validateProductVariantErr: ErrProductVariantNotFound,
	}
	service := NewService(repo, discardLogger())

	_, err := service.ValidateProductVariant(context.Background(), ValidateProductVariantQuery{
		ProductID: ProductID("prod_go_gopher_mug"),
		VariantID: VariantID("var_blue"),
	})
	if !errors.Is(err, ErrProductVariantNotFound) {
		t.Fatalf("ValidateProductVariant() error = %v, want wrapping %v", err, ErrProductVariantNotFound)
	}
}
