package model

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// Currency represents a supported currency
type Currency struct {
	Code     string `json:"code"`
	Name     string `json:"name"`
	Symbol   string `json:"symbol"`
	IsActive bool   `json:"is_active"`
}

// ExchangeRate between two currencies
type ExchangeRate struct {
	ID           uuid.UUID       `json:"id"`
	FromCurrency string          `json:"from_currency"`
	ToCurrency   string          `json:"to_currency"`
	Rate         decimal.Decimal `json:"rate"`
	FetchedAt    time.Time       `json:"fetched_at"`
}
