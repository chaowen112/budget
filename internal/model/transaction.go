package model

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// Transaction represents a spending or income record
type Transaction struct {
	ID              uuid.UUID        `json:"id"`
	UserID          uuid.UUID        `json:"user_id"`
	CategoryID      uuid.UUID        `json:"category_id"`
	CategoryName    string           `json:"category_name,omitempty"` // Joined field
	Amount          decimal.Decimal  `json:"amount"`
	Currency        string           `json:"currency"`
	Type            CategoryType     `json:"type"`
	TransactionDate time.Time        `json:"transaction_date"`
	Description     string           `json:"description"`
	Tags            []string         `json:"tags"`
	BudgetAmount    *decimal.Decimal `json:"budget_amount,omitempty"` // Portion that counts toward budget; nil = full amount
	CreatedAt       time.Time        `json:"created_at"`
	UpdatedAt       time.Time        `json:"updated_at"`
}

// EffectiveBudgetAmount returns the amount that counts toward the budget.
func (t *Transaction) EffectiveBudgetAmount() decimal.Decimal {
	if t.BudgetAmount != nil {
		return *t.BudgetAmount
	}
	return t.Amount
}
