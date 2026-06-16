package basket

import (
	"strings"
	"testing"
)

func TestNewBasketID(t *testing.T) {
	id, err := NewBasketID()
	if err != nil {
		t.Fatalf("NewBasketID() error = %v", err)
	}

	if !strings.HasPrefix(id, "basket_") {
		t.Fatalf("expected basket_ prefix, got %q", id)
	}

	if err := ValidateBasketID(id); err != nil {
		t.Fatalf("ValidateBasketID() error = %v", err)
	}
}

func TestNewBasketItemID(t *testing.T) {
	id, err := NewBasketItemID()
	if err != nil {
		t.Fatalf("NewBasketItemID() error = %v", err)
	}

	if !strings.HasPrefix(id, "bitem_") {
		t.Fatalf("expected bitem_ prefix, got %q", id)
	}

	if err := ValidateBasketItemID(id); err != nil {
		t.Fatalf("ValidateBasketItemID() error = %v", err)
	}
}

func TestValidateBasketIDRejectsWrongPrefix(t *testing.T) {
	id, err := NewBasketID()
	if err != nil {
		t.Fatalf("NewBasketID() error = %v", err)
	}

	badID := strings.Replace(id, "basket_", "bitem_", 1)

	if err := ValidateBasketID(badID); err == nil {
		t.Fatal("expected wrong prefix to be rejected")
	}
}

func TestValidateBasketIDRejectsTamperedPayload(t *testing.T) {
	id, err := NewBasketID()
	if err != nil {
		t.Fatalf("NewBasketID() error = %v", err)
	}

	parts := strings.Split(id, "_")
	parts[1] = parts[1][:len(parts[1])-1] + "0"

	badID := strings.Join(parts, "_")

	if err := ValidateBasketID(badID); err == nil {
		t.Fatal("expected tampered payload to be rejected")
	}
}

func TestValidateBasketIDRejectsTamperedChecksum(t *testing.T) {
	id, err := NewBasketID()
	if err != nil {
		t.Fatalf("NewBasketID() error = %v", err)
	}

	parts := strings.Split(id, "_")
	parts[2] = parts[2][:len(parts[2])-1] + "0"

	badID := strings.Join(parts, "_")

	if err := ValidateBasketID(badID); err == nil {
		t.Fatal("expected tampered checksum to be rejected")
	}
}
