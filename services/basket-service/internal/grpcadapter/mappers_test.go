package grpcadapter

import (
	"testing"
	"time"

	basketv1 "github.com/mantrobuslawal/bfstore/gen/go/bfstore/basket/v1"
	"github.com/mantrobuslawal/bfstore/services/basket-service/internal/basket"
)

func TestMapBasketToProtoIncludesEmptyBasket(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC()
	got, err := mapBasketToProto(basket.Basket{
		BasketID:    basket.BasketID("basket_01ARZ3NDEKTSV4RRFFQ69G5FAV_QG6M8K7Z"),
		BasketItems: nil,
		Subtotal:    basket.Money{AmountMinor: 0, CurrencyCode: "GBP"},
		Status:      basket.BasketStatusActive,
		CreatedAt:   now,
		UpdatedAt:   now,
	})
	if err != nil {
		t.Fatalf("mapBasketToProto() error = %v, want nil", err)
	}

	if got.GetBasketId() == "" {
		t.Fatal("BasketId is empty")
	}

	if got.GetStatus() != basketv1.BasketStatus_BASKET_STATUS_ACTIVE {
		t.Fatalf("Status = %v, want ACTIVE", got.GetStatus())
	}

	if len(got.GetItems()) != 0 {
		t.Fatalf("len(Items) = %d, want 0", len(got.GetItems()))
	}
}

func TestMapBasketStatusToProto(t *testing.T) {
	t.Parallel()

	tests := []struct {
		status basket.BasketStatus
		want   basketv1.BasketStatus
	}{
		{basket.BasketStatusActive, basketv1.BasketStatus_BASKET_STATUS_ACTIVE},
		{basket.BasketStatusCleared, basketv1.BasketStatus_BASKET_STATUS_CLEARED},
		{basket.BasketStatusExpired, basketv1.BasketStatus_BASKET_STATUS_EXPIRED},
		{basket.BasketStatusCheckedOut, basketv1.BasketStatus_BASKET_STATUS_CHECKED_OUT},
	}

	for _, tt := range tests {
		got, err := mapBasketStatusToProto(tt.status)
		if err != nil {
			t.Fatalf("mapBasketStatusToProto(%q) error = %v, want nil", tt.status, err)
		}

		if got != tt.want {
			t.Fatalf("mapBasketStatusToProto(%q) = %v, want %v", tt.status, got, tt.want)
		}
	}
}
