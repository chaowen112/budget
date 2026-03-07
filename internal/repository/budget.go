package repository

import (
	"context"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/shopspring/decimal"

	"github.com/chaowen/budget/internal/model"
)

var (
	ErrBudgetNotFound = errors.New("budget not found")
)

type BudgetRepository struct {
	db *DB
}

func NewBudgetRepository(db *DB) *BudgetRepository {
	return &BudgetRepository{db: db}
}

// List retrieves all budgets for a user
func (r *BudgetRepository) List(ctx context.Context, userID uuid.UUID, periodType *model.PeriodType) ([]model.Budget, error) {
	query := `
		SELECT b.id, b.user_id, b.category_id, c.name, b.amount, b.currency, b.period_type, b.start_date, b.created_at, b.updated_at
		FROM budgets b
		JOIN categories c ON b.category_id = c.id
		WHERE b.user_id = $1
	`
	args := []any{userID}

	if periodType != nil {
		query += ` AND b.period_type = $2`
		args = append(args, *periodType)
	}

	query += ` ORDER BY c.name`

	rows, err := r.db.Pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var budgets []model.Budget
	for rows.Next() {
		var b model.Budget
		err := rows.Scan(
			&b.ID, &b.UserID, &b.CategoryID, &b.CategoryName, &b.Amount, &b.Currency,
			&b.PeriodType, &b.StartDate, &b.CreatedAt, &b.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		budgets = append(budgets, b)
	}

	return budgets, rows.Err()
}

// GetByID retrieves a budget by ID
func (r *BudgetRepository) GetByID(ctx context.Context, id, userID uuid.UUID) (*model.Budget, error) {
	query := `
		SELECT b.id, b.user_id, b.category_id, c.name, b.amount, b.currency, b.period_type, b.start_date, b.created_at, b.updated_at
		FROM budgets b
		JOIN categories c ON b.category_id = c.id
		WHERE b.id = $1 AND b.user_id = $2
	`

	var b model.Budget
	err := r.db.Pool.QueryRow(ctx, query, id, userID).Scan(
		&b.ID, &b.UserID, &b.CategoryID, &b.CategoryName, &b.Amount, &b.Currency,
		&b.PeriodType, &b.StartDate, &b.CreatedAt, &b.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrBudgetNotFound
		}
		return nil, err
	}

	return &b, nil
}

// Create creates a new budget
func (r *BudgetRepository) Create(ctx context.Context, b *model.Budget) error {
	query := `
		INSERT INTO budgets (user_id, category_id, amount, currency, period_type, start_date)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at, updated_at
	`

	return r.db.Pool.QueryRow(ctx, query,
		b.UserID, b.CategoryID, b.Amount, b.Currency, b.PeriodType, b.StartDate,
	).Scan(&b.ID, &b.CreatedAt, &b.UpdatedAt)
}

// Update updates a budget
func (r *BudgetRepository) Update(ctx context.Context, b *model.Budget) error {
	query := `
		UPDATE budgets
		SET amount = $3, period_type = $4, start_date = $5, updated_at = NOW()
		WHERE id = $1 AND user_id = $2
		RETURNING updated_at
	`

	err := r.db.Pool.QueryRow(ctx, query,
		b.ID, b.UserID, b.Amount, b.PeriodType, b.StartDate,
	).Scan(&b.UpdatedAt)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrBudgetNotFound
		}
		return err
	}

	return nil
}

// Delete deletes a budget
func (r *BudgetRepository) Delete(ctx context.Context, id, userID uuid.UUID) error {
	query := `DELETE FROM budgets WHERE id = $1 AND user_id = $2`

	result, err := r.db.Pool.Exec(ctx, query, id, userID)
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return ErrBudgetNotFound
	}

	return nil
}

// GetSpentAmount calculates spent amount for a budget in current period
func (r *BudgetRepository) GetSpentAmount(ctx context.Context, userID, categoryID uuid.UUID, periodType model.PeriodType, startDate time.Time) (decimal.Decimal, error) {
	periodStart, periodEnd := GetPeriodBounds(periodType, time.Now(), startDate)

	query := `
		SELECT COALESCE(SUM(amount), 0)
		FROM transactions
		WHERE user_id = $1 AND category_id = $2 AND type = 'expense'
		  AND transaction_date >= $3 AND transaction_date <= $4
	`

	var spent decimal.Decimal
	err := r.db.Pool.QueryRow(ctx, query, userID, categoryID, periodStart, periodEnd).Scan(&spent)
	return spent, err
}

// GetPeriodBounds calculates the start and end of a budget period
func GetPeriodBounds(periodType model.PeriodType, now time.Time, startDate time.Time) (time.Time, time.Time) {
	startDate = startDate.In(now.Location())

	switch periodType {
	case model.PeriodTypeDaily:
		start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		end := start.Add(24*time.Hour - time.Second)
		return start, end

	case model.PeriodTypeWeekly:
		if now.Before(startDate) {
			cycleStart := time.Date(startDate.Year(), startDate.Month(), startDate.Day(), 0, 0, 0, 0, now.Location())
			cycleEnd := cycleStart.AddDate(0, 0, 7).Add(-time.Second)
			return cycleStart, cycleEnd
		}

		startAnchor := time.Date(startDate.Year(), startDate.Month(), startDate.Day(), 0, 0, 0, 0, now.Location())
		daysSinceStart := int(math.Floor(now.Sub(startAnchor).Hours() / 24))
		cycles := daysSinceStart / 7
		cycleStart := startAnchor.AddDate(0, 0, cycles*7)
		cycleEnd := cycleStart.AddDate(0, 0, 7).Add(-time.Second)
		return cycleStart, cycleEnd

	case model.PeriodTypeMonthly:
		cycleStart := time.Date(startDate.Year(), startDate.Month(), startDate.Day(), 0, 0, 0, 0, now.Location())
		if now.Before(cycleStart) {
			return cycleStart, addMonths(cycleStart, 1).Add(-time.Second)
		}

		for !addMonths(cycleStart, 1).After(now) {
			cycleStart = addMonths(cycleStart, 1)
		}
		cycleEnd := addMonths(cycleStart, 1).Add(-time.Second)
		return cycleStart, cycleEnd

	case model.PeriodTypeYearly:
		cycleStart := time.Date(startDate.Year(), startDate.Month(), startDate.Day(), 0, 0, 0, 0, now.Location())
		if now.Before(cycleStart) {
			return cycleStart, addYears(cycleStart, 1).Add(-time.Second)
		}

		for !addYears(cycleStart, 1).After(now) {
			cycleStart = addYears(cycleStart, 1)
		}
		cycleEnd := addYears(cycleStart, 1).Add(-time.Second)
		return cycleStart, cycleEnd

	default:
		return now, now
	}
}

func addMonths(d time.Time, months int) time.Time {
	base := time.Date(d.Year(), d.Month(), 1, d.Hour(), d.Minute(), d.Second(), d.Nanosecond(), d.Location())
	targetMonthStart := base.AddDate(0, months, 0)
	lastDay := time.Date(targetMonthStart.Year(), targetMonthStart.Month()+1, 0, 0, 0, 0, 0, d.Location()).Day()
	day := d.Day()
	if day > lastDay {
		day = lastDay
	}
	return time.Date(targetMonthStart.Year(), targetMonthStart.Month(), day, d.Hour(), d.Minute(), d.Second(), d.Nanosecond(), d.Location())
}

func addYears(d time.Time, years int) time.Time {
	base := time.Date(d.Year()+years, d.Month(), 1, d.Hour(), d.Minute(), d.Second(), d.Nanosecond(), d.Location())
	lastDay := time.Date(base.Year(), base.Month()+1, 0, 0, 0, 0, 0, d.Location()).Day()
	day := d.Day()
	if day > lastDay {
		day = lastDay
	}
	return time.Date(base.Year(), base.Month(), day, d.Hour(), d.Minute(), d.Second(), d.Nanosecond(), d.Location())
}

// GetBudgetStatus calculates the status of a budget
func (r *BudgetRepository) GetBudgetStatus(ctx context.Context, budget *model.Budget) (*model.BudgetStatus, error) {
	spent, err := r.GetSpentAmount(ctx, budget.UserID, budget.CategoryID, budget.PeriodType, budget.StartDate)
	if err != nil {
		return nil, fmt.Errorf("failed to get spent amount: %w", err)
	}

	remaining := budget.Amount.Sub(spent)
	percentUsed := float64(0)
	if !budget.Amount.IsZero() {
		percentUsed = spent.Div(budget.Amount).InexactFloat64() * 100
	}

	return &model.BudgetStatus{
		Budget:       *budget,
		Spent:        spent,
		Remaining:    remaining,
		PercentUsed:  percentUsed,
		IsOverBudget: remaining.IsNegative(),
	}, nil
}
