package basket

import (
	"context"
)

// Repository defines Basket Service persistence behaviour.
type Repository interface {
	CreateBasket(ctx context.Context, query BasketQuery) (BasketID, error) // Newly created basket, return BasketID only.
	GetBasket(ctx context.Context, query BasketQuery) (Basket, error)
	AddItem(ctx context.Context, query BasketQuery) (Basket, error)
	UpdateItemQuantity(ctx context.Context, query BasketQuery) (Basket, error)
	RemoveItem(ctx context.Context, query BasketQuery) (Basket, error)
	ClearBasket(ctx context.Context, query BasketQuery) (Basket, error)
}
