package currency

import (
	"database/sql/driver"
	"errors"
)

type Currency string

const (
	CurrencyRUB Currency = "RUB"
)

var ErrInvalidCurrency = errors.New("invalid currency")

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
		return "", ErrInvalidCurrency
	}
}
