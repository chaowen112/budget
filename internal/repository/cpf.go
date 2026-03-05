package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/shopspring/decimal"

	"github.com/chaowen/budget/internal/model"
)

var (
	ErrCPFAccountNotFound      = errors.New("CPF account not found")
	ErrCPFContributionNotFound = errors.New("CPF contribution not found")
)

type CPFRepository struct {
	db *DB
}

func NewCPFRepository(db *DB) *CPFRepository {
	return &CPFRepository{db: db}
}

// GetAccount retrieves CPF account for a user
func (r *CPFRepository) GetAccount(ctx context.Context, userID uuid.UUID) (*model.CPFAccount, error) {
	query := `
		SELECT id, user_id, oa_balance, sa_balance, ma_balance, updated_at
		FROM cpf_accounts
		WHERE user_id = $1
	`

	var a model.CPFAccount
	err := r.db.Pool.QueryRow(ctx, query, userID).Scan(
		&a.ID, &a.UserID, &a.OABalance, &a.SABalance, &a.MABalance, &a.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrCPFAccountNotFound
		}
		return nil, err
	}

	return &a, nil
}

// CreateOrUpdateAccount creates or updates CPF account
func (r *CPFRepository) CreateOrUpdateAccount(ctx context.Context, a *model.CPFAccount) error {
	query := `
		INSERT INTO cpf_accounts (user_id, oa_balance, sa_balance, ma_balance)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (user_id) DO UPDATE
		SET oa_balance = $2, sa_balance = $3, ma_balance = $4, updated_at = NOW()
		RETURNING id, updated_at
	`

	return r.db.Pool.QueryRow(ctx, query,
		a.UserID, a.OABalance, a.SABalance, a.MABalance,
	).Scan(&a.ID, &a.UpdatedAt)
}

// RecordContribution records a CPF contribution
func (r *CPFRepository) RecordContribution(ctx context.Context, c *model.CPFContribution) error {
	query := `
		INSERT INTO cpf_contributions (user_id, contribution_month, employee_amount, employer_amount, oa_amount, sa_amount, ma_amount)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (user_id, contribution_month) DO UPDATE
		SET employee_amount = $3, employer_amount = $4, oa_amount = $5, sa_amount = $6, ma_amount = $7
		RETURNING id, created_at
	`

	return r.db.Pool.QueryRow(ctx, query,
		c.UserID, c.ContributionMonth, c.EmployeeAmount, c.EmployerAmount,
		c.OAAmount, c.SAAmount, c.MAAmount,
	).Scan(&c.ID, &c.CreatedAt)
}

// ListContributions retrieves CPF contributions for a user
func (r *CPFRepository) ListContributions(ctx context.Context, userID uuid.UUID, year *int) ([]model.CPFContribution, error) {
	query := `
		SELECT id, user_id, contribution_month, employee_amount, employer_amount, oa_amount, sa_amount, ma_amount, created_at
		FROM cpf_contributions
		WHERE user_id = $1
	`
	args := []any{userID}

	if year != nil {
		query += ` AND contribution_month LIKE $2`
		args = append(args, string(rune('0'+*year/1000))+string(rune('0'+(*year%1000)/100))+string(rune('0'+(*year%100)/10))+string(rune('0'+*year%10))+"%")
	}

	query += ` ORDER BY contribution_month DESC`

	rows, err := r.db.Pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var contributions []model.CPFContribution
	for rows.Next() {
		var c model.CPFContribution
		err := rows.Scan(
			&c.ID, &c.UserID, &c.ContributionMonth, &c.EmployeeAmount, &c.EmployerAmount,
			&c.OAAmount, &c.SAAmount, &c.MAAmount, &c.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		contributions = append(contributions, c)
	}

	return contributions, rows.Err()
}

// DeleteContribution deletes a CPF contribution
func (r *CPFRepository) DeleteContribution(ctx context.Context, id, userID uuid.UUID) error {
	query := `DELETE FROM cpf_contributions WHERE id = $1 AND user_id = $2`

	result, err := r.db.Pool.Exec(ctx, query, id, userID)
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return ErrCPFContributionNotFound
	}

	return nil
}

// GetContributionTotals calculates total contributions for a user
func (r *CPFRepository) GetContributionTotals(ctx context.Context, userID uuid.UUID, year *int) (employee, employer, total decimal.Decimal, err error) {
	query := `
		SELECT COALESCE(SUM(employee_amount), 0), COALESCE(SUM(employer_amount), 0)
		FROM cpf_contributions
		WHERE user_id = $1
	`
	args := []any{userID}

	if year != nil {
		query += ` AND contribution_month LIKE $2`
		yearStr := string(rune('0'+*year/1000)) + string(rune('0'+(*year%1000)/100)) + string(rune('0'+(*year%100)/10)) + string(rune('0'+*year%10)) + "%"
		args = append(args, yearStr)
	}

	err = r.db.Pool.QueryRow(ctx, query, args...).Scan(&employee, &employer)
	if err != nil {
		return decimal.Zero, decimal.Zero, decimal.Zero, err
	}

	total = employee.Add(employer)
	return employee, employer, total, nil
}
