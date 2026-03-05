package model

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// CPFAccount represents a user's CPF account balances
type CPFAccount struct {
	ID        uuid.UUID       `json:"id"`
	UserID    uuid.UUID       `json:"user_id"`
	OABalance decimal.Decimal `json:"oa_balance"`
	SABalance decimal.Decimal `json:"sa_balance"`
	MABalance decimal.Decimal `json:"ma_balance"`
	UpdatedAt time.Time       `json:"updated_at"`
}

// TotalBalance returns the sum of all CPF accounts
func (c *CPFAccount) TotalBalance() decimal.Decimal {
	return c.OABalance.Add(c.SABalance).Add(c.MABalance)
}

// CPFContribution represents a monthly CPF contribution
type CPFContribution struct {
	ID                uuid.UUID       `json:"id"`
	UserID            uuid.UUID       `json:"user_id"`
	ContributionMonth string          `json:"contribution_month"` // Format: YYYY-MM
	EmployeeAmount    decimal.Decimal `json:"employee_amount"`
	EmployerAmount    decimal.Decimal `json:"employer_amount"`
	OAAmount          decimal.Decimal `json:"oa_amount"`
	SAAmount          decimal.Decimal `json:"sa_amount"`
	MAAmount          decimal.Decimal `json:"ma_amount"`
	CreatedAt         time.Time       `json:"created_at"`
}

// TotalAmount returns the sum of employee and employer contributions
func (c *CPFContribution) TotalAmount() decimal.Decimal {
	return c.EmployeeAmount.Add(c.EmployerAmount)
}
