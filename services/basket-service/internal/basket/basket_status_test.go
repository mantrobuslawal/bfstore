package basket

import (
	"errors"
	"testing"
)

func TestParseToBasketStatus(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantStatus BasketStatus
		wantErr    error
	}{
		{
			name:       "parses draft",
			input:      "draft",
			wantStatus: BasketStatusDraft,
		},
		{
			name:       "parses active",
			input:      "active",
			wantStatus: BasketStatusActive,
		},
		{
			name:       "parses cleared",
			input:      "cleared",
			wantStatus: BasketStatusCleared,
		},
		{
			name:       "parses expired",
			input:      "expired",
			wantStatus: BasketStatusExpired,
		},
		{
			name:       "parses checked out",
			input:      "checked_out",
			wantStatus: BasketStatusCheckedOut,
		},
		{
			name:       "trims surrounding whitespace",
			input:      "  active  ",
			wantStatus: BasketStatusActive,
		},
		{
			name:    "rejects empty string",
			input:   "",
			wantErr: ErrInvalidBasketStatus,
		},
		{
			name:    "rejects whitespace only",
			input:   "   ",
			wantErr: ErrInvalidBasketStatus,
		},
		{
			name:    "rejects unknown status",
			input:   "unknown",
			wantErr: ErrInvalidBasketStatus,
		},
		{
			name:    "rejects misspelled checked out value",
			input:   "check_out",
			wantErr: ErrInvalidBasketStatus,
		},
		{
			name:    "rejects uppercase active",
			input:   "ACTIVE",
			wantErr: ErrInvalidBasketStatus,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotStatus, err := ParseToBasketStatus(tt.input)

			if tt.wantErr != nil {
				if err == nil {
					t.Fatalf("ParseToBasketStatus(%q) error = nil, want %v", tt.input, tt.wantErr)
				}

				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("ParseToBasketStatus(%q) error = %v, want %v", tt.input, err, tt.wantErr)
				}

				if gotStatus != "" {
					t.Fatalf("ParseToBasketStatus(%q) status = %q, want empty status", tt.input, gotStatus)
				}

				return
			}

			if err != nil {
				t.Fatalf("ParseToBasketStatus(%q) error = %v, want nil", tt.input, err)
			}

			if gotStatus != tt.wantStatus {
				t.Fatalf("ParseToBasketStatus(%q) status = %q, want %q", tt.input, gotStatus, tt.wantStatus)
			}
		})
	}
}
