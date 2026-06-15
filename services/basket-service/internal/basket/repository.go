package basket

// Repository defines Basket Service persistence behaviour.
type Repository interface {
	CreateBasket(code CurrencyCode) (BasketID, error) // Newly created basket, return BasketID only.
	GetBasket(id BasketID) (Basket, error)
	AddItem(query BasketQuery) (Basket, error)
	UpdateItemQuantity(query BasketQuery) (Basket, error)
	RemoveItem(query BasketQuery) (Basket, error)
	ClearBasket(id BasketID) (Basket, error)
}
