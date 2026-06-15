package basket

import "strings"

type BasketStatus string

const (
	BasketStatusDraft      = "draft"
	BasketStatusActive     = "active"
	BasketStatusCleared    = "cleared"
	BasketStatusExpired    = "expired"
	BasketStatusCheckedOut = "checked_out"
)

func ParseToBasketStatus(status string) (BasketStatus, error) {
	status = strings.TrimSpace(status)
	switch status {
	case "draft":
		return BasketStatusDraft, nil
	case "active":
		return BasketStatusActive, nil
	case "cleared":
		return BasketStatusCleared, nil
	case "expired":
		return BasketStatusExpired, nil
	case "check_out":
		return BasketStatusCheckedOut, nil

	default:
		return "", ErrInvalidBasketStatus
	}
}
