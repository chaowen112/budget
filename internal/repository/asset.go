package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/shopspring/decimal"

	"github.com/chaowen/budget/internal/model"
)

var (
	ErrAssetNotFound     = errors.New("asset not found")
	ErrAssetTypeNotFound = errors.New("asset type not found")
	ErrAssetInUse        = errors.New("asset has linked records")
)

type AssetRepository struct {
	db *DB
}

type AssetDeleteBlocker struct {
	Kind        string
	ReferenceID uuid.UUID
	Description string
	OccurredAt  time.Time
}

type AssetBalanceAsOf struct {
	Currency    string
	IsLiability bool
	Balance     decimal.Decimal
}

// ListBalancesAsOf retrieves per-asset balances at a specific point in time.
func (r *AssetRepository) ListBalancesAsOf(ctx context.Context, userID uuid.UUID, asOf time.Time) ([]AssetBalanceAsOf, error) {
	query := `
		SELECT
			a.currency,
			a.is_liability,
			COALESCE(
				CASE
					WHEN acc.id IS NULL THEN a.current_value
					WHEN acc.account_type IN ('asset', 'expense')
						THEN acc.opening_balance + COALESCE(agg.total_debit, 0) - COALESCE(agg.total_credit, 0)
					ELSE acc.opening_balance + COALESCE(agg.total_credit, 0) - COALESCE(agg.total_debit, 0)
				END,
				a.current_value
			) AS balance
		FROM assets a
		LEFT JOIN accounts acc ON acc.asset_id = a.id
		LEFT JOIN LATERAL (
			SELECT
				COALESCE(SUM(jl.debit), 0) AS total_debit,
				COALESCE(SUM(jl.credit), 0) AS total_credit
			FROM journal_lines jl
			JOIN journal_entries je ON je.id = jl.entry_id
			WHERE jl.account_id = acc.id
			  AND je.user_id = a.user_id
			  AND je.entry_date <= $2
		) agg ON true
		WHERE a.user_id = $1
	`

	rows, err := r.db.Pool.Query(ctx, query, userID, asOf)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var balances []AssetBalanceAsOf
	for rows.Next() {
		var item AssetBalanceAsOf
		if err := rows.Scan(&item.Currency, &item.IsLiability, &item.Balance); err != nil {
			return nil, err
		}
		balances = append(balances, item)
	}

	return balances, rows.Err()
}

// GetTotalsAsOf calculates total assets and liabilities as of a specific point in time.
// It uses the latest snapshot recorded on/before asOf per asset, falling back to current_value.
func (r *AssetRepository) GetTotalsAsOf(ctx context.Context, userID uuid.UUID, asOf time.Time) (decimal.Decimal, decimal.Decimal, error) {
	balances, err := r.ListBalancesAsOf(ctx, userID, asOf)
	if err != nil {
		return decimal.Zero, decimal.Zero, err
	}

	totalAssets := decimal.Zero
	totalLiabilities := decimal.Zero
	for _, item := range balances {
		if item.IsLiability {
			totalLiabilities = totalLiabilities.Add(item.Balance)
			continue
		}
		totalAssets = totalAssets.Add(item.Balance)
	}

	return totalAssets, totalLiabilities, nil
}

func NewAssetRepository(db *DB) *AssetRepository {
	return &AssetRepository{db: db}
}

// ListAssetTypes retrieves all asset types
func (r *AssetRepository) ListAssetTypes(ctx context.Context, category *model.AssetCategory) ([]model.AssetType, error) {
	query := `
		SELECT id, name, category, is_system
		FROM asset_types
		WHERE name NOT IN ('Mutual Fund', 'Robo-Advisor', '401k', 'IRA')
	`
	args := []any{}

	if category != nil {
		query += ` AND category = $1`
		args = append(args, *category)
	}

	query += ` ORDER BY category, name`

	rows, err := r.db.Pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var types []model.AssetType
	for rows.Next() {
		var t model.AssetType
		if err := rows.Scan(&t.ID, &t.Name, &t.Category, &t.IsSystem); err != nil {
			return nil, err
		}
		types = append(types, t)
	}

	return types, rows.Err()
}

// List retrieves all assets for a user
func (r *AssetRepository) List(ctx context.Context, userID uuid.UUID, category *model.AssetCategory, includeLiabilities bool) ([]model.Asset, error) {
	query := `
		SELECT a.id, a.user_id, a.asset_type_id, t.name, t.category, a.name, a.currency, a.cost,
		       COALESCE(
				CASE
					WHEN acc.id IS NULL THEN a.current_value
					WHEN acc.account_type IN ('asset', 'expense')
						THEN acc.opening_balance + COALESCE(agg.total_debit, 0) - COALESCE(agg.total_credit, 0)
					ELSE acc.opening_balance + COALESCE(agg.total_credit, 0) - COALESCE(agg.total_debit, 0)
				END,
				a.current_value
			) AS current_value,
		       a.is_liability, a.custom_fields, a.created_at, a.updated_at
		FROM assets a
		JOIN asset_types t ON a.asset_type_id = t.id
		LEFT JOIN accounts acc ON acc.asset_id = a.id
		LEFT JOIN LATERAL (
			SELECT
				COALESCE(SUM(jl.debit), 0) AS total_debit,
				COALESCE(SUM(jl.credit), 0) AS total_credit
			FROM journal_lines jl
			JOIN journal_entries je ON je.id = jl.entry_id
			WHERE jl.account_id = acc.id
			  AND je.user_id = a.user_id
		) agg ON true
		WHERE a.user_id = $1
	`
	args := []any{userID}
	argIndex := 2

	if category != nil {
		query += ` AND t.category = $` + string(rune('0'+argIndex))
		args = append(args, *category)
		argIndex++
	}

	if !includeLiabilities {
		query += ` AND a.is_liability = false`
	}

	query += ` ORDER BY a.is_liability ASC, t.category, a.name`

	rows, err := r.db.Pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var assets []model.Asset
	for rows.Next() {
		var a model.Asset
		err := rows.Scan(
			&a.ID, &a.UserID, &a.AssetTypeID, &a.AssetTypeName, &a.Category, &a.Name,
			&a.Currency, &a.Cost, &a.CurrentValue, &a.IsLiability, &a.CustomFields, &a.CreatedAt, &a.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		assets = append(assets, a)
	}

	return assets, rows.Err()
}

// GetByID retrieves an asset by ID
func (r *AssetRepository) GetByID(ctx context.Context, id, userID uuid.UUID) (*model.Asset, error) {
	query := `
		SELECT a.id, a.user_id, a.asset_type_id, t.name, t.category, a.name, a.currency, a.cost,
		       COALESCE(
				CASE
					WHEN acc.id IS NULL THEN a.current_value
					WHEN acc.account_type IN ('asset', 'expense')
						THEN acc.opening_balance + COALESCE(agg.total_debit, 0) - COALESCE(agg.total_credit, 0)
					ELSE acc.opening_balance + COALESCE(agg.total_credit, 0) - COALESCE(agg.total_debit, 0)
				END,
				a.current_value
			) AS current_value,
		       a.is_liability, a.custom_fields, a.created_at, a.updated_at
		FROM assets a
		JOIN asset_types t ON a.asset_type_id = t.id
		LEFT JOIN accounts acc ON acc.asset_id = a.id
		LEFT JOIN LATERAL (
			SELECT
				COALESCE(SUM(jl.debit), 0) AS total_debit,
				COALESCE(SUM(jl.credit), 0) AS total_credit
			FROM journal_lines jl
			JOIN journal_entries je ON je.id = jl.entry_id
			WHERE jl.account_id = acc.id
			  AND je.user_id = a.user_id
		) agg ON true
		WHERE a.id = $1 AND a.user_id = $2
	`

	var a model.Asset
	err := r.db.Pool.QueryRow(ctx, query, id, userID).Scan(
		&a.ID, &a.UserID, &a.AssetTypeID, &a.AssetTypeName, &a.Category, &a.Name,
		&a.Currency, &a.Cost, &a.CurrentValue, &a.IsLiability, &a.CustomFields, &a.CreatedAt, &a.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrAssetNotFound
		}
		return nil, err
	}

	return &a, nil
}

// Create creates a new asset
func (r *AssetRepository) Create(ctx context.Context, a *model.Asset) error {
	if a.CustomFields == nil {
		a.CustomFields = json.RawMessage("{}")
	}

	query := `
		INSERT INTO assets (user_id, asset_type_id, name, currency, cost, current_value, is_liability, custom_fields)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, created_at, updated_at
	`

	return r.db.Pool.QueryRow(ctx, query,
		a.UserID, a.AssetTypeID, a.Name, a.Currency, a.Cost, a.CurrentValue, a.IsLiability, a.CustomFields,
	).Scan(&a.ID, &a.CreatedAt, &a.UpdatedAt)
}

// Update updates an asset
func (r *AssetRepository) Update(ctx context.Context, a *model.Asset) error {
	query := `
		UPDATE assets
		SET asset_type_id = $3, name = $4, currency = $5, cost = $6, current_value = $7, custom_fields = $8, updated_at = NOW()
		WHERE id = $1 AND user_id = $2
		RETURNING updated_at
	`

	err := r.db.Pool.QueryRow(ctx, query,
		a.ID, a.UserID, a.AssetTypeID, a.Name, a.Currency, a.Cost, a.CurrentValue, a.CustomFields,
	).Scan(&a.UpdatedAt)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrAssetNotFound
		}
		return err
	}

	return nil
}

// Delete deletes an asset
func (r *AssetRepository) Delete(ctx context.Context, id, userID uuid.UUID) error {
	query := `DELETE FROM assets WHERE id = $1 AND user_id = $2`

	result, err := r.db.Pool.Exec(ctx, query, id, userID)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23503" {
			return fmt.Errorf("%w: %s", ErrAssetInUse, pgErr.ConstraintName)
		}
		return err
	}

	if result.RowsAffected() == 0 {
		return ErrAssetNotFound
	}

	return nil
}

// ListDeleteBlockers returns linked records that currently block deleting an asset.
func (r *AssetRepository) ListDeleteBlockers(ctx context.Context, assetID, userID uuid.UUID, limit int) ([]AssetDeleteBlocker, error) {
	if limit <= 0 {
		limit = 3
	}

	blockers := make([]AssetDeleteBlocker, 0, limit)

	// 1) Transactions explicitly linked to this asset.
	transactionQuery := `
		SELECT t.id, t.type, c.name, t.amount, t.currency, t.transaction_date, COALESCE(t.description, '')
		FROM transaction_asset_links tal
		JOIN transactions t ON t.id = tal.transaction_id
		JOIN categories c ON c.id = t.category_id
		WHERE tal.asset_id = $1 AND t.user_id = $2
		ORDER BY t.transaction_date DESC
		LIMIT $3
	`
	rows, err := r.db.Pool.Query(ctx, transactionQuery, assetID, userID, limit)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var (
			txID        uuid.UUID
			txType      string
			category    string
			amount      decimal.Decimal
			currency    string
			occurredAt  time.Time
			description string
		)
		if err := rows.Scan(&txID, &txType, &category, &amount, &currency, &occurredAt, &description); err != nil {
			rows.Close()
			return nil, err
		}
		detail := fmt.Sprintf("%s %s %s (%s)", txType, amount.StringFixed(2), currency, category)
		if description != "" {
			detail += ": " + description
		}
		blockers = append(blockers, AssetDeleteBlocker{
			Kind:        "transaction",
			ReferenceID: txID,
			Description: detail,
			OccurredAt:  occurredAt,
		})
		if len(blockers) >= limit {
			rows.Close()
			return blockers, nil
		}
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, err
	}
	rows.Close()

	remaining := limit - len(blockers)
	if remaining <= 0 {
		return blockers, nil
	}

	// 2) Transfers linked to this asset.
	transferQuery := `
		SELECT id, from_asset_id, to_asset_id, from_amount, from_currency, to_amount, to_currency, transfer_date, COALESCE(description, '')
		FROM transfers
		WHERE user_id = $1 AND (from_asset_id = $2 OR to_asset_id = $2)
		ORDER BY transfer_date DESC
		LIMIT $3
	`
	rows, err = r.db.Pool.Query(ctx, transferQuery, userID, assetID, remaining)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var (
			transferID               uuid.UUID
			fromAssetID, toAssetID   uuid.UUID
			fromAmount, toAmount     decimal.Decimal
			fromCurrency, toCurrency string
			occurredAt               time.Time
			description              string
		)
		if err := rows.Scan(
			&transferID,
			&fromAssetID,
			&toAssetID,
			&fromAmount,
			&fromCurrency,
			&toAmount,
			&toCurrency,
			&occurredAt,
			&description,
		); err != nil {
			rows.Close()
			return nil, err
		}
		direction := "incoming"
		if fromAssetID == assetID {
			direction = "outgoing"
		}
		detail := fmt.Sprintf("%s transfer %s %s -> %s %s", direction, fromAmount.StringFixed(2), fromCurrency, toAmount.StringFixed(2), toCurrency)
		if description != "" {
			detail += ": " + description
		}
		blockers = append(blockers, AssetDeleteBlocker{
			Kind:        "transfer",
			ReferenceID: transferID,
			Description: detail,
			OccurredAt:  occurredAt,
		})
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, err
	}
	rows.Close()

	if len(blockers) > limit {
		blockers = blockers[:limit]
	}

	return blockers, nil
}

// RecordSnapshot records a value snapshot for an asset
func (r *AssetRepository) RecordSnapshot(ctx context.Context, snapshot *model.AssetSnapshot) error {
	query := `
		INSERT INTO asset_snapshots (asset_id, value, recorded_at)
		VALUES ($1, $2, $3)
		RETURNING id
	`

	if snapshot.RecordedAt.IsZero() {
		snapshot.RecordedAt = time.Now()
	}

	return r.db.Pool.QueryRow(ctx, query,
		snapshot.AssetID, snapshot.Value, snapshot.RecordedAt,
	).Scan(&snapshot.ID)
}

// GetSnapshots retrieves snapshots for an asset
func (r *AssetRepository) GetSnapshots(ctx context.Context, assetID uuid.UUID, startDate, endDate *time.Time) ([]model.AssetSnapshot, error) {
	query := `
		SELECT id, asset_id, value, recorded_at
		FROM asset_snapshots
		WHERE asset_id = $1
	`
	args := []any{assetID}
	argIndex := 2

	if startDate != nil {
		query += ` AND recorded_at >= $` + string(rune('0'+argIndex))
		args = append(args, *startDate)
		argIndex++
	}
	if endDate != nil {
		query += ` AND recorded_at <= $` + string(rune('0'+argIndex))
		args = append(args, *endDate)
	}

	query += ` ORDER BY recorded_at ASC`

	rows, err := r.db.Pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var snapshots []model.AssetSnapshot
	for rows.Next() {
		var s model.AssetSnapshot
		if err := rows.Scan(&s.ID, &s.AssetID, &s.Value, &s.RecordedAt); err != nil {
			return nil, err
		}
		snapshots = append(snapshots, s)
	}

	return snapshots, rows.Err()
}

// GetTotalAssets calculates total assets value for a user
func (r *AssetRepository) GetTotalAssets(ctx context.Context, userID uuid.UUID) (decimal.Decimal, error) {
	query := `
		SELECT COALESCE(SUM(balance), 0)
		FROM (
			SELECT
				CASE
					WHEN acc.id IS NULL THEN a.current_value
					WHEN acc.account_type IN ('asset', 'expense')
						THEN acc.opening_balance + COALESCE(agg.total_debit, 0) - COALESCE(agg.total_credit, 0)
					ELSE acc.opening_balance + COALESCE(agg.total_credit, 0) - COALESCE(agg.total_debit, 0)
				END AS balance
			FROM assets a
			LEFT JOIN accounts acc ON acc.asset_id = a.id
			LEFT JOIN LATERAL (
				SELECT
					COALESCE(SUM(jl.debit), 0) AS total_debit,
					COALESCE(SUM(jl.credit), 0) AS total_credit
				FROM journal_lines jl
				JOIN journal_entries je ON je.id = jl.entry_id
				WHERE jl.account_id = acc.id
				  AND je.user_id = a.user_id
			) agg ON true
			WHERE a.user_id = $1 AND a.is_liability = false
		) x
	`

	var total decimal.Decimal
	err := r.db.Pool.QueryRow(ctx, query, userID).Scan(&total)
	return total, err
}

// GetTotalLiabilities calculates total liabilities value for a user
func (r *AssetRepository) GetTotalLiabilities(ctx context.Context, userID uuid.UUID) (decimal.Decimal, error) {
	query := `
		SELECT COALESCE(SUM(balance), 0)
		FROM (
			SELECT
				CASE
					WHEN acc.id IS NULL THEN a.current_value
					WHEN acc.account_type IN ('asset', 'expense')
						THEN acc.opening_balance + COALESCE(agg.total_debit, 0) - COALESCE(agg.total_credit, 0)
					ELSE acc.opening_balance + COALESCE(agg.total_credit, 0) - COALESCE(agg.total_debit, 0)
				END AS balance
			FROM assets a
			LEFT JOIN accounts acc ON acc.asset_id = a.id
			LEFT JOIN LATERAL (
				SELECT
					COALESCE(SUM(jl.debit), 0) AS total_debit,
					COALESCE(SUM(jl.credit), 0) AS total_credit
				FROM journal_lines jl
				JOIN journal_entries je ON je.id = jl.entry_id
				WHERE jl.account_id = acc.id
				  AND je.user_id = a.user_id
			) agg ON true
			WHERE a.user_id = $1 AND a.is_liability = true
		) x
	`

	var total decimal.Decimal
	err := r.db.Pool.QueryRow(ctx, query, userID).Scan(&total)
	return total, err
}
