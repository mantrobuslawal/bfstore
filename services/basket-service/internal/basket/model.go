package basket

import (
	"strings"
	"time"
)

// CurrencyCode represents an ISO 4217 Currency Code.
type CurrencyCode string

const defaultCurrencyCode = "gbp"

func (c CurrencyCode) IsValid() bool {
	normalisedCurrency := strings.TrimSpace(strings.ToLower(string(c)))

	if normalisedCurrency == "" {
		c = "GBP" // Default Currency is GBP
		return true
	}
	if normalisedCurrency != defaultCurrencyCode { // Store only handles GBP currently.
		return false
	}
	return false

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

func (id BasketID) IsValid() bool {
	// implement
	return true
}

// BasketItemID represents a basket identifier.
type BasketItemID string

func (id BasketItemID) IsValid() bool {
	// implement
	return true
}

// ProductID represents a catalog product identifier.
type ProductID string

func (id ProductID) IsValid() bool {
	// implement
	return true
}

// VariantID represents the identifier of a catalog product variant.
type VariantID string

func (id VariantID) IsValid() bool {
	// implement
	return true
}

// BasketItem represents an unique basket line item e.g. a unique product type
// in the basket.
type BasketItem struct {
	BasketItemID        BasketItemID
	ProductID           ProductID
	VariantID           VariantID
	ProductNameSnapShot string
	VariantNameSnapShot string
	Quantity            int
	UnitPrice           Money
	LineTotal           Money
	AddedAt             time.Time
	UpdatedAt           time.Time
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

// BasketQuery represents filter options that maybe passed from
// the grpc request to the service and then forwarded to the
// repository layer to filter results.
// Depending on the request some fields maybe empty.
//
// Accepts strings from gRPC layer and validates at Service layer.
type BasketQuery struct {
	CurrencyCode string
	BasketID     string
	BasketItemID string
	ProductID    string
	VariantID    string
	Quantity     int
}
