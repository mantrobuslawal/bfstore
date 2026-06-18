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
		errors.Is(err, basket.ErrInvalidCurrenyCode),
		errors.Is(err, basket.ErrMissingProductID),
		errors.Is(err, basket.ErrMissingVariantID):

		return status.Error(codes.InvalidArgument, err.Error())

	case errors.Is(err, basket.ErrBasketNotFound),
		errors.Is(err, basket.ErrProductNotFound),
		errors.Is(err, basket.ErrVariantNotFound):
		return status.Error(codes.NotFound, "not found")

	case errors.Is(err, basket.ErrProductNotSellable):
		return status.Error(codes.FailedPrecondition, "product or variant not sellable")

	case errors.Is(err, basket.ErrProductVariantMismatch):
		return status.Error(codes.FailedPrecondition, "product and variant mismatch")

	case errors.Is(err, basket.ErrBasketCurrencyMismatch):
		return status.Error(codes.FailedPrecondition, "basket currency does not match catalog item currency")

	case errors.Is(err, basket.ErrBasketNotModifiable):
		return status.Error(codes.FailedPrecondition, "basket not modifiable")

	case errors.Is(err, basket.ErrCatalogServiceUnavailable):
		return status.Error(codes.Unavailable, "catalog service unavailable")

	default:
		return status.Error(codes.Internal, "internal server error")
	}
}
