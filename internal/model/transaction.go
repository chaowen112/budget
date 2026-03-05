package model

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// Transaction represents a spending or income record
type Transaction struct {
	ID              uuid.UUID       `json:"id"`
	UserID          uuid.UUID       `json:"user_id"`
	CategoryID      uuid.UUID       `json:"category_id"`
	CategoryName    string          `json:"category_name,omitempty"` // Joined field
	Amount          decimal.Decimal `json:"amount"`
	Currency        string          `json:"currency"`
	Type            CategoryType    `json:"type"`
	TransactionDate time.Time       `json:"transaction_date"`
	Description     string          `json:"description"`
	Tags            []string        `json:"tags"`
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
}
