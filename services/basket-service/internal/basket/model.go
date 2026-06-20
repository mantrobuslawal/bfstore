package basket

import (
	"strings"
	"time"
)

const (
	minBasketQuantity       = 1
	maxBasketQuantity       = 99
	maxIDGenerationAttempts = 3
)

// CurrencyCode represents an ISO 4217 currency code.
type CurrencyCode string

const defaultCurrencyCode CurrencyCode = "GBP"

// NormaliseCurrencyCode validates and normalises a currency code.
//
// The first bfstore basket slice supports GBP only.
// Empty currency defaults to GBP.
func NormaliseCurrencyCode(code string) (CurrencyCode, bool, error) {
	normalised := strings.ToUpper(strings.TrimSpace(code))
	if normalised == "" {
		return defaultCurrencyCode, true, nil
	}

	if normalised != string(defaultCurrencyCode) {
		return "", false, ErrInvalidCurrencyCode
	}

	return CurrencyCode(normalised), false, nil
}

// Money represents a monetary value in minor units.
//
// This mirrors the Protobuf Money contract.
type Money struct {
	AmountMinor  int64
	CurrencyCode CurrencyCode
}

// BasketID represents a basket identifier.
type BasketID string

// BasketItemID represents a basket item identifier.
type BasketItemID string

// ProductID represents a catalog product identifier.
type ProductID string

// VariantID represents the identifier of a catalog product variant.
type VariantID string

// BasketItem represents a unique basket line item.
type BasketItem struct {
	BasketItemID BasketItemID
	BasketID     BasketID
	ProductID    ProductID
	VariantID    VariantID

	ProductNameSnapshot string
	VariantNameSnapshot string

	Quantity  int
	UnitPrice Money
	LineTotal Money

	AddedAt   time.Time
	UpdatedAt time.Time
}

// Basket represents a store shopping basket.
type Basket struct {
	BasketID    BasketID
	BasketItems []*BasketItem
	Subtotal    Money
	Status      BasketStatus
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// ExistingItem checks whether an item already exists in the basket.
//
// If the item exists, it returns the BasketItem and true.
// Otherwise, it returns an empty BasketItem and false.
func (b Basket) ExistingItem(productID, variantID string) (BasketItem, bool) {
	for _, item := range b.BasketItems {
		if item == nil {
			continue
		}

		if item.ProductID == ProductID(productID) && item.VariantID == VariantID(variantID) {
			return *item, true
		}
	}

	return BasketItem{}, false
}

// BasketQuery represents input passed from gRPC handlers into the service layer.
type BasketQuery struct {
	CurrencyCode string
	BasketID     string
	BasketItemID string
	ProductID    string
	VariantID    string
	Quantity     int
}

// AddValidatedItemCommand is used once catalog-service has validated the
// product/variant pair and returned a snapshot.
type AddValidatedItemCommand struct {
	BasketID            string
	ProductID           string
	VariantID           string
	ProductNameSnapshot string
	VariantNameSnapshot string
	Quantity            int
	UnitPrice           Money
}

// UpdateItemQuantityCommand requests a basket item quantity replacement.
type UpdateItemQuantityCommand struct {
	BasketID     string
	BasketItemID string
	Quantity     int
}

// RemoveItemCommand requests removal of a basket item.
type RemoveItemCommand struct {
	BasketID     string
	BasketItemID string
}

// ClearBasketCommand requests removal of all basket items.
type ClearBasketCommand struct {
	BasketID string
}

// CatalogProductVariant is the basket-service view of a catalog-validated
// product variant.
type CatalogProductVariant struct {
	ProductID   string
	VariantID   string
	ProductName string
	VariantName string
	UnitPrice   Money
	Sellable    bool
}

// ValidateProductVariantQuery identifies a product/variant pair to validate
// against catalog-service.
type ValidateProductVariantQuery struct {
	ProductID string
	VariantID string
}

type existingBasketItem struct {
	ID       string
	Quantity int
}

type lockedBasketItem struct {
	ID                  string
	UnitPriceMinorUnits int64
	CurrencyCode        CurrencyCode
}
