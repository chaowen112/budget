package model

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// AssetCategory for grouping asset types
type AssetCategory string

const (
	AssetCategoryCash       AssetCategory = "cash"
	AssetCategoryBank       AssetCategory = "bank"
	AssetCategoryInvestment AssetCategory = "investment"
	AssetCategoryRetirement AssetCategory = "retirement"
	AssetCategoryProperty   AssetCategory = "property"
	AssetCategoryLiability  AssetCategory = "liability"
	AssetCategoryCustom     AssetCategory = "custom"
)

// AssetType represents a type of asset
type AssetType struct {
	ID       uuid.UUID     `json:"id"`
	Name     string        `json:"name"`
	Category AssetCategory `json:"category"`
	IsSystem bool          `json:"is_system"`
}

// Asset represents a user's asset or liability
type Asset struct {
	ID            uuid.UUID       `json:"id"`
	UserID        uuid.UUID       `json:"user_id"`
	AssetTypeID   uuid.UUID       `json:"asset_type_id"`
	AssetTypeName string          `json:"asset_type_name,omitempty"` // Joined field
	Category      AssetCategory   `json:"category,omitempty"`        // Joined field
	Name          string          `json:"name"`
	Currency      string          `json:"currency"`
	CurrentValue  decimal.Decimal `json:"current_value"`
	IsLiability   bool            `json:"is_liability"`
	CustomFields  json.RawMessage `json:"custom_fields"`
	CreatedAt     time.Time       `json:"created_at"`
	UpdatedAt     time.Time       `json:"updated_at"`
}

// AssetSnapshot for historical tracking
type AssetSnapshot struct {
	ID         uuid.UUID       `json:"id"`
	AssetID    uuid.UUID       `json:"asset_id"`
	Value      decimal.Decimal `json:"value"`
	RecordedAt time.Time       `json:"recorded_at"`
}
