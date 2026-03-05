package model

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// PeriodType for budgets
type PeriodType string

const (
	PeriodTypeDaily   PeriodType = "daily"
	PeriodTypeWeekly  PeriodType = "weekly"
	PeriodTypeMonthly PeriodType = "monthly"
	PeriodTypeYearly  PeriodType = "yearly"
)

// Budget represents a spending limit for a category
type Budget struct {
	ID           uuid.UUID       `json:"id"`
	UserID       uuid.UUID       `json:"user_id"`
	CategoryID   uuid.UUID       `json:"category_id"`
	CategoryName string          `json:"category_name,omitempty"` // Joined field
	Amount       decimal.Decimal `json:"amount"`
	Currency     string          `json:"currency"`
	PeriodType   PeriodType      `json:"period_type"`
	StartDate    time.Time       `json:"start_date"`
	CreatedAt    time.Time       `json:"created_at"`
	UpdatedAt    time.Time       `json:"updated_at"`
}

// BudgetStatus shows current spending vs budget
type BudgetStatus struct {
	Budget       Budget          `json:"budget"`
	Spent        decimal.Decimal `json:"spent"`
	Remaining    decimal.Decimal `json:"remaining"`
	PercentUsed  float64         `json:"percent_used"`
	IsOverBudget bool            `json:"is_over_budget"`
}
