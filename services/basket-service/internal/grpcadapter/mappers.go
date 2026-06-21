package grpcadapter

import (
	"fmt"
	"time"

	basketv1 "github.com/mantrobuslawal/bfstore/gen/go/bfstore/basket/v1"
	commonv1 "github.com/mantrobuslawal/bfstore/gen/go/bfstore/common/v1"
	"github.com/mantrobuslawal/bfstore/services/basket-service/internal/basket"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func mapBasketToProto(domainBasket basket.Basket) (*basketv1.Basket, error) {
	protoItems := make([]*basketv1.BasketItem, 0, len(domainBasket.BasketItems))

	for _, item := range domainBasket.BasketItems {
		if item == nil {
			continue
		}

		protoBasketItem := mapBasketItemToProto(*item)
		protoItems = append(protoItems, protoBasketItem)
	}

	status, err := mapBasketStatusToProto(domainBasket.Status)
	if err != nil {
		return nil, fmt.Errorf("map basket status to proto: %w", err)
	}

	return &basketv1.Basket{
		BasketId:  string(domainBasket.BasketID),
		Subtotal:  mapMoneyToProto(domainBasket.Subtotal),
		Items:     protoItems,
		Status:    status,
		CreatedAt: mapTimeToProto(domainBasket.CreatedAt),
		UpdatedAt: mapTimeToProto(domainBasket.UpdatedAt),
	}, nil
}

func mapBasketItemToProto(basketItem basket.BasketItem) *basketv1.BasketItem {
	return &basketv1.BasketItem{
		BasketItemId:        string(basketItem.BasketItemID),
		ProductId:           string(basketItem.ProductID),
		VariantId:           string(basketItem.VariantID),
		ProductNameSnapshot: basketItem.ProductNameSnapshot,
		VariantNameSnapshot: basketItem.VariantNameSnapshot,
		Quantity:            int32(basketItem.Quantity),
		UnitPrice:           mapMoneyToProto(basketItem.UnitPrice),
		LineTotal:           mapMoneyToProto(basketItem.LineTotal),
		AddedAt:             mapTimeToProto(basketItem.AddedAt),
		UpdatedAt:           mapTimeToProto(basketItem.UpdatedAt),
	}
}

func mapMoneyToProto(money basket.Money) *commonv1.Money {
	return &commonv1.Money{
		AmountMinor:  money.AmountMinor,
		CurrencyCode: string(money.CurrencyCode),
	}
}

func mapTimeToProto(value time.Time) *timestamppb.Timestamp {
	if value.IsZero() {
		return nil
	}

	return timestamppb.New(value)
}

func mapBasketStatusToProto(status basket.BasketStatus) (basketv1.BasketStatus, error) {
	switch status {
	case basket.BasketStatusActive:
		return basketv1.BasketStatus_BASKET_STATUS_ACTIVE, nil
	case basket.BasketStatusCheckedOut:
		return basketv1.BasketStatus_BASKET_STATUS_CHECKED_OUT, nil
	case basket.BasketStatusCleared:
		return basketv1.BasketStatus_BASKET_STATUS_CLEARED, nil
	case basket.BasketStatusExpired:
		return basketv1.BasketStatus_BASKET_STATUS_EXPIRED, nil
	default:
		return basketv1.BasketStatus_BASKET_STATUS_UNSPECIFIED,
			fmt.Errorf("unknown basket status: %q", string(status))
	}
}
