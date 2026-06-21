package grpcadapter

import (
	"errors"
	"testing"

	"github.com/mantrobuslawal/bfstore/services/basket-service/internal/basket"
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
		{name: "invalid basket id", err: basket.ErrInvalidBasketID, code: codes.InvalidArgument},
		{name: "invalid quantity", err: basket.ErrInvalidQuantity, code: codes.InvalidArgument},
		{name: "basket not found", err: basket.ErrBasketNotFound, code: codes.NotFound},
		{name: "basket item not found", err: basket.ErrBasketItemNotFound, code: codes.NotFound},
		{name: "product not found", err: basket.ErrProductNotFound, code: codes.NotFound},
		{name: "variant not found", err: basket.ErrVariantNotFound, code: codes.NotFound},
		{name: "product not sellable", err: basket.ErrProductNotSellable, code: codes.FailedPrecondition},
		{name: "catalog unavailable", err: basket.ErrCatalogServiceUnavailable, code: codes.Unavailable},
		{name: "unexpected", err: errors.New("boom"), code: codes.Internal},
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
