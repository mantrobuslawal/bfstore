package grpcadapter

import (
	"context"
	"log/slog"
	"strings"

	basketv1 "github.com/mantrobuslawal/bfstore/gen/go/bfstore/basket/v1"
	"github.com/mantrobuslawal/bfstore/services/basket-service/internal/basket"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// BasketHandler implements the generated Basket Service gRPC interface.
type BasketHandler struct {
	basketv1.UnimplementedBasketServiceServer

	basketService *basket.Service
	logger        *slog.Logger
}

// NewBasketHandler creates a basket service gRPC handler.
func NewBasketHandler(basketService *basket.Service, logger *slog.Logger) *BasketHandler {
	if basketService == nil {
		panic("grpcadapter: nil basket service")
	}

	if logger == nil {
		logger = slog.Default()
	}

	return &BasketHandler{
		basketService: basketService,
		logger:        logger.With("component", "basket_grpc_handler"),
	}
}

// CreateBasket creates a basket and returns it.
func (h *BasketHandler) CreateBasket(
	ctx context.Context,
	req *basketv1.CreateBasketRequest,
) (*basketv1.CreateBasketResponse, error) {
	if req == nil {
		h.logger.WarnContext(ctx, "create basket request is nil")
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}

	result, err := h.basketService.CreateBasket(ctx, basket.BasketQuery{
		CurrencyCode: req.GetCurrencyCode(),
	})
	if err != nil {
		h.logger.ErrorContext(ctx, "create basket failed",
			"error", err,
			"currency_code", req.GetCurrencyCode(),
		)

		return nil, mapServiceError(err)
	}

	protoBasket, err := mapBasketToProto(result)
	if err != nil {
		h.logger.ErrorContext(ctx, "map created basket to proto failed",
			"error", err,
			"basket_id", result.BasketID,
		)

		return nil, mapServiceError(err)
	}

	return &basketv1.CreateBasketResponse{Basket: protoBasket}, nil
}

// GetBasket retrieves a previously created basket.
//
// It requires the basket ID.
func (h *BasketHandler) GetBasket(
	ctx context.Context,
	req *basketv1.GetBasketRequest,
) (*basketv1.GetBasketResponse, error) {
	if req == nil {
		h.logger.WarnContext(ctx, "get basket request is nil")
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}

	basketID := strings.TrimSpace(req.GetBasketId())
	if basketID == "" {
		h.logger.WarnContext(ctx, "get basket request missing basket id")
		return nil, status.Error(codes.InvalidArgument, "basket id is required")
	}

	result, err := h.basketService.GetBasket(ctx, basket.BasketQuery{
		BasketID: basketID,
	})
	if err != nil {
		h.logger.ErrorContext(ctx, "get basket failed",
			"error", err,
			"basket_id", basketID,
		)

		return nil, mapServiceError(err)
	}

	protoBasket, err := mapBasketToProto(result)
	if err != nil {
		h.logger.ErrorContext(ctx, "map basket to proto failed",
			"error", err,
			"basket_id", basketID,
		)

		return nil, mapServiceError(err)
	}

	return &basketv1.GetBasketResponse{Basket: protoBasket}, nil
}

// AddItem adds a basket item to an existing basket.
func (h *BasketHandler) AddItem(
	ctx context.Context,
	req *basketv1.AddItemRequest,
) (*basketv1.AddItemResponse, error) {
	if req == nil {
		h.logger.WarnContext(ctx, "add item request is nil")
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}

	basketID := strings.TrimSpace(req.GetBasketId())
	if basketID == "" {
		h.logger.WarnContext(ctx, "add item request missing basket id")
		return nil, status.Error(codes.InvalidArgument, "basket id is required")
	}

	productID := strings.TrimSpace(req.GetProductId())
	if productID == "" {
		h.logger.WarnContext(ctx, "add item request missing product id",
			"basket_id", basketID,
		)

		return nil, status.Error(codes.InvalidArgument, "product id is required")
	}

	variantID := strings.TrimSpace(req.GetVariantId())
	if variantID == "" {
		h.logger.WarnContext(ctx, "add item request missing variant id",
			"basket_id", basketID,
			"product_id", productID,
		)

		return nil, status.Error(codes.InvalidArgument, "variant id is required")
	}

	result, err := h.basketService.AddItem(ctx, basket.BasketQuery{
		BasketID:  basketID,
		ProductID: productID,
		VariantID: variantID,
		Quantity:  int(req.GetQuantity()),
	})
	if err != nil {
		h.logger.ErrorContext(ctx, "add item failed",
			"error", err,
			"basket_id", basketID,
			"product_id", productID,
			"variant_id", variantID,
			"quantity", req.GetQuantity(),
		)

		return nil, mapServiceError(err)
	}

	protoBasket, err := mapBasketToProto(result)
	if err != nil {
		h.logger.ErrorContext(ctx, "map add item basket to proto failed",
			"error", err,
			"basket_id", basketID,
		)

		return nil, mapServiceError(err)
	}

	return &basketv1.AddItemResponse{Basket: protoBasket}, nil
}

// UpdateItemQuantity updates the quantity of a basket item.
func (h *BasketHandler) UpdateItemQuantity(
	ctx context.Context,
	req *basketv1.UpdateItemQuantityRequest,
) (*basketv1.UpdateItemQuantityResponse, error) {
	if req == nil {
		h.logger.WarnContext(ctx, "update item quantity request is nil")
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}

	basketID := strings.TrimSpace(req.GetBasketId())
	if basketID == "" {
		h.logger.WarnContext(ctx, "update item quantity request missing basket id")
		return nil, status.Error(codes.InvalidArgument, "basket id is required")
	}

	basketItemID := strings.TrimSpace(req.GetBasketItemId())
	if basketItemID == "" {
		h.logger.WarnContext(ctx, "update item quantity request missing basket item id",
			"basket_id", basketID,
		)

		return nil, status.Error(codes.InvalidArgument, "basket item id is required")
	}

	result, err := h.basketService.UpdateItemQuantity(ctx, basket.BasketQuery{
		BasketID:     basketID,
		BasketItemID: basketItemID,
		Quantity:     int(req.GetQuantity()),
	})
	if err != nil {
		h.logger.ErrorContext(ctx, "update item quantity failed",
			"error", err,
			"basket_id", basketID,
			"basket_item_id", basketItemID,
			"quantity", req.GetQuantity(),
		)

		return nil, mapServiceError(err)
	}

	protoBasket, err := mapBasketToProto(result)
	if err != nil {
		h.logger.ErrorContext(ctx, "map update item quantity basket to proto failed",
			"error", err,
			"basket_id", basketID,
		)

		return nil, mapServiceError(err)
	}

	return &basketv1.UpdateItemQuantityResponse{Basket: protoBasket}, nil
}

// RemoveItem removes a basket item from the basket.
func (h *BasketHandler) RemoveItem(
	ctx context.Context,
	req *basketv1.RemoveItemRequest,
) (*basketv1.RemoveItemResponse, error) {
	if req == nil {
		h.logger.WarnContext(ctx, "remove item request is nil")
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}

	basketID := strings.TrimSpace(req.GetBasketId())
	if basketID == "" {
		h.logger.WarnContext(ctx, "remove item request missing basket id")
		return nil, status.Error(codes.InvalidArgument, "basket id is required")
	}

	basketItemID := strings.TrimSpace(req.GetBasketItemId())
	if basketItemID == "" {
		h.logger.WarnContext(ctx, "remove item request missing basket item id",
			"basket_id", basketID,
		)

		return nil, status.Error(codes.InvalidArgument, "basket item id is required")
	}

	result, err := h.basketService.RemoveItem(ctx, basket.BasketQuery{
		BasketID:     basketID,
		BasketItemID: basketItemID,
	})
	if err != nil {
		h.logger.ErrorContext(ctx, "remove item failed",
			"error", err,
			"basket_id", basketID,
			"basket_item_id", basketItemID,
		)

		return nil, mapServiceError(err)
	}

	protoBasket, err := mapBasketToProto(result)
	if err != nil {
		h.logger.ErrorContext(ctx, "map remove item basket to proto failed",
			"error", err,
			"basket_id", basketID,
		)

		return nil, mapServiceError(err)
	}

	return &basketv1.RemoveItemResponse{Basket: protoBasket}, nil
}

// ClearBasket removes all basket items from the basket.
func (h *BasketHandler) ClearBasket(
	ctx context.Context,
	req *basketv1.ClearBasketRequest,
) (*basketv1.ClearBasketResponse, error) {
	if req == nil {
		h.logger.WarnContext(ctx, "clear basket request is nil")
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}

	basketID := strings.TrimSpace(req.GetBasketId())
	if basketID == "" {
		h.logger.WarnContext(ctx, "clear basket request missing basket id")
		return nil, status.Error(codes.InvalidArgument, "basket id is required")
	}

	result, err := h.basketService.ClearBasket(ctx, basket.BasketQuery{
		BasketID: basketID,
	})
	if err != nil {
		h.logger.ErrorContext(ctx, "clear basket failed",
			"error", err,
			"basket_id", basketID,
		)

		return nil, mapServiceError(err)
	}

	protoBasket, err := mapBasketToProto(result)
	if err != nil {
		h.logger.ErrorContext(ctx, "map clear basket response to proto failed",
			"error", err,
			"basket_id", basketID,
		)

		return nil, mapServiceError(err)
	}

	return &basketv1.ClearBasketResponse{Basket: protoBasket}, nil
}
