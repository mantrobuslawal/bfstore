package basket

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"
)

type fakeRepository struct {
	createBasketFn       func(context.Context, Basket) (Basket, error)
	getBasketFn          func(context.Context, string) (Basket, error)
	addItemFn            func(context.Context, AddValidatedItemCommand) (Basket, error)
	updateItemQuantityFn func(context.Context, UpdateItemQuantityCommand) (Basket, error)
	removeItemFn         func(context.Context, RemoveItemCommand) (Basket, error)
	clearBasketFn        func(context.Context, ClearBasketCommand) (Basket, error)

	addItemCalled            bool
	updateItemQuantityCalled bool
	removeItemCalled         bool
	clearBasketCalled        bool
}

func (f *fakeRepository) CreateBasket(ctx context.Context, basket Basket) (Basket, error) {
	if f.createBasketFn != nil {
		return f.createBasketFn(ctx, basket)
	}
	return basket, nil
}

func (f *fakeRepository) GetBasket(ctx context.Context, basketID string) (Basket, error) {
	if f.getBasketFn != nil {
		return f.getBasketFn(ctx, basketID)
	}
	return Basket{BasketID: BasketID(basketID), Status: BasketStatusActive, Subtotal: Money{CurrencyCode: "GBP"}}, nil
}

func (f *fakeRepository) AddItem(ctx context.Context, cmd AddValidatedItemCommand) (Basket, error) {
	f.addItemCalled = true
	if f.addItemFn != nil {
		return f.addItemFn(ctx, cmd)
	}
	return Basket{BasketID: BasketID(cmd.BasketID), Status: BasketStatusActive, Subtotal: cmd.UnitPrice}, nil
}

func (f *fakeRepository) UpdateItemQuantity(ctx context.Context, cmd UpdateItemQuantityCommand) (Basket, error) {
	f.updateItemQuantityCalled = true
	if f.updateItemQuantityFn != nil {
		return f.updateItemQuantityFn(ctx, cmd)
	}
	return Basket{BasketID: BasketID(cmd.BasketID), Status: BasketStatusActive, Subtotal: Money{CurrencyCode: "GBP"}}, nil
}

func (f *fakeRepository) RemoveItem(ctx context.Context, cmd RemoveItemCommand) (Basket, error) {
	f.removeItemCalled = true
	if f.removeItemFn != nil {
		return f.removeItemFn(ctx, cmd)
	}
	return Basket{BasketID: BasketID(cmd.BasketID), Status: BasketStatusActive, Subtotal: Money{CurrencyCode: "GBP"}}, nil
}

func (f *fakeRepository) ClearBasket(ctx context.Context, cmd ClearBasketCommand) (Basket, error) {
	f.clearBasketCalled = true
	if f.clearBasketFn != nil {
		return f.clearBasketFn(ctx, cmd)
	}
	return Basket{BasketID: BasketID(cmd.BasketID), Status: BasketStatusActive, Subtotal: Money{CurrencyCode: "GBP"}}, nil
}

type fakeCatalogClient struct {
	result CatalogProductVariant
	err    error
	called bool
}

func (f *fakeCatalogClient) ValidateProductVariant(
	ctx context.Context,
	query ValidateProductVariantQuery,
) (CatalogProductVariant, error) {
	f.called = true
	if f.err != nil {
		return CatalogProductVariant{}, f.err
	}
	return f.result, nil
}

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestServiceCreateBasketDefaultsCurrency(t *testing.T) {
	t.Parallel()

	repo := &fakeRepository{}
	service := NewService(repo, &fakeCatalogClient{}, testLogger())

	got, err := service.CreateBasket(context.Background(), BasketQuery{})
	if err != nil {
		t.Fatalf("CreateBasket() error = %v, want nil", err)
	}

	if got.Subtotal.CurrencyCode != "GBP" {
		t.Fatalf("currency = %q, want GBP", got.Subtotal.CurrencyCode)
	}

	if got.Status != BasketStatusActive {
		t.Fatalf("status = %q, want active", got.Status)
	}
}

func TestServiceAddItemCallsCatalogAndRepository(t *testing.T) {
	t.Parallel()

	basketID, _ := NewBasketID()

	repo := &fakeRepository{
		getBasketFn: func(ctx context.Context, id string) (Basket, error) {
			return Basket{
				BasketID: BasketID(id),
				Status:   BasketStatusActive,
				Subtotal: Money{CurrencyCode: "GBP"},
			}, nil
		},
		addItemFn: func(ctx context.Context, cmd AddValidatedItemCommand) (Basket, error) {
			if cmd.ProductNameSnapshot != "Go Gopher Mug" {
				t.Fatalf("ProductNameSnapshot = %q, want %q", cmd.ProductNameSnapshot, "Go Gopher Mug")
			}
			if cmd.VariantNameSnapshot != "Blue" {
				t.Fatalf("VariantNameSnapshot = %q, want %q", cmd.VariantNameSnapshot, "Blue")
			}
			return Basket{
				BasketID: BasketID(cmd.BasketID),
				Status:   BasketStatusActive,
				Subtotal: Money{AmountMinor: 1299, CurrencyCode: "GBP"},
				BasketItems: []*BasketItem{
					{
						BasketItemID:        BasketItemID("bitem_test"),
						ProductID:           ProductID(cmd.ProductID),
						VariantID:           VariantID(cmd.VariantID),
						ProductNameSnapshot: cmd.ProductNameSnapshot,
						VariantNameSnapshot: cmd.VariantNameSnapshot,
						Quantity:            cmd.Quantity,
						UnitPrice:           cmd.UnitPrice,
						LineTotal:           Money{AmountMinor: 1299, CurrencyCode: "GBP"},
						AddedAt:             time.Now(),
					},
				},
			}, nil
		},
	}

	catalogClient := &fakeCatalogClient{
		result: CatalogProductVariant{
			ProductID:   "prod_1",
			VariantID:   "var_1",
			ProductName: "Go Gopher Mug",
			VariantName: "Blue",
			UnitPrice:   Money{AmountMinor: 1299, CurrencyCode: "GBP"},
			Sellable:    true,
		},
	}

	service := NewService(repo, catalogClient, testLogger())

	got, err := service.AddItem(context.Background(), BasketQuery{
		BasketID:  basketID,
		ProductID: "prod_1",
		VariantID: "var_1",
		Quantity:  1,
	})
	if err != nil {
		t.Fatalf("AddItem() error = %v, want nil", err)
	}

	if !catalogClient.called {
		t.Fatal("catalog client was not called")
	}

	if !repo.addItemCalled {
		t.Fatal("repository AddItem was not called")
	}

	if len(got.BasketItems) != 1 {
		t.Fatalf("len(BasketItems) = %d, want 1", len(got.BasketItems))
	}
}

func TestServiceAddItemRejectsCombinedQuantityOverLimit(t *testing.T) {
	t.Parallel()

	basketID, _ := NewBasketID()

	repo := &fakeRepository{
		getBasketFn: func(ctx context.Context, id string) (Basket, error) {
			return Basket{
				BasketID: BasketID(id),
				Status:   BasketStatusActive,
				Subtotal: Money{CurrencyCode: "GBP"},
				BasketItems: []*BasketItem{
					{ProductID: "prod_1", VariantID: "var_1", Quantity: 98},
				},
			}, nil
		},
	}

	catalogClient := &fakeCatalogClient{}
	service := NewService(repo, catalogClient, testLogger())

	_, err := service.AddItem(context.Background(), BasketQuery{
		BasketID:  basketID,
		ProductID: "prod_1",
		VariantID: "var_1",
		Quantity:  2,
	})
	if !errors.Is(err, ErrInvalidQuantity) {
		t.Fatalf("AddItem() error = %v, want %v", err, ErrInvalidQuantity)
	}

	if catalogClient.called {
		t.Fatal("catalog client called despite invalid combined quantity")
	}
}

func TestServiceUpdateItemQuantityUsesRepositoryNotRecursion(t *testing.T) {
	t.Parallel()

	basketID, _ := NewBasketID()
	itemID, _ := NewBasketItemID()

	repo := &fakeRepository{}
	service := NewService(repo, &fakeCatalogClient{}, testLogger())

	_, err := service.UpdateItemQuantity(context.Background(), BasketQuery{
		BasketID:     basketID,
		BasketItemID: itemID,
		Quantity:     3,
	})
	if err != nil {
		t.Fatalf("UpdateItemQuantity() error = %v, want nil", err)
	}

	if !repo.updateItemQuantityCalled {
		t.Fatal("repository UpdateItemQuantity was not called")
	}
}

func TestServiceRemoveItemAndClearBasketUseCorrectRepositoryMethods(t *testing.T) {
	t.Parallel()

	basketID, _ := NewBasketID()
	itemID, _ := NewBasketItemID()

	repo := &fakeRepository{}
	service := NewService(repo, &fakeCatalogClient{}, testLogger())

	if _, err := service.RemoveItem(context.Background(), BasketQuery{BasketID: basketID, BasketItemID: itemID}); err != nil {
		t.Fatalf("RemoveItem() error = %v, want nil", err)
	}
	if !repo.removeItemCalled {
		t.Fatal("repository RemoveItem was not called")
	}

	if _, err := service.ClearBasket(context.Background(), BasketQuery{BasketID: basketID}); err != nil {
		t.Fatalf("ClearBasket() error = %v, want nil", err)
	}
	if !repo.clearBasketCalled {
		t.Fatal("repository ClearBasket was not called")
	}
}
