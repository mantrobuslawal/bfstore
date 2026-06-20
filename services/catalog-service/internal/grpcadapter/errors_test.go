package grpcadapter

import (
	"errors"
	"testing"

	"github.com/mantrobuslawal/bfstore/services/catalog-service/internal/catalog"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestMapServiceError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		err  error
		code codes.Code
	}{
		{
			name: "invalid product id maps to invalid argument",
			err:  catalog.ErrInvalidProductID,
			code: codes.InvalidArgument,
		},
		{
			name: "invalid variant id maps to invalid argument",
			err:  catalog.ErrInvalidVariantID,
			code: codes.InvalidArgument,
		},
		{
			name: "product not found maps to not found",
			err:  catalog.ErrProductNotFound,
			code: codes.NotFound,
		},
		{
			name: "product variant not found maps to not found",
			err:  catalog.ErrProductVariantNotFound,
			code: codes.NotFound,
		},
		{
			name: "variant product mismatch maps to failed precondition",
			err:  catalog.ErrProductVariantProductMismatch,
			code: codes.FailedPrecondition,
		},
		{
			name: "product not sellable maps to failed precondition",
			err:  catalog.ErrProductNotSellable,
			code: codes.FailedPrecondition,
		},
		{
			name: "variant not sellable maps to failed precondition",
			err:  catalog.ErrProductVariantNotSellable,
			code: codes.FailedPrecondition,
		},
		{
			name: "wrapped error maps correctly",
			err:  errors.Join(errors.New("context"), catalog.ErrProductVariantNotFound),
			code: codes.NotFound,
		},
		{
			name: "unexpected error maps to internal",
			err:  errors.New("database goblin escaped"),
			code: codes.Internal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := mapServiceError(tt.err)
			if status.Code(got) != tt.code {
				t.Fatalf("status.Code(mapServiceError()) = %v, want %v", status.Code(got), tt.code)
			}
		})
	}
}
