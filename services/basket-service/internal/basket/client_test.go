package basket

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	catalogv1 "github.com/mantrobuslawal/bfstore/gen/go/bfstore/catalog/v1"
	commonv1 "github.com/mantrobuslawal/bfstore/gen/go/bfstore/common/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type fakeCatalogServiceClient struct {
	resp *catalogv1.ValidateProductVariantResponse
	err  error
}

func (f *fakeCatalogServiceClient) ListProducts(ctx context.Context, in *catalogv1.ListProductsRequest, opts ...grpc.CallOption) (*catalogv1.ListProductsResponse, error) {
	return nil, nil
}

func (f *fakeCatalogServiceClient) GetProduct(ctx context.Context, in *catalogv1.GetProductRequest, opts ...grpc.CallOption) (*catalogv1.GetProductResponse, error) {
	return nil, nil
}

func (f *fakeCatalogServiceClient) ValidateProductVariant(ctx context.Context, in *catalogv1.ValidateProductVariantRequest, opts ...grpc.CallOption) (*catalogv1.ValidateProductVariantResponse, error) {
	return f.resp, f.err
}

func (f *fakeCatalogServiceClient) ListCategories(ctx context.Context, in *catalogv1.ListCategoriesRequest, opts ...grpc.CallOption) (*catalogv1.ListCategoriesResponse, error) {
	return nil, nil
}

func (f *fakeCatalogServiceClient) ListProductAttributeDefinitions(ctx context.Context, in *catalogv1.ListProductAttributeDefinitionsRequest, opts ...grpc.CallOption) (*catalogv1.ListProductAttributeDefinitionsResponse, error) {
	return nil, nil
}

func TestCatalogGRPCClientValidateProductVariant(t *testing.T) {
	t.Parallel()

	client := NewCatalogGRPCClient(
		&fakeCatalogServiceClient{
			resp: &catalogv1.ValidateProductVariantResponse{
				ProductId:   "prod_1",
				VariantId:   "var_1",
				ProductName: "Go Gopher Mug",
				VariantName: "Blue",
				UnitPrice:   &commonv1.Money{AmountMinor: 1299, CurrencyCode: "GBP"},
				Sellable:    true,
			},
		},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		time.Second,
	)

	got, err := client.ValidateProductVariant(context.Background(), ValidateProductVariantQuery{
		ProductID: "prod_1",
		VariantID: "var_1",
	})
	if err != nil {
		t.Fatalf("ValidateProductVariant() error = %v, want nil", err)
	}

	if got.ProductName != "Go Gopher Mug" {
		t.Fatalf("ProductName = %q, want %q", got.ProductName, "Go Gopher Mug")
	}

	if got.VariantName != "Blue" {
		t.Fatalf("VariantName = %q, want %q", got.VariantName, "Blue")
	}

	if got.UnitPrice.AmountMinor != 1299 {
		t.Fatalf("AmountMinor = %d, want 1299", got.UnitPrice.AmountMinor)
	}
}

func TestCatalogGRPCClientValidateProductVariantDetectsIDMismatch(t *testing.T) {
	t.Parallel()

	client := NewCatalogGRPCClient(
		&fakeCatalogServiceClient{
			resp: &catalogv1.ValidateProductVariantResponse{
				ProductId:   "prod_other",
				VariantId:   "var_1",
				ProductName: "Go Gopher Mug",
				VariantName: "Blue",
				UnitPrice:   &commonv1.Money{AmountMinor: 1299, CurrencyCode: "GBP"},
				Sellable:    true,
			},
		},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		time.Second,
	)

	_, err := client.ValidateProductVariant(context.Background(), ValidateProductVariantQuery{
		ProductID: "prod_1",
		VariantID: "var_1",
	})
	if !errors.Is(err, ErrProductVariantMismatch) {
		t.Fatalf("ValidateProductVariant() error = %v, want %v", err, ErrProductVariantMismatch)
	}
}

func TestMapCatalogValidationError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		err  error
		want error
	}{
		{
			name: "product not found",
			err:  status.Error(codes.NotFound, "product not found"),
			want: ErrProductNotFound,
		},
		{
			name: "variant not found",
			err:  status.Error(codes.NotFound, "variant not found"),
			want: ErrVariantNotFound,
		},
		{
			name: "failed precondition",
			err:  status.Error(codes.FailedPrecondition, "product not sellable"),
			want: ErrProductNotSellable,
		},
		{
			name: "unavailable wraps catalog unavailable",
			err:  status.Error(codes.Unavailable, "unavailable"),
			want: ErrCatalogServiceUnavailable,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := mapCatalogValidationError(tt.err)
			if !errors.Is(got, tt.want) {
				t.Fatalf("mapCatalogValidationError() = %v, want wrapping %v", got, tt.want)
			}
		})
	}
}
