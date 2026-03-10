package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"

	"github.com/chaowen/budget/internal/model"
)

var ErrApiKeyNotFound = errors.New("api key not found")

type ApiKeyRepository struct {
	db *DB
}

func NewApiKeyRepository(db *DB) *ApiKeyRepository {
	return &ApiKeyRepository{db: db}
}

func (r *ApiKeyRepository) Create(ctx context.Context, apiKey *model.ApiKey) error {
	query := `
		INSERT INTO api_keys (id, user_id, key_value, name, created_at, last_used_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at
	`
	return r.db.Pool.QueryRow(
		ctx, query,
		apiKey.ID, apiKey.UserID, apiKey.KeyValue, apiKey.Name, apiKey.CreatedAt, apiKey.LastUsedAt,
	).Scan(&apiKey.ID, &apiKey.CreatedAt)
}

func (r *ApiKeyRepository) ListByUserID(ctx context.Context, userID uuid.UUID) ([]*model.ApiKey, error) {
	query := `
		SELECT id, user_id, key_value, name, created_at, last_used_at
		FROM api_keys
		WHERE user_id = $1
		ORDER BY created_at DESC
	`
	rows, err := r.db.Pool.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var keys []*model.ApiKey
	for rows.Next() {
		var key model.ApiKey
		if err := rows.Scan(
			&key.ID, &key.UserID, &key.KeyValue, &key.Name, &key.CreatedAt, &key.LastUsedAt,
		); err != nil {
			return nil, err
		}
		keys = append(keys, &key)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return keys, nil
}

func (r *ApiKeyRepository) Delete(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	query := `DELETE FROM api_keys WHERE id = $1 AND user_id = $2`
	result, err := r.db.Pool.Exec(ctx, query, id, userID)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return ErrApiKeyNotFound
	}
	return nil
}

func (r *ApiKeyRepository) GetByKey(ctx context.Context, keyValue string) (*model.ApiKey, error) {
	query := `
		SELECT id, user_id, key_value, name, created_at, last_used_at
		FROM api_keys
		WHERE key_value = $1
	`
	var key model.ApiKey
	err := r.db.Pool.QueryRow(ctx, query, keyValue).Scan(
		&key.ID, &key.UserID, &key.KeyValue, &key.Name, &key.CreatedAt, &key.LastUsedAt,
	)
	if err != nil {
		return nil, err
	}
	return &key, nil
}

func (r *ApiKeyRepository) UpdateLastUsed(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE api_keys SET last_used_at = CURRENT_TIMESTAMP WHERE id = $1`
	_, err := r.db.Pool.Exec(ctx, query, id)
	return err
}
