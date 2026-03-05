package repository

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/shopspring/decimal"

	"github.com/chaowen/budget/internal/model"
)

var (
	ErrAssetNotFound     = errors.New("asset not found")
	ErrAssetTypeNotFound = errors.New("asset type not found")
)

type AssetRepository struct {
	db *DB
}

func NewAssetRepository(db *DB) *AssetRepository {
	return &AssetRepository{db: db}
}

// ListAssetTypes retrieves all asset types
func (r *AssetRepository) ListAssetTypes(ctx context.Context, category *model.AssetCategory) ([]model.AssetType, error) {
	query := `SELECT id, name, category, is_system FROM asset_types`
	args := []any{}

	if category != nil {
		query += ` WHERE category = $1`
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
		SELECT a.id, a.user_id, a.asset_type_id, t.name, t.category, a.name, a.currency,
		       a.current_value, a.is_liability, a.custom_fields, a.created_at, a.updated_at
		FROM assets a
		JOIN asset_types t ON a.asset_type_id = t.id
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

	query += ` ORDER BY t.category, a.name`

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
			&a.Currency, &a.CurrentValue, &a.IsLiability, &a.CustomFields, &a.CreatedAt, &a.UpdatedAt,
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
		SELECT a.id, a.user_id, a.asset_type_id, t.name, t.category, a.name, a.currency,
		       a.current_value, a.is_liability, a.custom_fields, a.created_at, a.updated_at
		FROM assets a
		JOIN asset_types t ON a.asset_type_id = t.id
		WHERE a.id = $1 AND a.user_id = $2
	`

	var a model.Asset
	err := r.db.Pool.QueryRow(ctx, query, id, userID).Scan(
		&a.ID, &a.UserID, &a.AssetTypeID, &a.AssetTypeName, &a.Category, &a.Name,
		&a.Currency, &a.CurrentValue, &a.IsLiability, &a.CustomFields, &a.CreatedAt, &a.UpdatedAt,
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
		INSERT INTO assets (user_id, asset_type_id, name, currency, current_value, is_liability, custom_fields)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, created_at, updated_at
	`

	return r.db.Pool.QueryRow(ctx, query,
		a.UserID, a.AssetTypeID, a.Name, a.Currency, a.CurrentValue, a.IsLiability, a.CustomFields,
	).Scan(&a.ID, &a.CreatedAt, &a.UpdatedAt)
}

// Update updates an asset
func (r *AssetRepository) Update(ctx context.Context, a *model.Asset) error {
	query := `
		UPDATE assets
		SET name = $3, current_value = $4, custom_fields = $5, updated_at = NOW()
		WHERE id = $1 AND user_id = $2
		RETURNING updated_at
	`

	err := r.db.Pool.QueryRow(ctx, query,
		a.ID, a.UserID, a.Name, a.CurrentValue, a.CustomFields,
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
		return err
	}

	if result.RowsAffected() == 0 {
		return ErrAssetNotFound
	}

	return nil
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
		SELECT COALESCE(SUM(current_value), 0)
		FROM assets
		WHERE user_id = $1 AND is_liability = false
	`

	var total decimal.Decimal
	err := r.db.Pool.QueryRow(ctx, query, userID).Scan(&total)
	return total, err
}

// GetTotalLiabilities calculates total liabilities value for a user
func (r *AssetRepository) GetTotalLiabilities(ctx context.Context, userID uuid.UUID) (decimal.Decimal, error) {
	query := `
		SELECT COALESCE(SUM(current_value), 0)
		FROM assets
		WHERE user_id = $1 AND is_liability = true
	`

	var total decimal.Decimal
	err := r.db.Pool.QueryRow(ctx, query, userID).Scan(&total)
	return total, err
}
