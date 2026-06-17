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
		s.logger.ErrorContext(ctx, "failed to create new basket",
			"error", err,
			"basket_id", basket.BasketID,
			"currency_code", code)
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
	s.logger.InfoContext(ctx, "basket retrieved",
		"basket_id", currentBasket.BasketID,
		"currency_code", string(currentBasket.Subtotal.CurrencyCode),
		"status", currentBasket.Status)

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

	if !catalogItem.Sellable {
		s.logger.WarnContext(ctx, "add item rejected because catalog item is not sellable",
			"basket_id", query.BasketID,
			"product_id", query.ProductID,
			"variant_id", query.VariantID,
		)

		return Basket{}, ErrProductNotSellable
	}

	updatedBasket, err := s.repository.AddItem(ctx, BasketQuery{
		BasketID:  query.BasketID,
		ProductID: query.ProductID,
		VariantID: query.VariantID,
		Quantity:  query.Quantity,
	})
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to add item to basket",
			"error", err,
			"basket_id", query.BasketID,
			"product_id", query.ProductID,
			"varianr_id", query.VariantID,
			"quantity", query.Quantity,
		)

		return Basket{}, fmt.Errorf("add item to basket: %w", err)
	}

	s.logger.InfoContext(ctx, "basket item added",
		"basket_id", query.BasketID,
		"product_id", query.ProductID,
		"varianr_id", query.VariantID,
		"quantity", query.Quantity,
		"item_count", len(updatedBasket.BasketItems),
		"currency_code", catalogItem.UnitPrice.CurrencyCode,
	)

	return updatedBasket, nil
}

// UpdateItemQuantity updates the Quantity field of a bask item within a
// basket and returns the updated basket.
func (s *Service) UpdateItemQuantity(ctx context.Context, query BasketQuery) (Basket, error) {
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

	// check basket item id
	basketItemId := query.BasketItemID
	err = basketItemIdCheck(basketItemId)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidItemID):
			s.logger.ErrorContext(ctx, "invalid basket_item_id",
				"error", ErrInvalidItemID,
				"basket_id", basketId,
				"basket_item_id", basketItemId)
			return Basket{}, ErrInvalidBasketID
		default:
			s.logger.ErrorContext(ctx, "failed to validate basket_item_id", "error", err,
				"basket_id", basketId,
				"basket_item_id", basketItemId)
			return Basket{}, fmt.Errorf("validate basket id: %w", err)
		}
	}

	// check quantity
	quantity := query.Quantity
	err = validateQuantity(quantity)
	if err != nil {
		s.logger.ErrorContext(ctx, "invalid basket item quantity",
			"error", ErrInvalidQuantity,
			"quantity", quantity)
		return Basket{}, ErrInvalidQuantity
	}

	updatedBasket, err := s.UpdateItemQuantity(ctx, BasketQuery{
		BasketID:     basketId,
		BasketItemID: basketItemId,
		Quantity:     quantity,
	})

	if err != nil {
		s.logger.ErrorContext(ctx, "failed to update basket item quantity",
			"error", err,
			"basket_id", query.BasketID,
			"basket_item_id", query.BasketItemID,
			"quantity", query.Quantity,
		)

		return Basket{}, fmt.Errorf("update basket item quantity: %w", err)
	}

	s.logger.InfoContext(ctx, "basket item added",
		"basket_id", query.BasketID,
		"basket_item_id", query.BasketItemID,
		"quantity", query.Quantity,
		"item_count", len(updatedBasket.BasketItems),
		"currency_code", updatedBasket.Subtotal.CurrencyCode,
	)

	return updatedBasket, nil
}

// RemoveItem removes a basket item from an existing basket and returns
// the basket.
func (s *Service) RemoveItem(ctx context.Context, query BasketQuery) (Basket, error) {
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

	// check basket item id
	basketItemId := query.BasketItemID
	err = basketItemIdCheck(basketItemId)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidItemID):
			s.logger.ErrorContext(ctx, "invalid basket_item_id",
				"error", ErrInvalidItemID,
				"basket_id", basketId,
				"basket_item_id", basketItemId)
			return Basket{}, ErrInvalidBasketID
		default:
			s.logger.ErrorContext(ctx, "failed to validate basket_item_id", "error", err,
				"basket_id", basketId,
				"basket_item_id", basketItemId)
			return Basket{}, fmt.Errorf("validate basket id: %w", err)
		}
	}

	updatedBasket, err := s.repository.RemoveItem(ctx, BasketQuery{
		BasketID:     basketId,
		BasketItemID: basketItemId,
	})

	if err != nil {
		s.logger.ErrorContext(ctx, "failed to remove basket item",
			"error", err,
			"basket_id", query.BasketID,
			"basket_item_id", query.BasketItemID,
		)

		return Basket{}, fmt.Errorf("remove basket item: %w", err)
	}

	s.logger.InfoContext(ctx, "basket item removed",
		"basket_id", query.BasketID,
		"basket_item_id", query.BasketItemID,
		"item_count", len(updatedBasket.BasketItems),
		"currency_code", updatedBasket.Subtotal.CurrencyCode,
	)

	return updatedBasket, nil
}

// ClearBasket clears all basket items from an existing basket
// and returns the basket.
func (s *Service) ClearBasket(ctx context.Context, query BasketQuery) (Basket, error) {
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

	updatedBasket, err := s.repository.ClearBasket(ctx, BasketQuery{BasketID: basketId})
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to clear basket",
			"error", err,
			"basket_id", query.BasketID,
		)

		return Basket{}, fmt.Errorf("clear basket: %w", err)
	}

	s.logger.InfoContext(ctx, "basket cleared",
		"basket_id", query.BasketID,
		"basket_status", updatedBasket.Status,
		"currency_code", updatedBasket.Subtotal.CurrencyCode,
	)

	return updatedBasket, nil
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

func basketItemIdCheck(basketItemId string) error {
	if strings.TrimSpace(basketItemId) == "" {
		return ErrInvalidBasketID
	}
	err := ValidateBasketItemID(basketItemId)
	if err != nil {
		return fmt.Errorf("validate basket item id: %w", err)
	}
	return nil
}
