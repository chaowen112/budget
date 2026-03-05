package model

import (
	"time"

	"github.com/google/uuid"
)

// CategoryType represents expense or income
type CategoryType string

const (
	CategoryTypeExpense CategoryType = "expense"
	CategoryTypeIncome  CategoryType = "income"
)

// Category represents a spending/income category
type Category struct {
	ID        uuid.UUID    `json:"id"`
	UserID    *uuid.UUID   `json:"user_id,omitempty"` // nil for system categories
	Name      string       `json:"name"`
	Type      CategoryType `json:"type"`
	Icon      string       `json:"icon"`
	IsSystem  bool         `json:"is_system"`
	CreatedAt time.Time    `json:"created_at"`
}
