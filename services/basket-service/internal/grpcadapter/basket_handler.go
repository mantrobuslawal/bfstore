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
	if logger == nil {
		logger = slog.Default()
	}

	if basketService == nil {
		panic("grpcadapter: nil basket service")
	}

	return &BasketHandler{
		basketService: basketService,
		logger:        logger,
	}
}

// CreateBasket
func (h *BasketHandler) CreateBasket(ctx context.Context, req *basketv1.CreateBasketRequest) (*basketv1.CreateBasketResponse, error) {
	if req == nil {
		return &basketv1.CreateBasketResponse{}, status.Error(codes.InvalidArgument, "request is required")
	}

	result, err := h.basketService.CreateBasket(ctx, basket.BasketQuery{
		CurrencyCode: req.CurrencyCode,
	})
	if err != nil {
		return &basketv1.CreateBasketResponse{}, mapServiceError(err)
	}

	basket, err := mapBasketToProto(result)
	if err != nil {
		return &basketv1.CreateBasketResponse{}, mapServiceError(err)
	}

	return &basketv1.CreateBasketResponse{
		Basket: basket,
	}, nil
}

// GetBasket
func (h *BasketHandler) GetBasket(ctx context.Context, req *basketv1.GetBasketRequest) (*basketv1.GetBasketResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}

	id := req.GetBasketId()
	if strings.TrimSpace(id) == "" {
		return &basketv1.GetBasketResponse{}, status.Error(codes.InvalidArgument, "basket id is required")
	}

	result, err := h.basketService.GetBasket(ctx, basket.BasketQuery{
		BasketID: id,
	})
	if err != nil {
		return &basketv1.GetBasketResponse{}, mapServiceError(err)
	}

	basket, err := mapBasketToProto(result)
	if err != nil {
		return &basketv1.GetBasketResponse{}, mapServiceError(err)
	}

	return &basketv1.GetBasketResponse{
		Basket: basket,
	}, nil
}

// AddItem
func (h *BasketHandler) AddItem(ctx context.Context, req *basketv1.AddItemRequest) (*basketv1.AddItemResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}

	basketId := req.GetBasketId()
	if strings.TrimSpace(basketId) == "" {
		return &basketv1.AddItemResponse{}, status.Error(codes.InvalidArgument, "basket id is required")
	}

	productId := req.GetProductId()
	if strings.TrimSpace(productId) == "" {
		return &basketv1.AddItemResponse{}, status.Error(codes.InvalidArgument, "product id is required")
	}

	variantId := req.GetVariantId()
	if strings.TrimSpace(variantId) == "" {
		return &basketv1.AddItemResponse{}, status.Error(codes.InvalidArgument, "variant id is required")
	}

	result, err := h.basketService.GetBasket(ctx, basket.BasketQuery{
		BasketID:  basketId,
		ProductID: productId,
		VariantID: variantId,
	})
	if err != nil {
		return &basketv1.AddItemResponse{}, mapServiceError(err)
	}

	basket, err := mapBasketToProto(result)
	if err != nil {
		return &basketv1.AddItemResponse{}, mapServiceError(err)
	}

	return &basketv1.AddItemResponse{
		Basket: basket,
	}, nil
}

// UpdateItemQuantity
func (h *BasketHandler) UpdateItemQuantity(ctx context.Context, req *basketv1.UpdateItemQuantityRequest) (*basketv1.UpdateItemQuantityResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}

	basketId := req.GetBasketId()
	if strings.TrimSpace(basketId) == "" {
		return &basketv1.UpdateItemQuantityResponse{}, status.Error(codes.InvalidArgument, "basket id is required")
	}

	basketItemId := req.GetBasketItemId()
	if strings.TrimSpace(basketItemId) == "" {
		return &basketv1.UpdateItemQuantityResponse{}, status.Error(codes.InvalidArgument, "basket item id is required")
	}

	result, err := h.basketService.UpdateItemQuantity(ctx, basket.BasketQuery{
		BasketID:     basketId,
		BasketItemID: basketItemId,
		Quantity:     int(req.GetQuantity()),
	})
	if err != nil {
		return &basketv1.UpdateItemQuantityResponse{}, mapServiceError(err)
	}

	basket, err := mapBasketToProto(result)
	if err != nil {
		return &basketv1.UpdateItemQuantityResponse{}, mapServiceError(err)
	}

	return &basketv1.UpdateItemQuantityResponse{
		Basket: basket,
	}, nil
}

// RemoveItem
func (h *BasketHandler) RemoveItem(ctx context.Context, req *basketv1.RemoveItemRequest) (*basketv1.RemoveItemResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}

	basketId := req.GetBasketId()
	if strings.TrimSpace(basketId) == "" {
		return &basketv1.RemoveItemResponse{}, status.Error(codes.InvalidArgument, "basket id is required")
	}

	basketItemID := req.GetBasketItemId()
	if strings.TrimSpace(basketItemID) == "" {
		return &basketv1.RemoveItemResponse{}, status.Error(codes.InvalidArgument, "basket item id is required")
	}

	result, err := h.basketService.RemoveItem(ctx, basket.BasketQuery{
		BasketID:     basketId,
		BasketItemID: basketItemID,
	})
	if err != nil {
		return &basketv1.RemoveItemResponse{}, mapServiceError(err)
	}

	basket, err := mapBasketToProto(result)
	if err != nil {
		return &basketv1.RemoveItemResponse{}, mapServiceError(err)
	}

	return &basketv1.RemoveItemResponse{
		Basket: basket,
	}, nil
}

// ClearBasket
func (h *BasketHandler) ClearBasket(ctx context.Context, req *basketv1.ClearBasketRequest) (*basketv1.ClearBasketResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}

	basketId := req.GetBasketId()
	if strings.TrimSpace(basketId) == "" {
		return &basketv1.ClearBasketResponse{}, status.Error(codes.InvalidArgument, "basket id is required")
	}

	result, err := h.basketService.RemoveItem(ctx, basket.BasketQuery{
		BasketID: basketId,
	})
	if err != nil {
		return &basketv1.ClearBasketResponse{}, mapServiceError(err)
	}

	basket, err := mapBasketToProto(result)
	if err != nil {
		return &basketv1.ClearBasketResponse{}, mapServiceError(err)
	}

	return &basketv1.ClearBasketResponse{
		Basket: basket,
	}, nil
}
