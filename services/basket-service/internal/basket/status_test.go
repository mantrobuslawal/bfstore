package basket

import (
	"errors"
	"testing"
)

func TestParseToBasketStatus(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  BasketStatus
		err   error
	}{
		{name: "active uppercase", input: "ACTIVE", want: BasketStatusActive},
		{name: "active lowercase accepted", input: "active", want: BasketStatusActive},
		{name: "cleared", input: "CLEARED", want: BasketStatusCleared},
		{name: "expired", input: "EXPIRED", want: BasketStatusExpired},
		{name: "checked out", input: "CHECKED_OUT", want: BasketStatusCheckedOut},
		{name: "trims whitespace", input: " ACTIVE ", want: BasketStatusActive},
		{name: "old typo rejected", input: "check_out", err: ErrInvalidBasketStatus},
		{name: "empty rejected", input: "", err: ErrInvalidBasketStatus},
		{name: "unknown rejected", input: "goblin", err: ErrInvalidBasketStatus},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := ParseToBasketStatus(tt.input)
			if tt.err != nil {
				if !errors.Is(err, tt.err) {
					t.Fatalf("ParseToBasketStatus(%q) error = %v, want %v", tt.input, err, tt.err)
				}
				return
			}

			if err != nil {
				t.Fatalf("ParseToBasketStatus(%q) error = %v, want nil", tt.input, err)
			}

			if got != tt.want {
				t.Fatalf("ParseToBasketStatus(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
