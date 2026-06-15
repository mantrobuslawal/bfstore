package grpcadapter

import (
	basketv1 "github.com/mantrobuslawal/bfstore/gen/go/bfstore/basket/v1"
	"github.com/mantrobuslawal/bfstore/services/basket-service/internal/basket"
)

// SHOULD BE DEBUG LOGGING HERE

func mapBasketToProto(basket basket.Basket) (*basketv1.Basket, error) {
	return nil, nil
}
