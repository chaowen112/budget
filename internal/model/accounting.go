package model

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type AccountType string

const (
	AccountTypeAsset     AccountType = "asset"
	AccountTypeLiability AccountType = "liability"
	AccountTypeEquity    AccountType = "equity"
	AccountTypeIncome    AccountType = "income"
	AccountTypeExpense   AccountType = "expense"
)

type Account struct {
	ID             uuid.UUID       `json:"id"`
	UserID         uuid.UUID       `json:"user_id"`
	Name           string          `json:"name"`
	AccountType    AccountType     `json:"account_type"`
	Currency       string          `json:"currency"`
	OpeningBalance decimal.Decimal `json:"opening_balance"`
	AssetID        *uuid.UUID      `json:"asset_id,omitempty"`
	CategoryID     *uuid.UUID      `json:"category_id,omitempty"`
	IsSystem       bool            `json:"is_system"`
	CreatedAt      time.Time       `json:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at"`
}

type AccountWithBalance struct {
	Account
	Balance       decimal.Decimal `json:"balance"`
	AssetTypeName string          `json:"asset_type_name,omitempty"`
}

type JournalEntry struct {
	ID            uuid.UUID    `json:"id"`
	UserID        uuid.UUID    `json:"user_id"`
	EntryDate     time.Time    `json:"entry_date"`
	Description   string       `json:"description"`
	Source        string       `json:"source"`
	ReferenceType string       `json:"reference_type"`
	ReferenceID   *uuid.UUID   `json:"reference_id,omitempty"`
	BaseCurrency  string       `json:"base_currency"`
	CreatedAt     time.Time    `json:"created_at"`
	Lines         []JournalLine `json:"lines"`
}

type JournalLine struct {
	ID          uuid.UUID       `json:"id"`
	EntryID     uuid.UUID       `json:"entry_id"`
	AccountID   uuid.UUID       `json:"account_id"`
	AccountName string          `json:"account_name"`
	AccountType AccountType     `json:"account_type"`
	Debit       decimal.Decimal `json:"debit"`
	Credit      decimal.Decimal `json:"credit"`
	BaseDebit   decimal.Decimal `json:"base_debit"`
	BaseCredit  decimal.Decimal `json:"base_credit"`
	Description string          `json:"description"`
	CreatedAt   time.Time       `json:"created_at"`
}
