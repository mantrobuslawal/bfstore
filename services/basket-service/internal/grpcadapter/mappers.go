package grpcadapter

import (
	"fmt"
	"time"

	basketv1 "github.com/mantrobuslawal/bfstore/gen/go/bfstore/basket/v1"
	v1 "github.com/mantrobuslawal/bfstore/gen/go/bfstore/common/v1"
	"github.com/mantrobuslawal/bfstore/services/basket-service/internal/basket"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func mapBasketToProto(basket basket.Basket) (*basketv1.Basket, error) {
	var protoItems []*basketv1.BasketItem

	// There are cases when a basket with zero basket items will be mapped,
	// such as when a call to ClearBasket is made. We want to avoid creating
	// a proto basket items slice, but still want to map other basket fields.
	if len(basket.BasketItems) > 0 {
		protoItems = make([]*basketv1.BasketItem, 0, len(basket.BasketItems))
		for _, item := range basket.BasketItems {
			protoBasketItem := mapBasketItemToProto(*item)
			protoItems = append(protoItems, protoBasketItem)
		}
	}

	status, err := mapBasketStatusToProto(basket.Status)
	if err != nil {
		return &basketv1.Basket{}, fmt.Errorf("map basket status to proto: %w", err)
	}

	return &basketv1.Basket{
		BasketId:  string(basket.BasketID),
		Subtotal:  mapMoneyToProto(basket.Subtotal),
		Items:     protoItems,
		Status:    status,
		CreatedAt: mapTimeToProto(basket.CreatedAt),
		UpdatedAt: mapTimeToProto(basket.UpdatedAt),
	}, nil
}

func mapBasketItemToProto(basketItem basket.BasketItem) *basketv1.BasketItem {
	return &basketv1.BasketItem{
		BasketItemId:        string(basketItem.BasketItemID),
		ProductId:           string(basketItem.ProductID),
		VariantId:           string(basketItem.VariantID),
		ProductNameSnapshot: basketItem.ProductNameSnapShot,
		VariantNameSnapshot: basketItem.VariantNameSnapShot,
		Quantity:            int32(basketItem.Quantity),
		UnitPrice:           mapMoneyToProto(basketItem.UnitPrice),
		LineTotal:           mapMoneyToProto(basketItem.LineTotal),
		AddedAt:             mapTimeToProto(basketItem.AddedAt),
		UpdatedAt:           mapTimeToProto(basketItem.UpdatedAt),
	}
}

func mapMoneyToProto(money basket.Money) *v1.Money {
	return &v1.Money{
		AmountMinor:  money.AmountMinor,
		CurrencyCode: string(money.CurrencyCode),
	}
}

func mapTimeToProto(time time.Time) *timestamppb.Timestamp {
	return timestamppb.New(time)
}

func mapBasketStatusToProto(status basket.BasketStatus) (basketv1.BasketStatus, error) {
	switch string(status) {
	case "active":
		return basketv1.BasketStatus_BASKET_STATUS_ACTIVE, nil
	case "checked_out":
		return basketv1.BasketStatus_BASKET_STATUS_CHECKED_OUT, nil
	case "cleared":
		return basketv1.BasketStatus_BASKET_STATUS_CLEARED, nil
	case "expired":
		return basketv1.BasketStatus_BASKET_STATUS_EXPIRED, nil
	default:
		return basketv1.BasketStatus_BASKET_STATUS_UNSPECIFIED, fmt.Errorf("unknown basket status: %q", string(status))
	}
}
