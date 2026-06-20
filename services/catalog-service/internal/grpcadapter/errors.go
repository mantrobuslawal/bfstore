package grpcadapter

import (
	"errors"

	"github.com/mantrobuslawal/bfstore/services/catalog-service/internal/catalog"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func mapServiceError(err error) error {
	switch {
	case errors.Is(err, catalog.ErrInvalidProductID),
		errors.Is(err, catalog.ErrInvalidVariantID),
		errors.Is(err, catalog.ErrInvalidCategoryID),
		errors.Is(err, catalog.ErrInvalidPageSize),
		errors.Is(err, catalog.ErrInvalidPageToken),
		errors.Is(err, catalog.ErrInvalidDisplayOrder):
		return status.Error(codes.InvalidArgument, err.Error())

	case errors.Is(err, catalog.ErrProductNotFound),
		errors.Is(err, catalog.ErrProductVariantNotFound):
		return status.Error(codes.NotFound, err.Error())

	case errors.Is(err, catalog.ErrProductVariantProductMismatch),
		errors.Is(err, catalog.ErrProductNotSellable),
		errors.Is(err, catalog.ErrProductVariantNotSellable):
		return status.Error(codes.FailedPrecondition, err.Error())

	default:
		return status.Error(codes.Internal, "internal server error")
	}
}
