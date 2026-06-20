package basket

import "errors"

// Domain errors.
var (
	ErrMissingBasketID = errors.New("missing basket id")
	ErrInvalidQuantity = errors.New("invalid item quantity")
	ErrUnknownItemID   = errors.New("unknown item id")
	ErrInvalidItemID   = errors.New("invalid item id")

	ErrInvalidBasketID     = errors.New("invalid basket id")
	ErrInvalidSubtotal     = errors.New("invalid subtotal")
	ErrInvalidBasketStatus = errors.New("invalid basket status")
	ErrInvalidCurrencyCode = errors.New("invalid currency code")

	ErrMissingProductID = errors.New("missing product id")
	ErrMissingVariantID = errors.New("missing variant id")

	ErrBasketNotFound     = errors.New("basket not found")
	ErrProductNotFound    = errors.New("product not found")
	ErrVariantNotFound    = errors.New("variant not found")
	ErrBasketItemNotFound = errors.New("basket item not found")

	ErrProductVariantMismatch = errors.New("product and variant mismatch")
	ErrBasketNotModifiable    = errors.New("basket not modifiable")

	ErrCatalogServiceUnavailable    = errors.New("catalog service unavailable")
	ErrUnexpectedPersistenceFailure = errors.New("unexpected persistence failure")

	ErrProductNotSellable     = errors.New("product or variant is not sellable")
	ErrBasketCurrencyMismatch = errors.New("basket currency does not match catalog item currency")
	ErrCouldNotAllocateItemID = errors.New("could not allocate unique basket item id")
)
