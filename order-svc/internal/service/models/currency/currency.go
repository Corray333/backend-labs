package currency

import (
	"database/sql/driver"
	"fmt"
)

type Currency string

const (
	CurrencyRUB Currency = "RUB"
)

func (c Currency) String() string {
	return string(c)
}

func (c Currency) Value() (driver.Value, error) {
	return c.String(), nil
}

func ParseCurrency(s string) (Currency, error) {
	switch s {
	case CurrencyRUB.String():
		return CurrencyRUB, nil
	default:
		return "", fmt.Errorf("unknown currency: %s", s)
	}
}
