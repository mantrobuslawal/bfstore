package basket

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
)

const (
	minBasketQuantity = 1
	maxBasketQuantity = 99
)

type CatalogClient interface {
	ValidateProductVariant(ctx context.Context, query ValidateProductVariantQuery) (CatalogProductVariant, error)
}

// Service contains Basket business logic.
type Service struct {
	repository    Repository
	catalogClient CatalogClient
	logger        *slog.Logger
}

// NewService creates a Basket Service
func NewService(repository Repository, catalogClient CatalogClient, logger *slog.Logger) *Service {
	if logger == nil {
		logger = slog.Default()
	}

	return &Service{
		repository:    repository,
		catalogClient: catalogClient,
		logger:        logger,
	}
}

// CreateBasket creates an empty basket and returns its basket id.
func (s *Service) CreateBasket(ctx context.Context, query BasketQuery) (Basket, error) {
	code := query.CurrencyCode
	defaultCurrencySet, currencyIsInValid := validateCurrency(code)

	if currencyIsInValid {
		s.logger.DebugContext(ctx, "invalid basket currency code", "invalid_currency_code", code)
		return Basket{}, ErrInvalidCurrenyCode
	}

	if defaultCurrencySet {
		s.logger.DebugContext(ctx, "basket currency defaulted",
			"default_currency_code", code)
	}

	id, err := NewBasketID()
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to generate new basket id", "error", err)
		return Basket{}, fmt.Errorf("generate new basket id: %w", err)
	}

	basket := Basket{
		BasketID:    BasketID(id),
		Status:      BasketStatusActive,
		Subtotal:    Money{CurrencyCode: CurrencyCode(code)},
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
	err := basketIdCheck(basketId)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidBasketID):
			s.logger.ErrorContext(ctx, "invalid basket_id", "error", ErrInvalidBasketID, "basket_id", basketId)
			return Basket{}, ErrInvalidBasketID
		default:
			s.logger.ErrorContext(ctx, "failed to validate basket_id", "error", err, "basket_id", basketId)
			return Basket{}, fmt.Errorf("validate basket id: %w", err)
		}
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
	// check basket id
	basketId := query.BasketID
	err := basketIdCheck(basketId)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidBasketID):
			s.logger.ErrorContext(ctx, "invalid basket_id", "error", ErrInvalidBasketID, "basket_id", basketId)
			return Basket{}, ErrInvalidBasketID
		default:
			s.logger.ErrorContext(ctx, "failed to validate basket_id", "error", err, "basket_id", basketId)
			return Basket{}, fmt.Errorf("validate basket id: %w", err)
		}
	}

	// check quantity
	quantity := query.Quantity
	err = validateQuantity(quantity)
	if err != nil {
		s.logger.ErrorContext(ctx, "invalid basket item quantity", "error", ErrInvalidQuantity, "quantity", quantity)
		return Basket{}, ErrInvalidQuantity
	}

	// get basket
	currentBasket, err := s.repository.GetBasket(ctx, BasketID(basketId))
	if err != nil {

		if errors.Is(err, ErrBasketNotFound) {
			s.logger.DebugContext(ctx, "add item rejected because basket was not found",
				"basket_id", basketId)
			return Basket{}, ErrBasketNotFound
		}

		s.logger.ErrorContext(ctx, "failed to load basket before adding item", "basket_id", basketId, "error", err)
		return Basket{}, fmt.Errorf("load basket before adding item: %w", err)
	}
	s.logger.InfoContext(ctx, "basket retrieved", "basket_id", currentBasket.BasketID, "currency_code", string(currentBasket.Subtotal.CurrencyCode), "status", string(basket.Status))

	// validate basket status
	if currentBasket.Status != BasketStatusActive {
		s.logger.WarnContext(ctx, "add item rejected because basket is not modifiable",
			"basket_id", currentBasket.BasketID,
			"status", currentBasket.Status,
			"prouduct_id", query.ProductID,
			"variant_id", query.VariantID)

		return Basket{}, ErrBasketNotModifiable
	}

	// verify product id & variant id
	catalogItem, err := s.catalogClient.ValidateProductVariant(ctx, ValidateProductVariantQuery{
		ProductID: query.ProductID,
		VariantID: query.VariantID,
	})

	if err != nil {
		switch {
		case errors.Is(err, ErrProductNotFound),
			errors.Is(err, ErrVariantNotFound),
			errors.Is(err, ErrProductVariantMismatch),
			errors.Is(err, ErrProductNotSellable):

			s.logger.DebugContext(ctx, "add item rejected by catalog validation",
				"error", err,
				"basket_id", basketId,
				"product_id", query.ProductID,
				"variant_id", query.VariantID,
			)
			return Basket{}, err

		default:
			s.logger.ErrorContext(ctx, "add item rejected because catalog item is not sellable",
				"basket_id", query.BasketID,
				"product_id", query.ProductID,
				"variant_id", query.VariantID,
			)

			return Basket{}, ErrProductNotSellable
		}

	}

	// check for exisistence of product in basket

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

func validateQuantity(quantity int) error {
	if (quantity < minBasketQuantity) || (quantity > maxBasketQuantity) {
		return ErrInvalidQuantity
	}
	return nil
}

func validateCurrency(code string) (bool, bool) {
	currencyCode := CurrencyCode(code)
	currencyIsValid, defaultCurrencySet := currencyCode.isValid()
	if !currencyIsValid {
		return false, true // default curreny code NOT used and invalid currency code
	}
	if defaultCurrencySet {
		return true, false // default currency code used and NO invalid currency code
	}
	return false, false // default currency code NOT used and NO invalid currency code
}

func basketIdCheck(basketId string) error {
	if strings.TrimSpace(basketId) == "" {
		return ErrInvalidBasketID
	}
	err := ValidateBasketID(basketId)
	if err != nil {
		return fmt.Errorf("validate basket id: %w", err)
	}
	return nil
}
