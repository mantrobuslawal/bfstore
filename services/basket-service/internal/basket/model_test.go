package basket

import (
	"errors"
	"testing"
)

func TestNormaliseCurrencyCode(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		input         string
		want          CurrencyCode
		wantDefaulted bool
		wantErr       error
	}{
		{name: "empty defaults to GBP", input: "", want: CurrencyCode("GBP"), wantDefaulted: true},
		{name: "whitespace defaults to GBP", input: "  ", want: CurrencyCode("GBP"), wantDefaulted: true},
		{name: "lowercase GBP normalises", input: "gbp", want: CurrencyCode("GBP")},
		{name: "uppercase GBP accepted", input: "GBP", want: CurrencyCode("GBP")},
		{name: "unsupported currency rejected", input: "usd", wantErr: ErrInvalidCurrencyCode},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, defaulted, err := NormaliseCurrencyCode(tt.input)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("NormaliseCurrencyCode(%q) error = %v, want %v", tt.input, err, tt.wantErr)
				}
				return
			}

			if err != nil {
				t.Fatalf("NormaliseCurrencyCode(%q) error = %v, want nil", tt.input, err)
			}

			if got != tt.want {
				t.Fatalf("NormaliseCurrencyCode(%q) = %q, want %q", tt.input, got, tt.want)
			}

			if defaulted != tt.wantDefaulted {
				t.Fatalf("defaulted = %v, want %v", defaulted, tt.wantDefaulted)
			}
		})
	}
}

func TestBasketExistingItem(t *testing.T) {
	t.Parallel()

	b := Basket{
		BasketItems: []*BasketItem{
			nil,
			{ProductID: ProductID("prod_1"), VariantID: VariantID("var_1"), Quantity: 2},
		},
	}

	item, found := b.ExistingItem("prod_1", "var_1")
	if !found {
		t.Fatal("ExistingItem() found = false, want true")
	}

	if item.Quantity != 2 {
		t.Fatalf("Quantity = %d, want 2", item.Quantity)
	}

	_, found = b.ExistingItem("prod_2", "var_2")
	if found {
		t.Fatal("ExistingItem() found = true, want false")
	}
}
