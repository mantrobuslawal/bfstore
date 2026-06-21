package grpcadapter

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	basketv1 "github.com/mantrobuslawal/bfstore/gen/go/bfstore/basket/v1"
	"github.com/mantrobuslawal/bfstore/services/basket-service/internal/basket"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type handlerTestRepository struct {
	addItemCalled            bool
	updateItemQuantityCalled bool
	removeItemCalled         bool
	clearBasketCalled        bool

	lastAddItemCommand            basket.AddValidatedItemCommand
	lastUpdateItemQuantityCommand basket.UpdateItemQuantityCommand
	lastRemoveItemCommand         basket.RemoveItemCommand
	lastClearBasketCommand        basket.ClearBasketCommand
}

func (r *handlerTestRepository) CreateBasket(ctx context.Context, b basket.Basket) (basket.Basket, error) {
	now := time.Now().UTC()
	b.CreatedAt = now
	b.UpdatedAt = now
	if b.Status == "" {
		b.Status = basket.BasketStatusActive
	}
	if b.Subtotal.CurrencyCode == "" {
		b.Subtotal.CurrencyCode = "GBP"
	}
	return b, nil
}

func (r *handlerTestRepository) GetBasket(ctx context.Context, basketID string) (basket.Basket, error) {
	return basket.Basket{
		BasketID:  basket.BasketID(basketID),
		Status:    basket.BasketStatusActive,
		Subtotal:  basket.Money{CurrencyCode: "GBP"},
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}, nil
}

func (r *handlerTestRepository) AddItem(ctx context.Context, cmd basket.AddValidatedItemCommand) (basket.Basket, error) {
	r.addItemCalled = true
	r.lastAddItemCommand = cmd

	now := time.Now().UTC()
	return basket.Basket{
		BasketID: basket.BasketID(cmd.BasketID),
		Status:   basket.BasketStatusActive,
		Subtotal: basket.Money{
			AmountMinor:  int64(cmd.Quantity) * cmd.UnitPrice.AmountMinor,
			CurrencyCode: cmd.UnitPrice.CurrencyCode,
		},
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
				LineTotal: basket.Money{
					AmountMinor:  int64(cmd.Quantity) * cmd.UnitPrice.AmountMinor,
					CurrencyCode: cmd.UnitPrice.CurrencyCode,
				},
				AddedAt:   now,
				UpdatedAt: now,
			},
		},
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}

func (r *handlerTestRepository) UpdateItemQuantity(ctx context.Context, cmd basket.UpdateItemQuantityCommand) (basket.Basket, error) {
	r.updateItemQuantityCalled = true
	r.lastUpdateItemQuantityCommand = cmd

	now := time.Now().UTC()
	return basket.Basket{
		BasketID:  basket.BasketID(cmd.BasketID),
		Status:    basket.BasketStatusActive,
		Subtotal:  basket.Money{CurrencyCode: "GBP"},
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}

func (r *handlerTestRepository) RemoveItem(ctx context.Context, cmd basket.RemoveItemCommand) (basket.Basket, error) {
	r.removeItemCalled = true
	r.lastRemoveItemCommand = cmd

	now := time.Now().UTC()
	return basket.Basket{
		BasketID:  basket.BasketID(cmd.BasketID),
		Status:    basket.BasketStatusActive,
		Subtotal:  basket.Money{CurrencyCode: "GBP"},
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}

func (r *handlerTestRepository) ClearBasket(ctx context.Context, cmd basket.ClearBasketCommand) (basket.Basket, error) {
	r.clearBasketCalled = true
	r.lastClearBasketCommand = cmd

	now := time.Now().UTC()
	return basket.Basket{
		BasketID:  basket.BasketID(cmd.BasketID),
		Status:    basket.BasketStatusActive,
		Subtotal:  basket.Money{CurrencyCode: "GBP"},
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}

type handlerTestCatalogClient struct{}

func (c *handlerTestCatalogClient) ValidateProductVariant(
	ctx context.Context,
	query basket.ValidateProductVariantQuery,
) (basket.CatalogProductVariant, error) {
	return basket.CatalogProductVariant{
		ProductID:   query.ProductID,
		VariantID:   query.VariantID,
		ProductName: "Go Gopher Mug",
		VariantName: "Blue",
		UnitPrice:   basket.Money{AmountMinor: 1299, CurrencyCode: "GBP"},
		Sellable:    true,
	}, nil
}

func newHandlerUnderTest(repo *handlerTestRepository) *BasketHandler {
	service := basket.NewService(
		repo,
		&handlerTestCatalogClient{},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
	)

	return NewBasketHandler(
		service,
		slog.New(slog.NewTextHandler(io.Discard, nil)),
	)
}

func TestBasketHandlerAddItemCallsAddItem(t *testing.T) {
	t.Parallel()

	repo := &handlerTestRepository{}
	handler := newHandlerUnderTest(repo)

	resp, err := handler.AddItem(context.Background(), &basketv1.AddItemRequest{
		BasketId:  "basket_01ARZ3NDEKTSV4RRFFQ69G5FAV_QG6M8K7Z",
		ProductId: "prod_01ARZ3NDEKTSV4RRFFQ69G5FAV",
		VariantId: "var_01ARZ3NDEKTSV4RRFFQ69G5FAV",
		Quantity:  2,
	})
	if err != nil {
		t.Fatalf("AddItem() error = %v, want nil", err)
	}

	if !repo.addItemCalled {
		t.Fatal("repository AddItem was not called through service")
	}

	if repo.lastAddItemCommand.BasketID != "basket_01ARZ3NDEKTSV4RRFFQ69G5FAV_QG6M8K7Z" {
		t.Fatalf("BasketID = %q, want request basket id", repo.lastAddItemCommand.BasketID)
	}

	if repo.lastAddItemCommand.ProductID != "prod_01ARZ3NDEKTSV4RRFFQ69G5FAV" {
		t.Fatalf("ProductID = %q, want request product id", repo.lastAddItemCommand.ProductID)
	}

	if repo.lastAddItemCommand.VariantID != "var_01ARZ3NDEKTSV4RRFFQ69G5FAV" {
		t.Fatalf("VariantID = %q, want request variant id", repo.lastAddItemCommand.VariantID)
	}

	if repo.lastAddItemCommand.Quantity != 2 {
		t.Fatalf("Quantity = %d, want 2", repo.lastAddItemCommand.Quantity)
	}

	if resp.GetBasket() == nil {
		t.Fatal("response Basket = nil, want basket")
	}
}

func TestBasketHandlerUpdateItemQuantityCallsUpdateItemQuantity(t *testing.T) {
	t.Parallel()

	repo := &handlerTestRepository{}
	handler := newHandlerUnderTest(repo)

	resp, err := handler.UpdateItemQuantity(context.Background(), &basketv1.UpdateItemQuantityRequest{
		BasketId:     "basket_01ARZ3NDEKTSV4RRFFQ69G5FAV_QG6M8K7Z",
		BasketItemId: "bitem_01ARZ3NDEKTSV4RRFFQ69G5FAV_QG6M8K7Z",
		Quantity:     5,
	})
	if err != nil {
		t.Fatalf("UpdateItemQuantity() error = %v, want nil", err)
	}

	if !repo.updateItemQuantityCalled {
		t.Fatal("repository UpdateItemQuantity was not called through service")
	}

	if repo.lastUpdateItemQuantityCommand.BasketItemID != "bitem_01ARZ3NDEKTSV4RRFFQ69G5FAV_QG6M8K7Z" {
		t.Fatalf("BasketItemID = %q, want request basket item id", repo.lastUpdateItemQuantityCommand.BasketItemID)
	}

	if repo.lastUpdateItemQuantityCommand.Quantity != 5 {
		t.Fatalf("Quantity = %d, want 5", repo.lastUpdateItemQuantityCommand.Quantity)
	}

	if resp.GetBasket() == nil {
		t.Fatal("response Basket = nil, want basket")
	}
}

func TestBasketHandlerRemoveItemCallsRemoveItem(t *testing.T) {
	t.Parallel()

	repo := &handlerTestRepository{}
	handler := newHandlerUnderTest(repo)

	resp, err := handler.RemoveItem(context.Background(), &basketv1.RemoveItemRequest{
		BasketId:     "basket_01ARZ3NDEKTSV4RRFFQ69G5FAV_QG6M8K7Z",
		BasketItemId: "bitem_01ARZ3NDEKTSV4RRFFQ69G5FAV_QG6M8K7Z",
	})
	if err != nil {
		t.Fatalf("RemoveItem() error = %v, want nil", err)
	}

	if !repo.removeItemCalled {
		t.Fatal("repository RemoveItem was not called through service")
	}

	if repo.lastRemoveItemCommand.BasketItemID != "bitem_01ARZ3NDEKTSV4RRFFQ69G5FAV_QG6M8K7Z" {
		t.Fatalf("BasketItemID = %q, want request basket item id", repo.lastRemoveItemCommand.BasketItemID)
	}

	if resp.GetBasket() == nil {
		t.Fatal("response Basket = nil, want basket")
	}
}

func TestBasketHandlerClearBasketCallsClearBasket(t *testing.T) {
	t.Parallel()

	repo := &handlerTestRepository{}
	handler := newHandlerUnderTest(repo)

	resp, err := handler.ClearBasket(context.Background(), &basketv1.ClearBasketRequest{
		BasketId: "basket_01ARZ3NDEKTSV4RRFFQ69G5FAV_QG6M8K7Z",
	})
	if err != nil {
		t.Fatalf("ClearBasket() error = %v, want nil", err)
	}

	if !repo.clearBasketCalled {
		t.Fatal("repository ClearBasket was not called through service")
	}

	if repo.lastClearBasketCommand.BasketID != "basket_01ARZ3NDEKTSV4RRFFQ69G5FAV_QG6M8K7Z" {
		t.Fatalf("BasketID = %q, want request basket id", repo.lastClearBasketCommand.BasketID)
	}

	if resp.GetBasket() == nil {
		t.Fatal("response Basket = nil, want basket")
	}
}

func TestBasketHandlerRejectsNilRequests(t *testing.T) {
	t.Parallel()

	handler := newHandlerUnderTest(&handlerTestRepository{})

	tests := []struct {
		name string
		call func() error
	}{
		{
			name: "CreateBasket",
			call: func() error {
				_, err := handler.CreateBasket(context.Background(), nil)
				return err
			},
		},
		{
			name: "GetBasket",
			call: func() error {
				_, err := handler.GetBasket(context.Background(), nil)
				return err
			},
		},
		{
			name: "AddItem",
			call: func() error {
				_, err := handler.AddItem(context.Background(), nil)
				return err
			},
		},
		{
			name: "UpdateItemQuantity",
			call: func() error {
				_, err := handler.UpdateItemQuantity(context.Background(), nil)
				return err
			},
		},
		{
			name: "RemoveItem",
			call: func() error {
				_, err := handler.RemoveItem(context.Background(), nil)
				return err
			},
		},
		{
			name: "ClearBasket",
			call: func() error {
				_, err := handler.ClearBasket(context.Background(), nil)
				return err
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := tt.call()
			if status.Code(err) != codes.InvalidArgument {
				t.Fatalf("status.Code(error) = %v, want %v", status.Code(err), codes.InvalidArgument)
			}
		})
	}
}

func TestBasketHandlerRejectsMissingIDs(t *testing.T) {
	t.Parallel()

	handler := newHandlerUnderTest(&handlerTestRepository{})

	tests := []struct {
		name string
		call func() error
	}{
		{
			name: "GetBasket missing basket id",
			call: func() error {
				_, err := handler.GetBasket(context.Background(), &basketv1.GetBasketRequest{})
				return err
			},
		},
		{
			name: "AddItem missing basket id",
			call: func() error {
				_, err := handler.AddItem(context.Background(), &basketv1.AddItemRequest{
					ProductId: "prod_01ARZ3NDEKTSV4RRFFQ69G5FAV",
					VariantId: "var_01ARZ3NDEKTSV4RRFFQ69G5FAV",
					Quantity:  1,
				})
				return err
			},
		},
		{
			name: "AddItem missing product id",
			call: func() error {
				_, err := handler.AddItem(context.Background(), &basketv1.AddItemRequest{
					BasketId:  "basket_01ARZ3NDEKTSV4RRFFQ69G5FAV_QG6M8K7Z",
					VariantId: "var_01ARZ3NDEKTSV4RRFFQ69G5FAV",
					Quantity:  1,
				})
				return err
			},
		},
		{
			name: "AddItem missing variant id",
			call: func() error {
				_, err := handler.AddItem(context.Background(), &basketv1.AddItemRequest{
					BasketId:  "basket_01ARZ3NDEKTSV4RRFFQ69G5FAV_QG6M8K7Z",
					ProductId: "prod_01ARZ3NDEKTSV4RRFFQ69G5FAV",
					Quantity:  1,
				})
				return err
			},
		},
		{
			name: "UpdateItemQuantity missing item id",
			call: func() error {
				_, err := handler.UpdateItemQuantity(context.Background(), &basketv1.UpdateItemQuantityRequest{
					BasketId: "basket_01ARZ3NDEKTSV4RRFFQ69G5FAV_QG6M8K7Z",
					Quantity: 1,
				})
				return err
			},
		},
		{
			name: "RemoveItem missing item id",
			call: func() error {
				_, err := handler.RemoveItem(context.Background(), &basketv1.RemoveItemRequest{
					BasketId: "basket_01ARZ3NDEKTSV4RRFFQ69G5FAV_QG6M8K7Z",
				})
				return err
			},
		},
		{
			name: "ClearBasket missing basket id",
			call: func() error {
				_, err := handler.ClearBasket(context.Background(), &basketv1.ClearBasketRequest{})
				return err
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := tt.call()
			if status.Code(err) != codes.InvalidArgument {
				t.Fatalf("status.Code(error) = %v, want %v", status.Code(err), codes.InvalidArgument)
			}
		})
	}
}
