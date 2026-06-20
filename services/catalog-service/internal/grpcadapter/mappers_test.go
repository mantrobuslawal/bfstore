package grpcadapter

import (
	"testing"

	"github.com/mantrobuslawal/bfstore/services/catalog-service/internal/catalog"
)

func TestValidatedProductVariantToProto(t *testing.T) {
	t.Parallel()

	resp, err := validatedProductVariantToProto(catalog.ValidatedProductVariant{
		ProductID:   catalog.ProductID("prod_go_gopher_mug"),
		VariantID:   catalog.VariantID("var_blue"),
		ProductName: "Go Gopher Mug",
		VariantName: "Blue",
		UnitPrice: catalog.Money{
			AmountMinor:  1299,
			CurrencyCode: "GBP",
		},
		Sellable: true,
	})
	if err != nil {
		t.Fatalf("validatedProductVariantToProto() error = %v, want nil", err)
	}

	if resp.GetProductId() != "prod_go_gopher_mug" {
		t.Fatalf("ProductId = %q, want %q", resp.GetProductId(), "prod_go_gopher_mug")
	}

	if resp.GetVariantId() != "var_blue" {
		t.Fatalf("VariantId = %q, want %q", resp.GetVariantId(), "var_blue")
	}

	if resp.GetProductName() != "Go Gopher Mug" {
		t.Fatalf("ProductName = %q, want %q", resp.GetProductName(), "Go Gopher Mug")
	}

	if resp.GetVariantName() != "Blue" {
		t.Fatalf("VariantName = %q, want %q", resp.GetVariantName(), "Blue")
	}

	if resp.GetUnitPrice() == nil {
		t.Fatal("UnitPrice = nil, want money proto")
	}

	if resp.GetUnitPrice().GetAmountMinor() != 1299 {
		t.Fatalf("UnitPrice.AmountMinor = %d, want %d", resp.GetUnitPrice().GetAmountMinor(), 1299)
	}

	if resp.GetUnitPrice().GetCurrencyCode() != "GBP" {
		t.Fatalf("UnitPrice.CurrencyCode = %q, want %q", resp.GetUnitPrice().GetCurrencyCode(), "GBP")
	}

	if !resp.GetSellable() {
		t.Fatal("Sellable = false, want true")
	}
}

func TestValidatedProductVariantToProtoRejectsInvalidSnapshot(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input catalog.ValidatedProductVariant
	}{
		{
			name: "missing product id",
			input: catalog.ValidatedProductVariant{
				VariantID:   catalog.VariantID("var_blue"),
				ProductName: "Go Gopher Mug",
				VariantName: "Blue",
				UnitPrice:   catalog.Money{AmountMinor: 1299, CurrencyCode: "GBP"},
				Sellable:    true,
			},
		},
		{
			name: "missing variant id",
			input: catalog.ValidatedProductVariant{
				ProductID:   catalog.ProductID("prod_go_gopher_mug"),
				ProductName: "Go Gopher Mug",
				VariantName: "Blue",
				UnitPrice:   catalog.Money{AmountMinor: 1299, CurrencyCode: "GBP"},
				Sellable:    true,
			},
		},
		{
			name: "missing product name",
			input: catalog.ValidatedProductVariant{
				ProductID:   catalog.ProductID("prod_go_gopher_mug"),
				VariantID:   catalog.VariantID("var_blue"),
				VariantName: "Blue",
				UnitPrice:   catalog.Money{AmountMinor: 1299, CurrencyCode: "GBP"},
				Sellable:    true,
			},
		},
		{
			name: "missing variant name",
			input: catalog.ValidatedProductVariant{
				ProductID:   catalog.ProductID("prod_go_gopher_mug"),
				VariantID:   catalog.VariantID("var_blue"),
				ProductName: "Go Gopher Mug",
				UnitPrice:   catalog.Money{AmountMinor: 1299, CurrencyCode: "GBP"},
				Sellable:    true,
			},
		},
		{
			name: "missing currency code",
			input: catalog.ValidatedProductVariant{
				ProductID:   catalog.ProductID("prod_go_gopher_mug"),
				VariantID:   catalog.VariantID("var_blue"),
				ProductName: "Go Gopher Mug",
				VariantName: "Blue",
				UnitPrice:   catalog.Money{AmountMinor: 1299},
				Sellable:    true,
			},
		},
		{
			name: "negative unit price",
			input: catalog.ValidatedProductVariant{
				ProductID:   catalog.ProductID("prod_go_gopher_mug"),
				VariantID:   catalog.VariantID("var_blue"),
				ProductName: "Go Gopher Mug",
				VariantName: "Blue",
				UnitPrice:   catalog.Money{AmountMinor: -1, CurrencyCode: "GBP"},
				Sellable:    true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			resp, err := validatedProductVariantToProto(tt.input)
			if err == nil {
				t.Fatalf("validatedProductVariantToProto() error = nil, want error")
			}

			if resp != nil {
				t.Fatalf("response = %+v, want nil", resp)
			}
		})
	}
}
