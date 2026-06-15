package basket

import "errors"

// Domain errors.
var (
	// Common Service layer errors.
	ErrMissingBasketID     = errors.New("missing basket id")
	ErrInvalidQuantity     = errors.New("invalid item quantity")
	ErrUnknownItemID       = errors.New("unknown item id")
	ErrInvalidItemID       = errors.New("invalid item id")
	ErrInvalidBasketID     = errors.New("invalid basket id")
	ErrInvalidSubTotal     = errors.New("invalid subtotal")
	ErrInvalidBasketStatus = errors.New("invalid basket status")
)
