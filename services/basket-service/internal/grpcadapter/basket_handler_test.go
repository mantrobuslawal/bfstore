package grpcadapter

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	basketv1 "github.com/mantrobuslawal/bfstore/gen/go/bfstore/basket/v1"
	"github.com/mantrobuslawal/bfstore/services/basket-service/internal/basket"
)

type handlerTestRepository struct {
	addItemCalled     bool
	clearBasketCalled bool
}

func (r *handlerTestRepository) CreateBasket(ctx context.Context, b basket.Basket) (basket.Basket, error) {
	now := time.Now().UTC()
	b.CreatedAt = now
	b.UpdatedAt = now
	return b, nil
}

func (r *handlerTestRepository) GetBasket(ctx context.Context, basketID string) (basket.Basket, error) {
	return basket.Basket{
		BasketID: basket.BasketID(basketID),
		Status:   basket.BasketStatusActive,
		Subtotal: basket.Money{CurrencyCode: "GBP"},
	}, nil
}

func (r *handlerTestRepository) AddItem(ctx context.Context, cmd basket.AddValidatedItemCommand) (basket.Basket, error) {
	r.addItemCalled = true

	now := time.Now().UTC()
	return basket.Basket{
		BasketID: basket.BasketID(cmd.BasketID),
		Status:   basket.BasketStatusActive,
		Subtotal: basket.Money{AmountMinor: cmd.UnitPrice.AmountMinor * int64(cmd.Quantity), CurrencyCode: cmd.UnitPrice.CurrencyCode},
		BasketItems: []*basket.BasketItem{
			{
				BasketItemID:        basket.BasketItemID("bitem_test"),
				BasketID:            basket.BasketID(cmd.BasketID),
				ProductID:           basket.ProductID(cmd.ProductID),
				VariantID:           basket.VariantID(cmd.VariantID),
				ProductNameSnapshot: cmd.ProductNameSnapshot,
				VariantNameSnapshot: cmd.VariantNameSnapshot,
				Quantity:            cmd.Quantity,
				UnitPrice:           cmd.UnitPrice,
				LineTotal:           basket.Money{AmountMinor: cmd.UnitPrice.AmountMinor * int64(cmd.Quantity), CurrencyCode: cmd.UnitPrice.CurrencyCode},
				AddedAt:             now,
				UpdatedAt:           now,
			},
		},
	}, nil
}

func (r *handlerTestRepository) UpdateItemQuantity(ctx context.Context, cmd basket.UpdateItemQuantityCommand) (basket.Basket, error) {
	return basket.Basket{BasketID: basket.BasketID(cmd.BasketID), Status: basket.BasketStatusActive, Subtotal: basket.Money{CurrencyCode: "GBP"}}, nil
}

func (r *handlerTestRepository) RemoveItem(ctx context.Context, cmd basket.RemoveItemCommand) (basket.Basket, error) {
	return basket.Basket{BasketID: basket.BasketID(cmd.BasketID), Status: basket.BasketStatusActive, Subtotal: basket.Money{CurrencyCode: "GBP"}}, nil
}

func (r *handlerTestRepository) ClearBasket(ctx context.Context, cmd basket.ClearBasketCommand) (basket.Basket, error) {
	r.clearBasketCalled = true
	return basket.Basket{BasketID: basket.BasketID(cmd.BasketID), Status: basket.BasketStatusActive, Subtotal: basket.Money{CurrencyCode: "GBP"}}, nil
}

type handlerTestCatalogClient struct{}

func (c *handlerTestCatalogClient) ValidateProductVariant(ctx context.Context, query basket.ValidateProductVariantQuery) (basket.CatalogProductVariant, error) {
	return basket.CatalogProductVariant{
		ProductID:   query.ProductID,
		VariantID:   query.VariantID,
		ProductName: "Go Gopher Mug",
		VariantName: "Blue",
		UnitPrice:   basket.Money{AmountMinor: 1299, CurrencyCode: "GBP"},
		Sellable:    true,
	}, nil
}

func TestBasketHandlerAddItemCallsAddItem(t *testing.T) {
	t.Parallel()

	repo := &handlerTestRepository{}
	service := basket.NewService(repo, &handlerTestCatalogClient{}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	handler := NewBasketHandler(service, slog.New(slog.NewTextHandler(io.Discard, nil)))

	resp, err := handler.AddItem(context.Background(), &basketv1.AddItemRequest{
		BasketId:  "basket_01ARZ3NDEKTSV4RRFFQ69G5FAV_QG6M8K7Z",
		ProductId: "prod_1",
		VariantId: "var_1",
		Quantity:  1,
	})
	if err != nil {
		t.Fatalf("AddItem() error = %v, want nil", err)
	}

	if !repo.addItemCalled {
		t.Fatal("repository AddItem was not called through service")
	}

	if resp.GetBasket() == nil {
		t.Fatal("response Basket = nil, want basket")
	}
}

func TestBasketHandlerClearBasketCallsClearBasket(t *testing.T) {
	t.Parallel()

	repo := &handlerTestRepository{}
	service := basket.NewService(repo, &handlerTestCatalogClient{}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	handler := NewBasketHandler(service, slog.New(slog.NewTextHandler(io.Discard, nil)))

	resp, err := handler.ClearBasket(context.Background(), &basketv1.ClearBasketRequest{
		BasketId: "basket_01ARZ3NDEKTSV4RRFFQ69G5FAV_QG6M8K7Z",
	})
	if err != nil {
		t.Fatalf("ClearBasket() error = %v, want nil", err)
	}

	if !repo.clearBasketCalled {
		t.Fatal("repository ClearBasket was not called through service")
	}

	if resp.GetBasket() == nil {
		t.Fatal("response Basket = nil, want basket")
	}
}

func TestBasketHandlerRejectsNilRequest(t *testing.T) {
	t.Parallel()

	service := basket.NewService(&handlerTestRepository{}, &handlerTestCatalogClient{}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	handler := NewBasketHandler(service, slog.New(slog.NewTextHandler(io.Discard, nil)))

	if _, err := handler.AddItem(context.Background(), nil); err == nil {
		t.Fatal("AddItem(nil) error = nil, want error")
	}

	if _, err := handler.ClearBasket(context.Background(), nil); err == nil {
		t.Fatal("ClearBasket(nil) error = nil, want error")
	}
}
