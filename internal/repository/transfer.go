package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/chaowen/budget/internal/model"
)

var ErrTransferNotFound = errors.New("transfer not found")

type TransferRepository struct {
	db *DB
}

func NewTransferRepository(db *DB) *TransferRepository {
	return &TransferRepository{db: db}
}

func (r *TransferRepository) Create(ctx context.Context, t *model.Transfer) error {
	query := `
		INSERT INTO transfers (user_id, from_asset_id, to_asset_id, from_amount, to_amount, from_currency, to_currency, exchange_rate, transfer_date, description)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
		RETURNING id, created_at, updated_at
	`
	return r.db.Pool.QueryRow(ctx, query,
		t.UserID, t.FromAssetID, t.ToAssetID, t.FromAmount, t.ToAmount, t.FromCurrency, t.ToCurrency, t.ExchangeRate, t.TransferDate, t.Description,
	).Scan(&t.ID, &t.CreatedAt, &t.UpdatedAt)
}

func (r *TransferRepository) GetByID(ctx context.Context, id, userID uuid.UUID) (*model.Transfer, error) {
	query := `
		SELECT t.id, t.user_id, t.from_asset_id, t.to_asset_id, t.from_amount, t.to_amount,
		       t.from_currency, t.to_currency, t.exchange_rate, t.transfer_date, t.description,
		       t.created_at, t.updated_at, af.name, at.name
		FROM transfers t
		JOIN assets af ON af.id = t.from_asset_id
		JOIN assets at ON at.id = t.to_asset_id
		WHERE t.id = $1 AND t.user_id = $2
	`
	var t model.Transfer
	err := r.db.Pool.QueryRow(ctx, query, id, userID).Scan(
		&t.ID, &t.UserID, &t.FromAssetID, &t.ToAssetID, &t.FromAmount, &t.ToAmount,
		&t.FromCurrency, &t.ToCurrency, &t.ExchangeRate, &t.TransferDate, &t.Description,
		&t.CreatedAt, &t.UpdatedAt, &t.FromAssetName, &t.ToAssetName,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrTransferNotFound
		}
		return nil, err
	}
	return &t, nil
}

func (r *TransferRepository) List(ctx context.Context, userID uuid.UUID, limit int) ([]model.Transfer, error) {
	if limit <= 0 {
		limit = 100
	}
	query := `
		SELECT t.id, t.user_id, t.from_asset_id, t.to_asset_id, t.from_amount, t.to_amount,
		       t.from_currency, t.to_currency, t.exchange_rate, t.transfer_date, t.description,
		       t.created_at, t.updated_at, af.name, at.name
		FROM transfers t
		JOIN assets af ON af.id = t.from_asset_id
		JOIN assets at ON at.id = t.to_asset_id
		WHERE t.user_id = $1
		ORDER BY t.transfer_date DESC, t.created_at DESC
		LIMIT $2
	`
	rows, err := r.db.Pool.Query(ctx, query, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []model.Transfer
	for rows.Next() {
		var t model.Transfer
		if err := rows.Scan(
			&t.ID, &t.UserID, &t.FromAssetID, &t.ToAssetID, &t.FromAmount, &t.ToAmount,
			&t.FromCurrency, &t.ToCurrency, &t.ExchangeRate, &t.TransferDate, &t.Description,
			&t.CreatedAt, &t.UpdatedAt, &t.FromAssetName, &t.ToAssetName,
		); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

func (r *TransferRepository) Update(ctx context.Context, t *model.Transfer) error {
	query := `
		UPDATE transfers
		SET from_asset_id = $3, to_asset_id = $4, from_amount = $5, to_amount = $6,
		    from_currency = $7, to_currency = $8, exchange_rate = $9, transfer_date = $10,
		    description = $11, updated_at = NOW()
		WHERE id = $1 AND user_id = $2
		RETURNING updated_at
	`
	err := r.db.Pool.QueryRow(ctx, query,
		t.ID, t.UserID, t.FromAssetID, t.ToAssetID, t.FromAmount, t.ToAmount,
		t.FromCurrency, t.ToCurrency, t.ExchangeRate, t.TransferDate, t.Description,
	).Scan(&t.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrTransferNotFound
		}
		return err
	}
	return nil
}

func (r *TransferRepository) Delete(ctx context.Context, id, userID uuid.UUID) error {
	res, err := r.db.Pool.Exec(ctx, `DELETE FROM transfers WHERE id = $1 AND user_id = $2`, id, userID)
	if err != nil {
		return err
	}
	if res.RowsAffected() == 0 {
		return ErrTransferNotFound
	}
	return nil
}
