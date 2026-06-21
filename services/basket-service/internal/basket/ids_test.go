package basket

import (
	"errors"
	"strings"
	"testing"
)

func TestNewBasketIDAndValidateBasketID(t *testing.T) {
	t.Parallel()

	id, err := NewBasketID()
	if err != nil {
		t.Fatalf("NewBasketID() error = %v, want nil", err)
	}

	if !strings.HasPrefix(id, "basket_") {
		t.Fatalf("NewBasketID() = %q, want basket_ prefix", id)
	}

	if err := ValidateBasketID(id); err != nil {
		t.Fatalf("ValidateBasketID(%q) error = %v, want nil", id, err)
	}
}

func TestNewBasketItemIDAndValidateBasketItemID(t *testing.T) {
	t.Parallel()

	id, err := NewBasketItemID()
	if err != nil {
		t.Fatalf("NewBasketItemID() error = %v, want nil", err)
	}

	if !strings.HasPrefix(id, "bitem_") {
		t.Fatalf("NewBasketItemID() = %q, want bitem_ prefix", id)
	}

	if err := ValidateBasketItemID(id); err != nil {
		t.Fatalf("ValidateBasketItemID(%q) error = %v, want nil", id, err)
	}
}

func TestValidatePrefixedIDRejectsWrongPrefix(t *testing.T) {
	t.Parallel()

	id, err := NewBasketID()
	if err != nil {
		t.Fatalf("NewBasketID() error = %v, want nil", err)
	}

	err = ValidateBasketItemID(id)
	if !errors.Is(err, ErrInvalidIDPrefix) {
		t.Fatalf("ValidateBasketItemID(%q) error = %v, want %v", id, err, ErrInvalidIDPrefix)
	}
}

func TestValidatePrefixedIDRejectsBadChecksum(t *testing.T) {
	t.Parallel()

	id, err := NewBasketID()
	if err != nil {
		t.Fatalf("NewBasketID() error = %v, want nil", err)
	}

	badID := id[:len(id)-1] + "0"
	err = ValidateBasketID(badID)
	if !errors.Is(err, ErrInvalidIDChecksum) {
		t.Fatalf("ValidateBasketID(%q) error = %v, want %v", badID, err, ErrInvalidIDChecksum)
	}
}
