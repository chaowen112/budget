package repository

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/shopspring/decimal"

	"github.com/chaowen/budget/internal/model"
)

var (
	ErrCurrencyNotFound     = errors.New("currency not found")
	ErrExchangeRateNotFound = errors.New("exchange rate not found")
)

type CurrencyRepository struct {
	db *DB
}

func NewCurrencyRepository(db *DB) *CurrencyRepository {
	return &CurrencyRepository{db: db}
}

// ListCurrencies retrieves all currencies
func (r *CurrencyRepository) ListCurrencies(ctx context.Context, activeOnly bool) ([]model.Currency, error) {
	query := `SELECT code, name, symbol, is_active FROM currencies`
	if activeOnly {
		query += ` WHERE is_active = true`
	}
	query += ` ORDER BY code`

	rows, err := r.db.Pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var currencies []model.Currency
	for rows.Next() {
		var c model.Currency
		if err := rows.Scan(&c.Code, &c.Name, &c.Symbol, &c.IsActive); err != nil {
			return nil, err
		}
		currencies = append(currencies, c)
	}

	return currencies, rows.Err()
}

// GetExchangeRate retrieves exchange rate between two currencies
func (r *CurrencyRepository) GetExchangeRate(ctx context.Context, from, to string) (*model.ExchangeRate, error) {
	query := `
		SELECT id, from_currency, to_currency, rate, fetched_at
		FROM exchange_rates
		WHERE from_currency = $1 AND to_currency = $2
	`

	var rate model.ExchangeRate
	err := r.db.Pool.QueryRow(ctx, query, from, to).Scan(
		&rate.ID, &rate.FromCurrency, &rate.ToCurrency, &rate.Rate, &rate.FetchedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrExchangeRateNotFound
		}
		return nil, err
	}

	return &rate, nil
}

// UpsertExchangeRate inserts or updates an exchange rate
func (r *CurrencyRepository) UpsertExchangeRate(ctx context.Context, from, to string, rate decimal.Decimal) error {
	query := `
		INSERT INTO exchange_rates (from_currency, to_currency, rate, fetched_at)
		VALUES ($1, $2, $3, NOW())
		ON CONFLICT (from_currency, to_currency) DO UPDATE
		SET rate = $3, fetched_at = NOW()
	`

	_, err := r.db.Pool.Exec(ctx, query, from, to, rate)
	return err
}

// BulkUpsertExchangeRates inserts or updates multiple exchange rates
func (r *CurrencyRepository) BulkUpsertExchangeRates(ctx context.Context, baseCurrency string, rates map[string]decimal.Decimal) error {
	tx, err := r.db.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	query := `
		INSERT INTO exchange_rates (from_currency, to_currency, rate, fetched_at)
		VALUES ($1, $2, $3, NOW())
		ON CONFLICT (from_currency, to_currency) DO UPDATE
		SET rate = $3, fetched_at = NOW()
	`

	for currency, rate := range rates {
		if _, err := tx.Exec(ctx, query, baseCurrency, currency, rate); err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}
