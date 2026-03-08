package repository

import (
	"context"
	"errors"
	"strings"
	"time"

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

	tx, err := r.db.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, query, from, to, rate); err != nil {
		return err
	}

	historyQuery := `
		INSERT INTO exchange_rate_history (from_currency, to_currency, rate, fetched_at)
		VALUES ($1, $2, $3, NOW())
	`
	if _, err := tx.Exec(ctx, historyQuery, from, to, rate); err != nil {
		return err
	}

	return tx.Commit(ctx)
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
	historyQuery := `
		INSERT INTO exchange_rate_history (from_currency, to_currency, rate, fetched_at)
		VALUES ($1, $2, $3, NOW())
	`

	for currency, rate := range rates {
		if _, err := tx.Exec(ctx, query, baseCurrency, currency, rate); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, historyQuery, baseCurrency, currency, rate); err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}

// GetExchangeRateAsOf retrieves the most recent exchange rate at/before a timestamp.
func (r *CurrencyRepository) GetExchangeRateAsOf(ctx context.Context, from, to string, asOf time.Time) (*model.ExchangeRate, error) {
	query := `
		SELECT id, from_currency, to_currency, rate, fetched_at
		FROM exchange_rate_history
		WHERE from_currency = $1 AND to_currency = $2 AND fetched_at <= $3
		ORDER BY fetched_at DESC
		LIMIT 1
	`

	var rate model.ExchangeRate
	err := r.db.Pool.QueryRow(ctx, query, from, to, asOf).Scan(
		&rate.ID, &rate.FromCurrency, &rate.ToCurrency, &rate.Rate, &rate.FetchedAt,
	)
	if err == nil {
		return &rate, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}

	return r.GetExchangeRate(ctx, from, to)
}

// ConvertAmount converts an amount from one currency to another using stored rates.
// It first tries direct rate (from->to), then inverse (to->from).
func (r *CurrencyRepository) ConvertAmount(ctx context.Context, amount decimal.Decimal, from, to string) (decimal.Decimal, error) {
	from = strings.ToUpper(strings.TrimSpace(from))
	to = strings.ToUpper(strings.TrimSpace(to))

	if from == "" || to == "" {
		return decimal.Zero, ErrExchangeRateNotFound
	}
	if from == to {
		return amount.Round(2), nil
	}

	direct, err := r.GetExchangeRate(ctx, from, to)
	if err == nil {
		return amount.Mul(direct.Rate).Round(2), nil
	}
	if !errors.Is(err, ErrExchangeRateNotFound) {
		return decimal.Zero, err
	}

	inverse, invErr := r.GetExchangeRate(ctx, to, from)
	if invErr == nil {
		if inverse.Rate.IsZero() {
			return decimal.Zero, errors.New("invalid zero exchange rate")
		}
		return amount.Div(inverse.Rate).Round(2), nil
	}
	if errors.Is(invErr, ErrExchangeRateNotFound) {
		return decimal.Zero, ErrExchangeRateNotFound
	}

	return decimal.Zero, invErr
}

// ConvertAmountAsOf converts an amount using the latest rate at/before a timestamp.
func (r *CurrencyRepository) ConvertAmountAsOf(ctx context.Context, amount decimal.Decimal, from, to string, asOf time.Time) (decimal.Decimal, error) {
	from = strings.ToUpper(strings.TrimSpace(from))
	to = strings.ToUpper(strings.TrimSpace(to))

	if from == "" || to == "" {
		return decimal.Zero, ErrExchangeRateNotFound
	}
	if from == to {
		return amount.Round(2), nil
	}

	direct, err := r.GetExchangeRateAsOf(ctx, from, to, asOf)
	if err == nil {
		return amount.Mul(direct.Rate).Round(2), nil
	}
	if !errors.Is(err, ErrExchangeRateNotFound) {
		return decimal.Zero, err
	}

	inverse, invErr := r.GetExchangeRateAsOf(ctx, to, from, asOf)
	if invErr == nil {
		if inverse.Rate.IsZero() {
			return decimal.Zero, errors.New("invalid zero exchange rate")
		}
		return amount.Div(inverse.Rate).Round(2), nil
	}
	if errors.Is(invErr, ErrExchangeRateNotFound) {
		return decimal.Zero, ErrExchangeRateNotFound
	}

	return decimal.Zero, invErr
}
