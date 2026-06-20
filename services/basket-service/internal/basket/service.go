package basket

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
)

// Service contains Basket Service business logic.
type Service struct {
	repository    Repository
	catalogClient CatalogClient
	logger        *slog.Logger
}

// NewService creates a Basket Service.
func NewService(repository Repository, catalogClient CatalogClient, logger *slog.Logger) *Service {
	if repository == nil {
		panic("basket: nil repository")
	}

	if catalogClient == nil {
		panic("basket: nil catalog client")
	}

	if logger == nil {
		logger = slog.Default()
	}

	return &Service{
		repository:    repository,
		catalogClient: catalogClient,
		logger:        logger.With("component", "basket_service"),
	}
}

// CreateBasket creates an empty basket and returns it.
func (s *Service) CreateBasket(ctx context.Context, query BasketQuery) (Basket, error) {
	currencyCode, defaultCurrencySet, err := NormaliseCurrencyCode(query.CurrencyCode)
	if err != nil {
		s.logger.DebugContext(ctx, "invalid basket currency code",
			"currency_code", query.CurrencyCode,
		)

		return Basket{}, ErrInvalidCurrencyCode
	}

	if defaultCurrencySet {
		s.logger.DebugContext(ctx, "basket currency defaulted",
			"default_currency_code", currencyCode,
		)
	}

	id, err := NewBasketID()
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to generate new basket id",
			"error", err,
		)

		return Basket{}, fmt.Errorf("generate new basket id: %w", err)
	}

	basket := Basket{
		BasketID: BasketID(id),
		Status:   BasketStatusActive,
		Subtotal: Money{
			CurrencyCode: currencyCode,
		},
		BasketItems: nil,
	}

	created, err := s.repository.CreateBasket(ctx, basket)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to create basket",
			"error", err,
			"basket_id", basket.BasketID,
			"currency_code", currencyCode,
		)

		return Basket{}, fmt.Errorf("create basket: %w", err)
	}

	s.logger.InfoContext(ctx, "basket created",
		"basket_id", created.BasketID,
		"currency_code", created.Subtotal.CurrencyCode,
		"status", created.Status,
	)

	return created, nil
}

// GetBasket returns an existing basket.
func (s *Service) GetBasket(ctx context.Context, query BasketQuery) (Basket, error) {
	basketID := strings.TrimSpace(query.BasketID)
	if err := basketIDCheck(basketID); err != nil {
		s.logger.DebugContext(ctx, "invalid basket id",
			"error", err,
			"basket_id", query.BasketID,
		)

		return Basket{}, ErrInvalidBasketID
	}

	basket, err := s.repository.GetBasket(ctx, basketID)
	if err != nil {
		if errors.Is(err, ErrBasketNotFound) {
			s.logger.DebugContext(ctx, "basket not found",
				"basket_id", basketID,
			)

			return Basket{}, ErrBasketNotFound
		}

		s.logger.ErrorContext(ctx, "failed to get basket",
			"basket_id", basketID,
			"error", err,
		)

		return Basket{}, fmt.Errorf("get basket: %w", err)
	}

	s.logger.DebugContext(ctx, "basket retrieved",
		"basket_id", basket.BasketID,
		"currency_code", basket.Subtotal.CurrencyCode,
		"status", basket.Status,
		"item_count", len(basket.BasketItems),
	)

	return basket, nil
}

// AddItem adds a line item to an existing basket.
//
// If the same product/variant pair already exists, the repository increases the
// existing quantity.
func (s *Service) AddItem(ctx context.Context, query BasketQuery) (Basket, error) {
	basketID := strings.TrimSpace(query.BasketID)
	if err := basketIDCheck(basketID); err != nil {
		s.logger.DebugContext(ctx, "invalid basket id for add item",
			"error", err,
			"basket_id", query.BasketID,
		)

		return Basket{}, ErrInvalidBasketID
	}

	productID := strings.TrimSpace(query.ProductID)
	if productID == "" {
		s.logger.DebugContext(ctx, "missing product id for add item",
			"basket_id", basketID,
		)

		return Basket{}, ErrMissingProductID
	}

	variantID := strings.TrimSpace(query.VariantID)
	if variantID == "" {
		s.logger.DebugContext(ctx, "missing variant id for add item",
			"basket_id", basketID,
			"product_id", productID,
		)

		return Basket{}, ErrMissingVariantID
	}

	if err := validateQuantity(query.Quantity); err != nil {
		s.logger.DebugContext(ctx, "invalid basket item quantity",
			"error", err,
			"basket_id", basketID,
			"product_id", productID,
			"variant_id", variantID,
			"quantity", query.Quantity,
		)

		return Basket{}, ErrInvalidQuantity
	}

	currentBasket, err := s.repository.GetBasket(ctx, basketID)
	if err != nil {
		if errors.Is(err, ErrBasketNotFound) {
			s.logger.DebugContext(ctx, "add item rejected because basket was not found",
				"basket_id", basketID,
			)

			return Basket{}, ErrBasketNotFound
		}

		s.logger.ErrorContext(ctx, "failed to load basket before adding item",
			"basket_id", basketID,
			"error", err,
		)

		return Basket{}, fmt.Errorf("load basket before adding item: %w", err)
	}

	if currentBasket.Status != BasketStatusActive {
		s.logger.WarnContext(ctx, "add item rejected because basket is not modifiable",
			"basket_id", currentBasket.BasketID,
			"status", currentBasket.Status,
			"product_id", productID,
			"variant_id", variantID,
		)

		return Basket{}, ErrBasketNotModifiable
	}

	if existingItem, found := currentBasket.ExistingItem(productID, variantID); found {
		newQuantity := existingItem.Quantity + query.Quantity
		if err := validateQuantity(newQuantity); err != nil {
			s.logger.DebugContext(ctx, "add item rejected because combined quantity is invalid",
				"basket_id", basketID,
				"basket_item_id", existingItem.BasketItemID,
				"product_id", productID,
				"variant_id", variantID,
				"existing_quantity", existingItem.Quantity,
				"add_quantity", query.Quantity,
				"combined_quantity", newQuantity,
			)

			return Basket{}, ErrInvalidQuantity
		}
	}

	catalogItem, err := s.catalogClient.ValidateProductVariant(ctx, ValidateProductVariantQuery{
		ProductID: productID,
		VariantID: variantID,
	})
	if err != nil {
		if isExpectedCatalogValidationFailure(err) || errors.Is(err, ErrCatalogServiceUnavailable) {
			s.logger.DebugContext(ctx, "add item rejected by catalog validation",
				"error", err,
				"basket_id", basketID,
				"product_id", productID,
				"variant_id", variantID,
			)

			return Basket{}, err
		}

		s.logger.ErrorContext(ctx, "failed to validate catalog product variant",
			"error", err,
			"basket_id", basketID,
			"product_id", productID,
			"variant_id", variantID,
		)

		return Basket{}, fmt.Errorf("validate catalog product variant: %w", err)
	}

	if !catalogItem.Sellable {
		s.logger.WarnContext(ctx, "add item rejected because catalog item is not sellable",
			"basket_id", basketID,
			"product_id", productID,
			"variant_id", variantID,
		)

		return Basket{}, ErrProductNotSellable
	}

	updatedBasket, err := s.repository.AddItem(ctx, AddValidatedItemCommand{
		BasketID:            basketID,
		ProductID:           catalogItem.ProductID,
		VariantID:           catalogItem.VariantID,
		ProductNameSnapshot: catalogItem.ProductName,
		VariantNameSnapshot: catalogItem.VariantName,
		Quantity:            query.Quantity,
		UnitPrice:           catalogItem.UnitPrice,
	})
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to add item to basket",
			"error", err,
			"basket_id", basketID,
			"product_id", productID,
			"variant_id", variantID,
			"quantity", query.Quantity,
		)

		return Basket{}, fmt.Errorf("add item to basket: %w", err)
	}

	s.logger.InfoContext(ctx, "basket item added",
		"basket_id", basketID,
		"product_id", productID,
		"variant_id", variantID,
		"quantity", query.Quantity,
		"item_count", len(updatedBasket.BasketItems),
		"currency_code", catalogItem.UnitPrice.CurrencyCode,
	)

	return updatedBasket, nil
}

// UpdateItemQuantity replaces the quantity of an existing basket item.
func (s *Service) UpdateItemQuantity(ctx context.Context, query BasketQuery) (Basket, error) {
	basketID := strings.TrimSpace(query.BasketID)
	if err := basketIDCheck(basketID); err != nil {
		s.logger.DebugContext(ctx, "invalid basket id for update item quantity",
			"error", err,
			"basket_id", query.BasketID,
		)

		return Basket{}, ErrInvalidBasketID
	}

	basketItemID := strings.TrimSpace(query.BasketItemID)
	if err := basketItemIDCheck(basketItemID); err != nil {
		s.logger.DebugContext(ctx, "invalid basket item id for update item quantity",
			"error", err,
			"basket_id", basketID,
			"basket_item_id", query.BasketItemID,
		)

		return Basket{}, ErrInvalidItemID
	}

	if err := validateQuantity(query.Quantity); err != nil {
		s.logger.DebugContext(ctx, "invalid basket item quantity",
			"error", err,
			"basket_id", basketID,
			"basket_item_id", basketItemID,
			"quantity", query.Quantity,
		)

		return Basket{}, ErrInvalidQuantity
	}

	updatedBasket, err := s.repository.UpdateItemQuantity(ctx, UpdateItemQuantityCommand{
		BasketID:     basketID,
		BasketItemID: basketItemID,
		Quantity:     query.Quantity,
	})
	if err != nil {
		if errors.Is(err, ErrBasketNotFound) || errors.Is(err, ErrBasketItemNotFound) {
			s.logger.DebugContext(ctx, "update item quantity target not found",
				"error", err,
				"basket_id", basketID,
				"basket_item_id", basketItemID,
			)

			return Basket{}, err
		}

		s.logger.ErrorContext(ctx, "failed to update basket item quantity",
			"error", err,
			"basket_id", basketID,
			"basket_item_id", basketItemID,
			"quantity", query.Quantity,
		)

		return Basket{}, fmt.Errorf("update basket item quantity: %w", err)
	}

	s.logger.InfoContext(ctx, "basket item quantity updated",
		"basket_id", basketID,
		"basket_item_id", basketItemID,
		"quantity", query.Quantity,
		"item_count", len(updatedBasket.BasketItems),
		"currency_code", updatedBasket.Subtotal.CurrencyCode,
	)

	return updatedBasket, nil
}

// RemoveItem removes a basket item from an existing basket.
func (s *Service) RemoveItem(ctx context.Context, query BasketQuery) (Basket, error) {
	basketID := strings.TrimSpace(query.BasketID)
	if err := basketIDCheck(basketID); err != nil {
		s.logger.DebugContext(ctx, "invalid basket id for remove item",
			"error", err,
			"basket_id", query.BasketID,
		)

		return Basket{}, ErrInvalidBasketID
	}

	basketItemID := strings.TrimSpace(query.BasketItemID)
	if err := basketItemIDCheck(basketItemID); err != nil {
		s.logger.DebugContext(ctx, "invalid basket item id for remove item",
			"error", err,
			"basket_id", basketID,
			"basket_item_id", query.BasketItemID,
		)

		return Basket{}, ErrInvalidItemID
	}

	updatedBasket, err := s.repository.RemoveItem(ctx, RemoveItemCommand{
		BasketID:     basketID,
		BasketItemID: basketItemID,
	})
	if err != nil {
		if errors.Is(err, ErrBasketNotFound) || errors.Is(err, ErrBasketItemNotFound) {
			s.logger.DebugContext(ctx, "remove item target not found",
				"error", err,
				"basket_id", basketID,
				"basket_item_id", basketItemID,
			)

			return Basket{}, err
		}

		s.logger.ErrorContext(ctx, "failed to remove basket item",
			"error", err,
			"basket_id", basketID,
			"basket_item_id", basketItemID,
		)

		return Basket{}, fmt.Errorf("remove basket item: %w", err)
	}

	s.logger.InfoContext(ctx, "basket item removed",
		"basket_id", basketID,
		"basket_item_id", basketItemID,
		"item_count", len(updatedBasket.BasketItems),
		"currency_code", updatedBasket.Subtotal.CurrencyCode,
	)

	return updatedBasket, nil
}

// ClearBasket clears all basket items from an existing basket and returns the
// active empty basket.
func (s *Service) ClearBasket(ctx context.Context, query BasketQuery) (Basket, error) {
	basketID := strings.TrimSpace(query.BasketID)
	if err := basketIDCheck(basketID); err != nil {
		s.logger.DebugContext(ctx, "invalid basket id for clear basket",
			"error", err,
			"basket_id", query.BasketID,
		)

		return Basket{}, ErrInvalidBasketID
	}

	updatedBasket, err := s.repository.ClearBasket(ctx, ClearBasketCommand{BasketID: basketID})
	if err != nil {
		if errors.Is(err, ErrBasketNotFound) {
			s.logger.DebugContext(ctx, "clear basket target not found",
				"basket_id", basketID,
			)

			return Basket{}, ErrBasketNotFound
		}

		s.logger.ErrorContext(ctx, "failed to clear basket",
			"error", err,
			"basket_id", basketID,
		)

		return Basket{}, fmt.Errorf("clear basket: %w", err)
	}

	s.logger.InfoContext(ctx, "basket cleared",
		"basket_id", basketID,
		"basket_status", updatedBasket.Status,
		"currency_code", updatedBasket.Subtotal.CurrencyCode,
	)

	return updatedBasket, nil
}

func validateQuantity(quantity int) error {
	if quantity < minBasketQuantity || quantity > maxBasketQuantity {
		return ErrInvalidQuantity
	}

	return nil
}

func basketIDCheck(basketID string) error {
	if strings.TrimSpace(basketID) == "" {
		return ErrInvalidBasketID
	}

	if err := ValidateBasketID(basketID); err != nil {
		return fmt.Errorf("validate basket id: %w", err)
	}

	return nil
}

func basketItemIDCheck(basketItemID string) error {
	if strings.TrimSpace(basketItemID) == "" {
		return ErrInvalidItemID
	}

	if err := ValidateBasketItemID(basketItemID); err != nil {
		return fmt.Errorf("validate basket item id: %w", err)
	}

	return nil
}
