package basket

// Service contains Basket business logic.
type Service struct{ repository Repository }

// NewService creates a Basket Service
func NewService(repo Repository) *Service {
	return &Service{repository: repo}
}

// CreateBasket creates an empty basket and returns its basket id.
func (s *Service) CreateBasket() (BasketID, error) {
	// TODO: Implement
	return BasketID(""), nil
}

// GetBasket takes a basket id and returns the assosicated basket.
func (s *Service) GetBasket(id BasketID) (Basket, error) {
	// TODO: Implement
	return Basket{}, nil
}

// AddItem adds a line item to an existing basket.
func (s *Service) AddItem(query BasketQuery) (Basket, error) {
	// TODO: Implement
	return Basket{}, nil
}

// UpdateItemQuantity updates the Quantity field of a bask item within a
// basket and rerturns the updated basket.
func (s *Service) UpdateItemQuantity(query BasketQuery) (Basket, error) {
	// TODO: Implement
	return Basket{}, nil
}

// RemoveItem removes a basket item from an existing basket and returns
// the basket.
func (s *Service) RemoveItem(query BasketQuery) (Basket, error) {
	// TODO: Implement
	return Basket{}, nil
}

// ClearBasket clears all basket items from an existing basket
// and returns the basket.
func (s *Service) ClearBasket(id BasketID) (Basket, error) {
	// TODO: Implement
	return Basket{}, nil
}
