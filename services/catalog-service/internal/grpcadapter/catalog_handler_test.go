package grpcadapter

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"

	"github.com/mantrobuslawal/bfstore/services/catalog-service/internal/catalog"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	catalogv1 "github.com/mantrobuslawal/bfstore/gen/go/bfstore/catalog/v1"
)

type fakeProductVariantValidator struct {
	gotQuery catalog.ValidateProductVariantQuery
	result   catalog.ValidatedProductVariant
	err      error
	called   bool
}

func (f *fakeProductVariantValidator) ValidateProductVariant(
	ctx context.Context,
	query catalog.ValidateProductVariantQuery,
) (catalog.ValidatedProductVariant, error) {
	f.called = true
	f.gotQuery = query

	if f.err != nil {
		return catalog.ValidatedProductVariant{}, f.err
	}

	return f.result, nil
}

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestCatalogHandler_ValidateProductVariantReturnsSnapshot(t *testing.T) {
	t.Parallel()

	validator := &fakeProductVariantValidator{
		result: catalog.ValidatedProductVariant{
			ProductID:   catalog.ProductID("prod_go_gopher_mug"),
			VariantID:   catalog.VariantID("var_blue"),
			ProductName: "Go Gopher Mug",
			VariantName: "Blue",
			UnitPrice: catalog.Money{
				AmountMinor:  1299,
				CurrencyCode: "GBP",
			},
			Sellable: true,
		},
	}

	handler := &CatalogHandler{
		productVariantValidator: validator,
		logger:                  testLogger(),
	}

	resp, err := handler.ValidateProductVariant(
		context.Background(),
		&catalogv1.ValidateProductVariantRequest{
			ProductId: " prod_go_gopher_mug ",
			VariantId: " var_blue ",
		},
	)
	if err != nil {
		t.Fatalf("ValidateProductVariant() error = %v, want nil", err)
	}

	if !validator.called {
		t.Fatal("ValidateProductVariant() did not call product variant validator")
	}

	if validator.gotQuery.ProductID != catalog.ProductID("prod_go_gopher_mug") {
		t.Fatalf("ProductID query = %q, want %q", validator.gotQuery.ProductID, "prod_go_gopher_mug")
	}

	if validator.gotQuery.VariantID != catalog.VariantID("var_blue") {
		t.Fatalf("VariantID query = %q, want %q", validator.gotQuery.VariantID, "var_blue")
	}

	if resp.GetProductId() != "prod_go_gopher_mug" {
		t.Fatalf("ProductId = %q, want %q", resp.GetProductId(), "prod_go_gopher_mug")
	}

	if resp.GetVariantId() != "var_blue" {
		t.Fatalf("VariantId = %q, want %q", resp.GetVariantId(), "var_blue")
	}

	if resp.GetProductName() != "Go Gopher Mug" {
		t.Fatalf("ProductName = %q, want %q", resp.GetProductName(), "Go Gopher Mug")
	}

	if resp.GetVariantName() != "Blue" {
		t.Fatalf("VariantName = %q, want %q", resp.GetVariantName(), "Blue")
	}

	if resp.GetUnitPrice().GetAmountMinor() != 1299 {
		t.Fatalf("UnitPrice.AmountMinor = %d, want %d", resp.GetUnitPrice().GetAmountMinor(), 1299)
	}

	if resp.GetUnitPrice().GetCurrencyCode() != "GBP" {
		t.Fatalf("UnitPrice.CurrencyCode = %q, want %q", resp.GetUnitPrice().GetCurrencyCode(), "GBP")
	}

	if !resp.GetSellable() {
		t.Fatal("Sellable = false, want true")
	}
}

func TestCatalogHandler_ValidateProductVariantRejectsNilRequest(t *testing.T) {
	t.Parallel()

	handler := &CatalogHandler{
		productVariantValidator: &fakeProductVariantValidator{},
		logger:                  testLogger(),
	}

	resp, err := handler.ValidateProductVariant(context.Background(), nil)
	if resp != nil {
		t.Fatalf("response = %+v, want nil", resp)
	}

	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("status.Code(error) = %v, want %v", status.Code(err), codes.InvalidArgument)
	}
}

func TestCatalogHandler_ValidateProductVariantRejectsMissingProductID(t *testing.T) {
	t.Parallel()

	validator := &fakeProductVariantValidator{}
	handler := &CatalogHandler{
		productVariantValidator: validator,
		logger:                  testLogger(),
	}

	resp, err := handler.ValidateProductVariant(
		context.Background(),
		&catalogv1.ValidateProductVariantRequest{
			ProductId: "   ",
			VariantId: "var_blue",
		},
	)
	if resp != nil {
		t.Fatalf("response = %+v, want nil", resp)
	}

	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("status.Code(error) = %v, want %v", status.Code(err), codes.InvalidArgument)
	}

	if validator.called {
		t.Fatal("validator was called, want request validation to fail before service call")
	}
}

func TestCatalogHandler_ValidateProductVariantRejectsMissingVariantID(t *testing.T) {
	t.Parallel()

	validator := &fakeProductVariantValidator{}
	handler := &CatalogHandler{
		productVariantValidator: validator,
		logger:                  testLogger(),
	}

	resp, err := handler.ValidateProductVariant(
		context.Background(),
		&catalogv1.ValidateProductVariantRequest{
			ProductId: "prod_go_gopher_mug",
			VariantId: "   ",
		},
	)
	if resp != nil {
		t.Fatalf("response = %+v, want nil", resp)
	}

	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("status.Code(error) = %v, want %v", status.Code(err), codes.InvalidArgument)
	}

	if validator.called {
		t.Fatal("validator was called, want request validation to fail before service call")
	}
}

func TestCatalogHandler_ValidateProductVariantMapsServiceError(t *testing.T) {
	t.Parallel()

	validator := &fakeProductVariantValidator{
		err: errors.New("catalog goblin escaped"),
	}

	handler := &CatalogHandler{
		productVariantValidator: validator,
		logger:                  testLogger(),
	}

	resp, err := handler.ValidateProductVariant(
		context.Background(),
		&catalogv1.ValidateProductVariantRequest{
			ProductId: "prod_go_gopher_mug",
			VariantId: "var_blue",
		},
	)
	if resp != nil {
		t.Fatalf("response = %+v, want nil", resp)
	}

	if status.Code(err) != codes.Internal {
		t.Fatalf("status.Code(error) = %v, want %v", status.Code(err), codes.Internal)
	}
}
