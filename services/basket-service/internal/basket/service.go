package basket

//TODO LOG IN THIS PACKAGE NOT ENOUGH LOGGING

import "context"

// Service contains Basket business logic.
type Service struct{ repository Repository }

// NewService creates a Basket Service
func NewService(repo Repository) *Service {
	return &Service{repository: repo}
}

// CreateBasket creates an empty basket and returns its basket id.
func (s *Service) CreateBasket(ctx context.Context, query BasketQuery) (Basket, error) {
	// TODO: Implement
	return Basket{}, nil
}

// GetBasket takes a basket id and returns the assosicated basket.
func (s *Service) GetBasket(ctx context.Context, query BasketQuery) (Basket, error) {
	// TODO: Implement
	return Basket{}, nil
}

// AddItem adds a line item to an existing basket.
func (s *Service) AddItem(ctx context.Context, query BasketQuery) (Basket, error) {
	// TODO: Implement
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
