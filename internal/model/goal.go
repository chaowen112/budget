package model

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// SavingGoal represents a savings target
type SavingGoal struct {
	ID             uuid.UUID       `json:"id"`
	UserID         uuid.UUID       `json:"user_id"`
	Name           string          `json:"name"`
	TargetAmount   decimal.Decimal `json:"target_amount"`
	CurrentAmount  decimal.Decimal `json:"current_amount"`
	Currency       string          `json:"currency"`
	Deadline       *time.Time      `json:"deadline,omitempty"`
	LinkedAssetIDs []uuid.UUID     `json:"linked_asset_ids"`
	Notes          string          `json:"notes"`
	CreatedAt      time.Time       `json:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at"`
}

// GoalProgressSnapshot stores historical progress points for a goal.
type GoalProgressSnapshot struct {
	ID         uuid.UUID       `json:"id"`
	GoalID     uuid.UUID       `json:"goal_id"`
	Amount     decimal.Decimal `json:"amount"`
	RecordedAt time.Time       `json:"recorded_at"`
}

// GoalContribution stores each add/remove money event for a goal.
type GoalContribution struct {
	ID           uuid.UUID       `json:"id"`
	GoalID       uuid.UUID       `json:"goal_id"`
	AmountDelta  decimal.Decimal `json:"amount_delta"`
	BalanceAfter decimal.Decimal `json:"balance_after"`
	Source       string          `json:"source"`
	RecordedAt   time.Time       `json:"recorded_at"`
}

// PercentageComplete returns the completion percentage
func (g *SavingGoal) PercentageComplete() float64 {
	if g.TargetAmount.IsZero() {
		return 0
	}
	return g.CurrentAmount.Div(g.TargetAmount).InexactFloat64() * 100
}

// AmountRemaining returns how much is left to save
func (g *SavingGoal) AmountRemaining() decimal.Decimal {
	remaining := g.TargetAmount.Sub(g.CurrentAmount)
	if remaining.IsNegative() {
		return decimal.Zero
	}
	return remaining
}

// NetWorthGoal represents a user's target net worth milestone
type NetWorthGoal struct {
	ID           uuid.UUID       `json:"id"`
	UserID       uuid.UUID       `json:"user_id"`
	Name         string          `json:"name"`
	TargetAmount decimal.Decimal `json:"target_amount"`
	Currency     string          `json:"currency"`
	Notes        string          `json:"notes"`
	CreatedAt    time.Time       `json:"created_at"`
	UpdatedAt    time.Time       `json:"updated_at"`
}

// PercentageComplete returns the completion percentage given current net worth
func (g *NetWorthGoal) PercentageComplete(currentNetWorth decimal.Decimal) float64 {
	if g.TargetAmount.IsZero() {
		return 0
	}
	pct := currentNetWorth.Div(g.TargetAmount).InexactFloat64() * 100
	if pct > 100 {
		return 100
	}
	if pct < 0 {
		return 0
	}
	return pct
}

// AmountRemaining returns how much is left to reach the goal
func (g *NetWorthGoal) AmountRemaining(currentNetWorth decimal.Decimal) decimal.Decimal {
	remaining := g.TargetAmount.Sub(currentNetWorth)
	if remaining.IsNegative() {
		return decimal.Zero
	}
	return remaining
}
