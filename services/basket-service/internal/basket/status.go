package basket

import "strings"

type BasketStatus string

const (
	BasketStatusActive     BasketStatus = "ACTIVE"
	BasketStatusCleared    BasketStatus = "CLEARED"
	BasketStatusExpired    BasketStatus = "EXPIRED"
	BasketStatusCheckedOut BasketStatus = "CHECKED_OUT"
)

func ParseToBasketStatus(status string) (BasketStatus, error) {
	switch strings.ToUpper(strings.TrimSpace(status)) {
	case string(BasketStatusActive):
		return BasketStatusActive, nil
	case string(BasketStatusCleared):
		return BasketStatusCleared, nil
	case string(BasketStatusExpired):
		return BasketStatusExpired, nil
	case string(BasketStatusCheckedOut):
		return BasketStatusCheckedOut, nil
	default:
		return "", ErrInvalidBasketStatus
	}
}
