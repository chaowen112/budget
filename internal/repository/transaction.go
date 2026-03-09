package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/shopspring/decimal"

	"github.com/chaowen/budget/internal/model"
)

var (
	ErrTransactionNotFound          = errors.New("transaction not found")
	ErrTransactionAssetLinkNotFound = errors.New("transaction asset link not found")
)

type TransactionRepository struct {
	db *DB
}

type TransactionSourceLink struct {
	TransactionID uuid.UUID
	AssetID       uuid.UUID
	AssetName     string
}

func NewTransactionRepository(db *DB) *TransactionRepository {
	return &TransactionRepository{db: db}
}

// TransactionFilter for querying transactions
type TransactionFilter struct {
	UserID     uuid.UUID
	CategoryID *uuid.UUID
	Type       *model.CategoryType
	StartDate  *time.Time
	EndDate    *time.Time
	Search     string
	Tags       []string
	Currency   string
	Page       int
	PageSize   int
}

// ListResult contains paginated results
type ListResult struct {
	Transactions []model.Transaction
	TotalCount   int
}

// List retrieves transactions with filters
func (r *TransactionRepository) List(ctx context.Context, filter TransactionFilter) (*ListResult, error) {
	// Count query
	countQuery := `
		SELECT COUNT(*)
		FROM transactions t
		JOIN categories c ON t.category_id = c.id
		LEFT JOIN transaction_asset_links tal ON tal.transaction_id = t.id
		LEFT JOIN assets sa ON sa.id = tal.asset_id
		WHERE t.user_id = $1
	`
	args := []any{filter.UserID}
	argIndex := 2

	if filter.CategoryID != nil {
		countQuery += ` AND t.category_id = ` + placeholder(argIndex)
		args = append(args, *filter.CategoryID)
		argIndex++
	}
	if filter.Type != nil {
		countQuery += ` AND t.type = ` + placeholder(argIndex)
		args = append(args, *filter.Type)
		argIndex++
	}
	if filter.StartDate != nil {
		countQuery += ` AND t.transaction_date >= ` + placeholder(argIndex)
		args = append(args, *filter.StartDate)
		argIndex++
	}
	if filter.EndDate != nil {
		countQuery += ` AND t.transaction_date <= ` + placeholder(argIndex)
		args = append(args, *filter.EndDate)
		argIndex++
	}
	if len(filter.Tags) > 0 {
		countQuery += ` AND t.tags @> ` + placeholder(argIndex)
		args = append(args, filter.Tags)
		argIndex++
	}
	if filter.Currency != "" {
		countQuery += ` AND t.currency = ` + placeholder(argIndex)
		args = append(args, filter.Currency)
		argIndex++
	}
	if filter.Search != "" {
		countQuery += ` AND (
			COALESCE(t.description, '') ILIKE ` + placeholder(argIndex) + `
			OR c.name ILIKE ` + placeholder(argIndex) + `
			OR COALESCE(sa.name, '') ILIKE ` + placeholder(argIndex) + `
			OR EXISTS (
				SELECT 1
				FROM unnest(t.tags) AS tag
				WHERE tag ILIKE ` + placeholder(argIndex) + `
			)
		)`
		args = append(args, "%"+filter.Search+"%")
		argIndex++
	}

	var totalCount int
	err := r.db.Pool.QueryRow(ctx, countQuery, args...).Scan(&totalCount)
	if err != nil {
		return nil, err
	}

	// Main query
	query := `
		SELECT t.id, t.user_id, t.category_id, c.name, t.amount, t.currency, t.type,
		       t.transaction_date, t.description, t.tags, t.created_at, t.updated_at
		FROM transactions t
		JOIN categories c ON t.category_id = c.id
		LEFT JOIN transaction_asset_links tal ON tal.transaction_id = t.id
		LEFT JOIN assets sa ON sa.id = tal.asset_id
		WHERE t.user_id = $1
	`

	args = []any{filter.UserID}
	argIndex = 2

	if filter.CategoryID != nil {
		query += ` AND t.category_id = ` + placeholder(argIndex)
		args = append(args, *filter.CategoryID)
		argIndex++
	}
	if filter.Type != nil {
		query += ` AND t.type = ` + placeholder(argIndex)
		args = append(args, *filter.Type)
		argIndex++
	}
	if filter.StartDate != nil {
		query += ` AND t.transaction_date >= ` + placeholder(argIndex)
		args = append(args, *filter.StartDate)
		argIndex++
	}
	if filter.EndDate != nil {
		query += ` AND t.transaction_date <= ` + placeholder(argIndex)
		args = append(args, *filter.EndDate)
		argIndex++
	}
	if len(filter.Tags) > 0 {
		query += ` AND t.tags @> ` + placeholder(argIndex)
		args = append(args, filter.Tags)
		argIndex++
	}
	if filter.Currency != "" {
		query += ` AND t.currency = ` + placeholder(argIndex)
		args = append(args, filter.Currency)
		argIndex++
	}
	if filter.Search != "" {
		query += ` AND (
			COALESCE(t.description, '') ILIKE ` + placeholder(argIndex) + `
			OR c.name ILIKE ` + placeholder(argIndex) + `
			OR COALESCE(sa.name, '') ILIKE ` + placeholder(argIndex) + `
			OR EXISTS (
				SELECT 1
				FROM unnest(t.tags) AS tag
				WHERE tag ILIKE ` + placeholder(argIndex) + `
			)
		)`
		args = append(args, "%"+filter.Search+"%")
		argIndex++
	}

	query += ` ORDER BY t.transaction_date DESC, t.created_at DESC`

	// Pagination
	if filter.PageSize > 0 {
		offset := (filter.Page - 1) * filter.PageSize
		if offset < 0 {
			offset = 0
		}
		query += ` LIMIT ` + placeholder(argIndex) + ` OFFSET ` + placeholder(argIndex+1)
		args = append(args, filter.PageSize, offset)
	}

	rows, err := r.db.Pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var transactions []model.Transaction
	for rows.Next() {
		var t model.Transaction
		err := rows.Scan(
			&t.ID, &t.UserID, &t.CategoryID, &t.CategoryName, &t.Amount, &t.Currency,
			&t.Type, &t.TransactionDate, &t.Description, &t.Tags, &t.CreatedAt, &t.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		transactions = append(transactions, t)
	}

	return &ListResult{
		Transactions: transactions,
		TotalCount:   totalCount,
	}, rows.Err()
}

func placeholder(index int) string {
	return fmt.Sprintf("$%d", index)
}

// GetByID retrieves a transaction by ID
func (r *TransactionRepository) GetByID(ctx context.Context, id, userID uuid.UUID) (*model.Transaction, error) {
	query := `
		SELECT t.id, t.user_id, t.category_id, c.name, t.amount, t.currency, t.type,
		       t.transaction_date, t.description, t.tags, t.created_at, t.updated_at
		FROM transactions t
		JOIN categories c ON t.category_id = c.id
		WHERE t.id = $1 AND t.user_id = $2
	`

	var t model.Transaction
	err := r.db.Pool.QueryRow(ctx, query, id, userID).Scan(
		&t.ID, &t.UserID, &t.CategoryID, &t.CategoryName, &t.Amount, &t.Currency,
		&t.Type, &t.TransactionDate, &t.Description, &t.Tags, &t.CreatedAt, &t.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrTransactionNotFound
		}
		return nil, err
	}

	return &t, nil
}

// Create creates a new transaction
func (r *TransactionRepository) Create(ctx context.Context, t *model.Transaction) error {
	query := `
		INSERT INTO transactions (user_id, category_id, amount, currency, type, transaction_date, description, tags)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, created_at, updated_at
	`

	return r.db.Pool.QueryRow(ctx, query,
		t.UserID, t.CategoryID, t.Amount, t.Currency, t.Type,
		t.TransactionDate, t.Description, t.Tags,
	).Scan(&t.ID, &t.CreatedAt, &t.UpdatedAt)
}

// Update updates a transaction
func (r *TransactionRepository) Update(ctx context.Context, t *model.Transaction) error {
	query := `
		UPDATE transactions
		SET category_id = $3, amount = $4, transaction_date = $5, description = $6, tags = $7, updated_at = NOW()
		WHERE id = $1 AND user_id = $2
		RETURNING updated_at
	`

	err := r.db.Pool.QueryRow(ctx, query,
		t.ID, t.UserID, t.CategoryID, t.Amount, t.TransactionDate, t.Description, t.Tags,
	).Scan(&t.UpdatedAt)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrTransactionNotFound
		}
		return err
	}

	return nil
}

// Delete deletes a transaction
func (r *TransactionRepository) Delete(ctx context.Context, id, userID uuid.UUID) error {
	query := `DELETE FROM transactions WHERE id = $1 AND user_id = $2`

	result, err := r.db.Pool.Exec(ctx, query, id, userID)
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return ErrTransactionNotFound
	}

	return nil
}

// SetSourceAssetLink creates or updates the source asset linked to a transaction.
func (r *TransactionRepository) SetSourceAssetLink(ctx context.Context, transactionID, assetID uuid.UUID) error {
	query := `
		INSERT INTO transaction_asset_links (transaction_id, asset_id, updated_at)
		VALUES ($1, $2, NOW())
		ON CONFLICT (transaction_id)
		DO UPDATE SET asset_id = EXCLUDED.asset_id, updated_at = NOW()
	`

	_, err := r.db.Pool.Exec(ctx, query, transactionID, assetID)
	return err
}

// GetSourceAssetLink returns the linked source asset for a transaction.
func (r *TransactionRepository) GetSourceAssetLink(ctx context.Context, transactionID uuid.UUID) (uuid.UUID, error) {
	query := `
		SELECT asset_id
		FROM transaction_asset_links
		WHERE transaction_id = $1
	`

	var assetID uuid.UUID
	err := r.db.Pool.QueryRow(ctx, query, transactionID).Scan(&assetID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return uuid.Nil, ErrTransactionAssetLinkNotFound
		}
		return uuid.Nil, err
	}

	return assetID, nil
}

// ListSourceAssetLinks returns source asset links for all transactions of a user.
func (r *TransactionRepository) ListSourceAssetLinks(ctx context.Context, userID uuid.UUID) ([]TransactionSourceLink, error) {
	query := `
		SELECT tal.transaction_id, tal.asset_id, a.name
		FROM transaction_asset_links tal
		JOIN transactions t ON t.id = tal.transaction_id
		JOIN assets a ON a.id = tal.asset_id
		WHERE t.user_id = $1
	`

	rows, err := r.db.Pool.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	links := make([]TransactionSourceLink, 0)
	for rows.Next() {
		var item TransactionSourceLink
		if err := rows.Scan(&item.TransactionID, &item.AssetID, &item.AssetName); err != nil {
			return nil, err
		}
		links = append(links, item)
	}

	return links, rows.Err()
}

// GetSpendingByCategory gets spending grouped by category and currency for a date range
func (r *TransactionRepository) GetSpendingByCategory(ctx context.Context, userID uuid.UUID, startDate, endDate time.Time, transactionType model.CategoryType) ([]CategorySpending, error) {
	query := `
		SELECT c.id, c.name, t.currency, SUM(t.amount) as total, COUNT(*) as count
		FROM transactions t
		JOIN categories c ON t.category_id = c.id
		WHERE t.user_id = $1 AND t.type = $2
		  AND t.transaction_date >= $3 AND t.transaction_date <= $4
		GROUP BY c.id, c.name, t.currency
		ORDER BY total DESC
	`

	rows, err := r.db.Pool.Query(ctx, query, userID, transactionType, startDate, endDate)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []CategorySpending
	for rows.Next() {
		var cs CategorySpending
		if err := rows.Scan(&cs.CategoryID, &cs.CategoryName, &cs.Currency, &cs.Total, &cs.Count); err != nil {
			return nil, err
		}
		results = append(results, cs)
	}

	return results, rows.Err()
}

// CategorySpending represents spending for a category in a specific currency
type CategorySpending struct {
	CategoryID   uuid.UUID
	CategoryName string
	Currency     string
	Total        decimal.Decimal
	Count        int
}

// CurrencyAmount represents an amount in a specific currency
type CurrencyAmount struct {
	Currency string
	Amount   decimal.Decimal
}

// GetTotalByType gets total amounts grouped by currency for a transaction type and date range
func (r *TransactionRepository) GetTotalByType(ctx context.Context, userID uuid.UUID, startDate, endDate time.Time, transactionType model.CategoryType) ([]CurrencyAmount, error) {
	query := `
		SELECT currency, COALESCE(SUM(amount), 0)
		FROM transactions
		WHERE user_id = $1 AND type = $2
		  AND transaction_date >= $3 AND transaction_date <= $4
		GROUP BY currency
	`

	rows, err := r.db.Pool.Query(ctx, query, userID, transactionType, startDate, endDate)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []CurrencyAmount
	for rows.Next() {
		var ca CurrencyAmount
		if err := rows.Scan(&ca.Currency, &ca.Amount); err != nil {
			return nil, err
		}
		results = append(results, ca)
	}

	return results, rows.Err()
}
