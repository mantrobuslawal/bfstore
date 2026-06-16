package basket

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
)

const (
	minBasketQuantity = 1
	maxBasketQuantity = 99
)

// Service contains Basket business logic.
type Service struct {
	repository Repository
	logger     *slog.Logger
}

// NewService creates a Basket Service
func NewService(repo Repository, logger *slog.Logger) *Service {
	if logger == nil {
		logger = slog.Default()
	}

	return &Service{
		repository: repo,
		logger:     logger,
	}
}

type catalogClient interface {
	ValidateProductVariant(ctx context.Context, query ValidateProductVariantQuery) (CatalogProductVariant, error)
}

type ValidateProductVariantQuery struct {
	ProductID string
	VariantID string
}

type CatalogProductVariant struct {
	ProductID   string
	VariantID   string
	ProductName string
	VariantName string
	UnitPrice   Money
	Sellable    bool
}

// CreateBasket creates an empty basket and returns its basket id.
func (s *Service) CreateBasket(ctx context.Context, query BasketQuery) (Basket, error) {
	currencyCode := CurrencyCode(query.CurrencyCode)
	currencyIsValid, defaultCurrencySet := currencyCode.isValid()

	if !currencyIsValid {
		s.logger.DebugContext(ctx, "invalid basket currency code", "invalid_currency_code", string(currencyCode))
		return Basket{}, ErrInvalidCurrenyCode
	}
	if defaultCurrencySet {
		s.logger.DebugContext(ctx, "basket currency defaulted",
			"default_currency_code", string(currencyCode))
	}

	id, err := NewBasketID()
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to generate new basket id", "error", err)
		return Basket{}, fmt.Errorf("generate new basket id: %w", err)
	}

	basket := Basket{
		BasketID:    BasketID(id),
		Status:      BasketStatusActive,
		Subtotal:    Money{CurrencyCode: currencyCode},
		BasketItems: nil,
	}

	created, err := s.repository.CreateBasket(ctx, basket)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to create new basket", "error", err, "basket_id", basket.BasketID, "currency_code", currencyCode)
		return Basket{}, fmt.Errorf("persist newly created basket: %w", err)
	}

	s.logger.InfoContext(ctx, "basket created", "basket_id", created.BasketID, "currency_code", string(created.Subtotal.CurrencyCode), "status", string(created.Status))

	return created, nil
}

// GetBasket takes a basket id and returns the assosicated basket.
func (s *Service) GetBasket(ctx context.Context, query BasketQuery) (Basket, error) {
	basketId := query.BasketID
	if strings.TrimSpace(basketId) == "" {
		s.logger.ErrorContext(ctx, "invalid basket_id", "error", ErrInvalidBasketID, "basket_id", basketId)
		return Basket{}, ErrInvalidBasketID
	}

	err := ValidateBasketID(basketId)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to validate basket_id", "error", err, "basket_id", basketId)
		return Basket{}, fmt.Errorf("validate basket id: %w", err)
	}

	basket, err := s.repository.GetBasket(ctx, BasketID(basketId))
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to get basket", "basket_id", basketId, "error", err)
		return Basket{}, fmt.Errorf("get basket: %w", err)
	}

	s.logger.InfoContext(ctx, "basket retrieved", "basket_id", basket.BasketID, "currency_code", string(basket.Subtotal.CurrencyCode), "status", string(basket.Status))
	return basket, nil
}

// AddItem adds a line item to an existing basket.
func (s *Service) AddItem(ctx context.Context, query BasketQuery) (Basket, error) {
	basketId := query.BasketID
	if strings.TrimSpace(basketId) == "" {
		s.logger.ErrorContext(ctx, "invalid basket_id", "error", ErrInvalidBasketID, "basket_id", basketId)
		return Basket{}, ErrInvalidBasketID
	}

	err := ValidateBasketID(basketId)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to validate basket_id", "error", err, "basket_id", basketId)
		return Basket{}, fmt.Errorf("validate basket id: %w", err)
	}

	basket, err := s.repository.GetBasket(ctx, BasketID(basketId))
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to get basket", "basket_id", basketId, "error", err)
		return Basket{}, fmt.Errorf("get basket: %w", err)
	}

	// search basket items for matching product and variant id, if found call Update Item quanitity
	// else create new item and add basket

	return Basket{}, nil
}

// UpdateItemQuantity updates the Quantity field of a bask item within a
// basket and rerturns the updated basket.
func (s *Service) UpdateItemQuantity(ctx context.Context, query BasketQuery) (Basket, error) {
	// TODO: Implement
	return Basket{}, nil
}

// RemoveItem removes a basket item from an existing basket and returns
// the basket.
func (s *Service) RemoveItem(ctx context.Context, query BasketQuery) (Basket, error) {
	// TODO: Implement
	return Basket{}, nil
}

// ClearBasket clears all basket items from an existing basket
// and returns the basket.
func (s *Service) ClearBasket(ctx context.Context, query BasketQuery) (Basket, error) {
	// TODO: Implement
	return Basket{}, nil
}
