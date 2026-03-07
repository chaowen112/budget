package model

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type Transfer struct {
	ID          uuid.UUID       `json:"id"`
	UserID      uuid.UUID       `json:"user_id"`
	FromAssetID uuid.UUID       `json:"from_asset_id"`
	ToAssetID   uuid.UUID       `json:"to_asset_id"`
	FromAmount  decimal.Decimal `json:"from_amount"`
	ToAmount    decimal.Decimal `json:"to_amount"`
	FromCurrency string         `json:"from_currency"`
	ToCurrency   string         `json:"to_currency"`
	ExchangeRate decimal.Decimal `json:"exchange_rate"`
	TransferDate time.Time      `json:"transfer_date"`
	Description  string         `json:"description"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`

	FromAssetName string `json:"from_asset_name,omitempty"`
	ToAssetName   string `json:"to_asset_name,omitempty"`
}
