package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/chaowen/budget/internal/model"
)

var (
	ErrCategoryNotFound = errors.New("category not found")
)

type CategoryRepository struct {
	db *DB
}

func NewCategoryRepository(db *DB) *CategoryRepository {
	return &CategoryRepository{db: db}
}

// ListAll retrieves all categories (system + user's custom)
func (r *CategoryRepository) ListAll(ctx context.Context, userID uuid.UUID, categoryType *model.CategoryType) ([]model.Category, error) {
	query := `
		SELECT id, user_id, name, type, icon, is_system, created_at
		FROM categories
		WHERE (user_id IS NULL OR user_id = $1)
	`
	args := []any{userID}

	if categoryType != nil {
		query += ` AND type = $2`
		args = append(args, *categoryType)
	}

	query += ` ORDER BY is_system DESC, name ASC`

	rows, err := r.db.Pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var categories []model.Category
	for rows.Next() {
		var c model.Category
		err := rows.Scan(&c.ID, &c.UserID, &c.Name, &c.Type, &c.Icon, &c.IsSystem, &c.CreatedAt)
		if err != nil {
			return nil, err
		}
		categories = append(categories, c)
	}

	return categories, rows.Err()
}

// GetByID retrieves a category by ID
func (r *CategoryRepository) GetByID(ctx context.Context, id uuid.UUID, userID uuid.UUID) (*model.Category, error) {
	query := `
		SELECT id, user_id, name, type, icon, is_system, created_at
		FROM categories
		WHERE id = $1 AND (user_id IS NULL OR user_id = $2)
	`

	var c model.Category
	err := r.db.Pool.QueryRow(ctx, query, id, userID).Scan(
		&c.ID, &c.UserID, &c.Name, &c.Type, &c.Icon, &c.IsSystem, &c.CreatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrCategoryNotFound
		}
		return nil, err
	}

	return &c, nil
}

// Create creates a new custom category
func (r *CategoryRepository) Create(ctx context.Context, category *model.Category) error {
	query := `
		INSERT INTO categories (user_id, name, type, icon, is_system)
		VALUES ($1, $2, $3, $4, false)
		RETURNING id, created_at
	`

	return r.db.Pool.QueryRow(ctx, query,
		category.UserID,
		category.Name,
		category.Type,
		category.Icon,
	).Scan(&category.ID, &category.CreatedAt)
}

// Update updates a custom category
func (r *CategoryRepository) Update(ctx context.Context, category *model.Category) error {
	query := `
		UPDATE categories
		SET name = $3, icon = $4
		WHERE id = $1 AND user_id = $2 AND is_system = false
	`

	result, err := r.db.Pool.Exec(ctx, query,
		category.ID,
		category.UserID,
		category.Name,
		category.Icon,
	)
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return ErrCategoryNotFound
	}

	return nil
}

// Delete deletes a custom category
func (r *CategoryRepository) Delete(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	query := `DELETE FROM categories WHERE id = $1 AND user_id = $2 AND is_system = false`

	result, err := r.db.Pool.Exec(ctx, query, id, userID)
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return ErrCategoryNotFound
	}

	return nil
}
