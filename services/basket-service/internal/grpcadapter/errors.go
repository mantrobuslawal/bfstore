package grpcadapter

import (
	"errors"

	"github.com/mantrobuslawal/bfstore/services/basket-service/internal/basket"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func mapServiceError(err error) error {
	switch {
	case errors.Is(err, basket.ErrMissingBasketID),
		errors.Is(err, basket.ErrInvalidBasketID),
		errors.Is(err, basket.ErrInvalidBasketStatus),
		errors.Is(err, basket.ErrInvalidItemID),
		errors.Is(err, basket.ErrInvalidQuantity),
		errors.Is(err, basket.ErrInvalidSubTotal),
		errors.Is(err, basket.ErrUnknownItemID),
		errors.Is(err, basket.ErrInvalidCurrenyCode):

		return status.Error(codes.InvalidArgument, err.Error())

	default:
		return status.Error(codes.Internal, "internal server error")
	}
}
