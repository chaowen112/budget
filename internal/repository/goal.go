package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/chaowen/budget/internal/model"
)

var (
	ErrGoalNotFound         = errors.New("saving goal not found")
	ErrNetWorthGoalNotFound = errors.New("net worth goal not found")
)

type GoalRepository struct {
	db *DB
}

func NewGoalRepository(db *DB) *GoalRepository {
	return &GoalRepository{db: db}
}

// List retrieves all saving goals for a user
func (r *GoalRepository) List(ctx context.Context, userID uuid.UUID, includeCompleted bool) ([]model.SavingGoal, error) {
	query := `
		SELECT id, user_id, name, target_amount, current_amount, currency, deadline, linked_asset_ids, notes, created_at, updated_at
		FROM saving_goals
		WHERE user_id = $1
	`

	if !includeCompleted {
		query += ` AND current_amount < target_amount`
	}

	query += ` ORDER BY deadline NULLS LAST, created_at DESC`

	rows, err := r.db.Pool.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var goals []model.SavingGoal
	for rows.Next() {
		var g model.SavingGoal
		err := rows.Scan(
			&g.ID, &g.UserID, &g.Name, &g.TargetAmount, &g.CurrentAmount, &g.Currency,
			&g.Deadline, &g.LinkedAssetIDs, &g.Notes, &g.CreatedAt, &g.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		goals = append(goals, g)
	}

	return goals, rows.Err()
}

// GetByID retrieves a saving goal by ID
func (r *GoalRepository) GetByID(ctx context.Context, id, userID uuid.UUID) (*model.SavingGoal, error) {
	query := `
		SELECT id, user_id, name, target_amount, current_amount, currency, deadline, linked_asset_ids, notes, created_at, updated_at
		FROM saving_goals
		WHERE id = $1 AND user_id = $2
	`

	var g model.SavingGoal
	err := r.db.Pool.QueryRow(ctx, query, id, userID).Scan(
		&g.ID, &g.UserID, &g.Name, &g.TargetAmount, &g.CurrentAmount, &g.Currency,
		&g.Deadline, &g.LinkedAssetIDs, &g.Notes, &g.CreatedAt, &g.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrGoalNotFound
		}
		return nil, err
	}

	return &g, nil
}

// Create creates a new saving goal
func (r *GoalRepository) Create(ctx context.Context, g *model.SavingGoal) error {
	query := `
		INSERT INTO saving_goals (user_id, name, target_amount, current_amount, currency, deadline, linked_asset_ids, notes)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, created_at, updated_at
	`

	return r.db.Pool.QueryRow(ctx, query,
		g.UserID, g.Name, g.TargetAmount, g.CurrentAmount, g.Currency, g.Deadline, g.LinkedAssetIDs, g.Notes,
	).Scan(&g.ID, &g.CreatedAt, &g.UpdatedAt)
}

// Update updates a saving goal
func (r *GoalRepository) Update(ctx context.Context, g *model.SavingGoal) error {
	query := `
		UPDATE saving_goals
		SET name = $3, target_amount = $4, current_amount = $5, deadline = $6, linked_asset_ids = $7, notes = $8, updated_at = NOW()
		WHERE id = $1 AND user_id = $2
		RETURNING updated_at
	`

	err := r.db.Pool.QueryRow(ctx, query,
		g.ID, g.UserID, g.Name, g.TargetAmount, g.CurrentAmount, g.Deadline, g.LinkedAssetIDs, g.Notes,
	).Scan(&g.UpdatedAt)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrGoalNotFound
		}
		return err
	}

	return nil
}

// UpdateProgress updates only the current amount of a goal
func (r *GoalRepository) UpdateProgress(ctx context.Context, g *model.SavingGoal) error {
	query := `
		UPDATE saving_goals
		SET current_amount = $3, updated_at = NOW()
		WHERE id = $1 AND user_id = $2
		RETURNING updated_at
	`

	err := r.db.Pool.QueryRow(ctx, query, g.ID, g.UserID, g.CurrentAmount).Scan(&g.UpdatedAt)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrGoalNotFound
		}
		return err
	}

	return nil
}

// Delete deletes a saving goal
func (r *GoalRepository) Delete(ctx context.Context, id, userID uuid.UUID) error {
	query := `DELETE FROM saving_goals WHERE id = $1 AND user_id = $2`

	result, err := r.db.Pool.Exec(ctx, query, id, userID)
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return ErrGoalNotFound
	}

	return nil
}

// ========== NetWorthGoal Methods ==========

// GetNetWorthGoal retrieves the user's net worth goal (each user has at most one)
func (r *GoalRepository) GetNetWorthGoal(ctx context.Context, userID uuid.UUID) (*model.NetWorthGoal, error) {
	query := `
		SELECT id, user_id, name, target_amount, currency, notes, created_at, updated_at
		FROM net_worth_goals
		WHERE user_id = $1
	`

	var g model.NetWorthGoal
	err := r.db.Pool.QueryRow(ctx, query, userID).Scan(
		&g.ID, &g.UserID, &g.Name, &g.TargetAmount, &g.Currency,
		&g.Notes, &g.CreatedAt, &g.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNetWorthGoalNotFound
		}
		return nil, err
	}

	return &g, nil
}

// SetNetWorthGoal creates or updates the user's net worth goal (upsert)
func (r *GoalRepository) SetNetWorthGoal(ctx context.Context, g *model.NetWorthGoal) error {
	query := `
		INSERT INTO net_worth_goals (user_id, name, target_amount, currency, notes)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (user_id) DO UPDATE SET
			name = EXCLUDED.name,
			target_amount = EXCLUDED.target_amount,
			currency = EXCLUDED.currency,
			notes = EXCLUDED.notes,
			updated_at = NOW()
		RETURNING id, created_at, updated_at
	`

	return r.db.Pool.QueryRow(ctx, query,
		g.UserID, g.Name, g.TargetAmount, g.Currency, g.Notes,
	).Scan(&g.ID, &g.CreatedAt, &g.UpdatedAt)
}

// DeleteNetWorthGoal deletes the user's net worth goal
func (r *GoalRepository) DeleteNetWorthGoal(ctx context.Context, userID uuid.UUID) error {
	query := `DELETE FROM net_worth_goals WHERE user_id = $1`

	result, err := r.db.Pool.Exec(ctx, query, userID)
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return ErrNetWorthGoalNotFound
	}

	return nil
}
