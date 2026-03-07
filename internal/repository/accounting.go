package repository

import (
	"context"
	"errors"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/shopspring/decimal"

	"github.com/chaowen/budget/internal/model"
)

type AccountingRepository struct {
	db *DB
}

func NewAccountingRepository(db *DB) *AccountingRepository {
	return &AccountingRepository{db: db}
}

func (r *AccountingRepository) EnsureAssetAccount(ctx context.Context, asset *model.Asset) (uuid.UUID, error) {
	query := `SELECT id FROM accounts WHERE asset_id = $1`
	var accountID uuid.UUID
	err := r.db.Pool.QueryRow(ctx, query, asset.ID).Scan(&accountID)
	if err == nil {
		return accountID, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, err
	}

	accountType := "asset"
	if asset.IsLiability {
		accountType = "liability"
	}

	insert := `
		INSERT INTO accounts (user_id, name, account_type, currency, opening_balance, asset_id, is_system)
		VALUES ($1, $2, $3::account_type, $4, $5, $6, true)
		RETURNING id
	`
	err = r.db.Pool.QueryRow(ctx, insert, asset.UserID, asset.Name, accountType, asset.Currency, asset.CurrentValue, asset.ID).Scan(&accountID)
	return accountID, err
}

func (r *AccountingRepository) EnsureCategoryAccount(ctx context.Context, userID, categoryID uuid.UUID, currency string, categoryType model.CategoryType, categoryName string) (uuid.UUID, error) {
	query := `SELECT id FROM accounts WHERE category_id = $1`
	var accountID uuid.UUID
	err := r.db.Pool.QueryRow(ctx, query, categoryID).Scan(&accountID)
	if err == nil {
		return accountID, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, err
	}

	accountType := "expense"
	if categoryType == model.CategoryTypeIncome {
		accountType = "income"
	}

	insert := `
		INSERT INTO accounts (user_id, name, account_type, currency, opening_balance, category_id, is_system)
		VALUES ($1, $2, $3::account_type, $4, 0, $5, true)
		RETURNING id
	`
	err = r.db.Pool.QueryRow(ctx, insert, userID, categoryName, accountType, currency, categoryID).Scan(&accountID)
	return accountID, err
}

func (r *AccountingRepository) EnsureBalanceAdjustmentAccount(ctx context.Context, userID uuid.UUID, currency string) (uuid.UUID, error) {
	query := `SELECT id FROM accounts WHERE user_id = $1 AND account_type = 'equity' AND is_system = true ORDER BY created_at ASC LIMIT 1`
	var accountID uuid.UUID
	err := r.db.Pool.QueryRow(ctx, query, userID).Scan(&accountID)
	if err == nil {
		return accountID, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, err
	}

	insert := `
		INSERT INTO accounts (user_id, name, account_type, currency, opening_balance, is_system)
		VALUES ($1, 'Balance Adjustments', 'equity', $2, 0, true)
		RETURNING id
	`
	err = r.db.Pool.QueryRow(ctx, insert, userID, currency).Scan(&accountID)
	return accountID, err
}

func (r *AccountingRepository) UpsertTransactionEntry(
	ctx context.Context,
	userID uuid.UUID,
	transaction *model.Transaction,
	assetAccountID uuid.UUID,
	categoryAccountID uuid.UUID,
) error {
	baseCurrency, err := r.getUserBaseCurrency(ctx, userID)
	if err != nil {
		return err
	}
	baseAmount, err := r.convertToBase(ctx, transaction.Amount, transaction.Currency, baseCurrency)
	if err != nil {
		return err
	}

	tx, err := r.db.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var entryID uuid.UUID
	find := `SELECT id FROM journal_entries WHERE user_id = $1 AND reference_type = 'transaction' AND reference_id = $2`
	err = tx.QueryRow(ctx, find, userID, transaction.ID).Scan(&entryID)
	if errors.Is(err, pgx.ErrNoRows) {
		insert := `
			INSERT INTO journal_entries (user_id, entry_date, description, source, reference_type, reference_id, base_currency)
			VALUES ($1, $2, $3, 'transaction', 'transaction', $4, $5)
			RETURNING id
		`
		err = tx.QueryRow(ctx, insert, userID, transaction.TransactionDate, transaction.Description, transaction.ID, baseCurrency).Scan(&entryID)
		if err != nil {
			return err
		}
	} else if err != nil {
		return err
	} else {
		if _, err = tx.Exec(ctx, `DELETE FROM journal_lines WHERE entry_id = $1`, entryID); err != nil {
			return err
		}
		if _, err = tx.Exec(ctx, `UPDATE journal_entries SET entry_date = $2, description = $3, base_currency = $4 WHERE id = $1`, entryID, transaction.TransactionDate, transaction.Description, baseCurrency); err != nil {
			return err
		}
	}

	var debitAccountID, creditAccountID uuid.UUID
	if transaction.Type == model.CategoryTypeIncome {
		debitAccountID = assetAccountID
		creditAccountID = categoryAccountID
	} else {
		debitAccountID = categoryAccountID
		creditAccountID = assetAccountID
	}

	if _, err = tx.Exec(ctx, `
		INSERT INTO journal_lines (entry_id, account_id, debit, credit, base_debit, base_credit, description)
		VALUES ($1, $2, $3, 0, $4, 0, 'transaction debit')
	`, entryID, debitAccountID, transaction.Amount, baseAmount); err != nil {
		return err
	}

	if _, err = tx.Exec(ctx, `
		INSERT INTO journal_lines (entry_id, account_id, debit, credit, base_debit, base_credit, description)
		VALUES ($1, $2, 0, $3, 0, $4, 'transaction credit')
	`, entryID, creditAccountID, transaction.Amount, baseAmount); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func (r *AccountingRepository) UpsertTransferEntry(
	ctx context.Context,
	userID uuid.UUID,
	transfer *model.Transfer,
	fromAssetAccountID uuid.UUID,
	toAssetAccountID uuid.UUID,
) error {
	baseCurrency, err := r.getUserBaseCurrency(ctx, userID)
	if err != nil {
		return err
	}

	baseAmount, err := r.convertToBase(ctx, transfer.FromAmount, transfer.FromCurrency, baseCurrency)
	if err != nil {
		if transfer.ToCurrency == baseCurrency {
			baseAmount = transfer.ToAmount
		} else {
			return err
		}
	}

	tx, err := r.db.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var entryID uuid.UUID
	find := `SELECT id FROM journal_entries WHERE user_id = $1 AND reference_type = 'transfer' AND reference_id = $2`
	err = tx.QueryRow(ctx, find, userID, transfer.ID).Scan(&entryID)
	if errors.Is(err, pgx.ErrNoRows) {
		err = tx.QueryRow(ctx, `
			INSERT INTO journal_entries (user_id, entry_date, description, source, reference_type, reference_id, base_currency)
			VALUES ($1, $2, $3, 'transfer', 'transfer', $4, $5)
			RETURNING id
		`, userID, transfer.TransferDate, transfer.Description, transfer.ID, baseCurrency).Scan(&entryID)
		if err != nil {
			return err
		}
	} else if err != nil {
		return err
	} else {
		if _, err = tx.Exec(ctx, `DELETE FROM journal_lines WHERE entry_id = $1`, entryID); err != nil {
			return err
		}
		if _, err = tx.Exec(ctx, `UPDATE journal_entries SET entry_date = $2, description = $3, base_currency = $4 WHERE id = $1`, entryID, transfer.TransferDate, transfer.Description, baseCurrency); err != nil {
			return err
		}
	}

	// Debit destination asset, credit source asset.
	if _, err = tx.Exec(ctx, `
		INSERT INTO journal_lines (entry_id, account_id, debit, credit, base_debit, base_credit, description)
		VALUES ($1, $2, $3, 0, $4, 0, 'transfer debit')
	`, entryID, toAssetAccountID, transfer.ToAmount, baseAmount); err != nil {
		return err
	}

	if _, err = tx.Exec(ctx, `
		INSERT INTO journal_lines (entry_id, account_id, debit, credit, base_debit, base_credit, description)
		VALUES ($1, $2, 0, $3, 0, $4, 'transfer credit')
	`, entryID, fromAssetAccountID, transfer.FromAmount, baseAmount); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func (r *AccountingRepository) DeleteTransferEntry(ctx context.Context, userID, transferID uuid.UUID) error {
	_, err := r.db.Pool.Exec(ctx, `DELETE FROM journal_entries WHERE user_id = $1 AND reference_type = 'transfer' AND reference_id = $2`, userID, transferID)
	return err
}

func (r *AccountingRepository) DeleteTransactionEntry(ctx context.Context, userID, transactionID uuid.UUID) error {
	_, err := r.db.Pool.Exec(ctx, `DELETE FROM journal_entries WHERE user_id = $1 AND reference_type = 'transaction' AND reference_id = $2`, userID, transactionID)
	return err
}

func (r *AccountingRepository) GetAccountBalance(ctx context.Context, accountID uuid.UUID) (decimal.Decimal, error) {
	query := `
		SELECT
			CASE
				WHEN a.account_type IN ('asset', 'expense')
					THEN a.opening_balance + COALESCE(SUM(jl.debit), 0) - COALESCE(SUM(jl.credit), 0)
				ELSE a.opening_balance + COALESCE(SUM(jl.credit), 0) - COALESCE(SUM(jl.debit), 0)
			END AS balance
		FROM accounts a
		LEFT JOIN journal_lines jl ON jl.account_id = a.id
		LEFT JOIN journal_entries je ON je.id = jl.entry_id
		WHERE a.id = $1
		GROUP BY a.id, a.account_type, a.opening_balance
	`

	var balance decimal.Decimal
	err := r.db.Pool.QueryRow(ctx, query, accountID).Scan(&balance)
	return balance, err
}

func (r *AccountingRepository) AdjustAssetToValue(ctx context.Context, userID uuid.UUID, asset *model.Asset, targetValue decimal.Decimal, reason string) error {
	assetAccountID, err := r.EnsureAssetAccount(ctx, asset)
	if err != nil {
		return err
	}

	balance, err := r.GetAccountBalance(ctx, assetAccountID)
	if err != nil {
		return err
	}

	delta := targetValue.Sub(balance)
	if delta.IsZero() {
		return nil
	}

	equityAccountID, err := r.EnsureBalanceAdjustmentAccount(ctx, userID, asset.Currency)
	if err != nil {
		return err
	}

	tx, err := r.db.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var entryID uuid.UUID
	desc := "asset balance adjustment"
	if reason != "" {
		desc = desc + ": " + reason
	}
	err = tx.QueryRow(ctx, `
		INSERT INTO journal_entries (user_id, entry_date, description, source, reference_type, base_currency)
		VALUES ($1, $2, $3, 'asset_adjustment', 'asset_adjustment', $4)
		RETURNING id
	`, userID, time.Now(), desc, asset.Currency).Scan(&entryID)
	if err != nil {
		return err
	}

	amount := delta.Abs()
	var debitAccount, creditAccount uuid.UUID
	if delta.IsPositive() {
		if asset.IsLiability {
			debitAccount = equityAccountID
			creditAccount = assetAccountID
		} else {
			debitAccount = assetAccountID
			creditAccount = equityAccountID
		}
	} else {
		if asset.IsLiability {
			debitAccount = assetAccountID
			creditAccount = equityAccountID
		} else {
			debitAccount = equityAccountID
			creditAccount = assetAccountID
		}
	}

	if _, err = tx.Exec(ctx, `INSERT INTO journal_lines (entry_id, account_id, debit, credit, base_debit, base_credit, description) VALUES ($1, $2, $3, 0, $4, 0, 'adjustment debit')`, entryID, debitAccount, amount, amount); err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `INSERT INTO journal_lines (entry_id, account_id, debit, credit, base_debit, base_credit, description) VALUES ($1, $2, 0, $3, 0, $4, 'adjustment credit')`, entryID, creditAccount, amount, amount); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func (r *AccountingRepository) ListAccountsWithBalances(ctx context.Context, userID uuid.UUID) ([]model.AccountWithBalance, error) {
	query := `
		SELECT
			a.id, a.user_id, a.name, a.account_type, a.currency, a.opening_balance,
				a.asset_id, a.category_id, a.is_system, a.created_at, a.updated_at,
				CASE
					WHEN a.account_type IN ('asset', 'expense')
						THEN a.opening_balance + COALESCE(SUM(jl.debit), 0) - COALESCE(SUM(jl.credit), 0)
					ELSE a.opening_balance + COALESCE(SUM(jl.credit), 0) - COALESCE(SUM(jl.debit), 0)
				END AS balance
		FROM accounts a
		LEFT JOIN journal_lines jl ON jl.account_id = a.id
		LEFT JOIN journal_entries je ON je.id = jl.entry_id AND je.user_id = a.user_id
		WHERE a.user_id = $1
		GROUP BY a.id, a.user_id, a.name, a.account_type, a.currency, a.opening_balance,
			a.asset_id, a.category_id, a.is_system, a.created_at, a.updated_at
		ORDER BY a.account_type, a.name
	`

	rows, err := r.db.Pool.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var accounts []model.AccountWithBalance
	for rows.Next() {
		var a model.AccountWithBalance
		if err := rows.Scan(
			&a.ID, &a.UserID, &a.Name, &a.AccountType, &a.Currency, &a.OpeningBalance,
			&a.AssetID, &a.CategoryID, &a.IsSystem, &a.CreatedAt, &a.UpdatedAt, &a.Balance,
		); err != nil {
			return nil, err
		}
		accounts = append(accounts, a)
	}

	return accounts, rows.Err()
}

func (r *AccountingRepository) ListJournalEntries(ctx context.Context, userID uuid.UUID, limit int) ([]model.JournalEntry, error) {
	if limit <= 0 {
		limit = 50
	}

	entryQuery := `
		SELECT id, user_id, entry_date, description, source, COALESCE(reference_type, ''), reference_id, base_currency, created_at
		FROM journal_entries
		WHERE user_id = $1
		ORDER BY entry_date DESC, created_at DESC
		LIMIT $2
	`

	entryRows, err := r.db.Pool.Query(ctx, entryQuery, userID, limit)
	if err != nil {
		return nil, err
	}
	defer entryRows.Close()

	entries := make([]model.JournalEntry, 0, limit)
	entryOrder := make([]uuid.UUID, 0, limit)
	entryMap := map[uuid.UUID]*model.JournalEntry{}

	for entryRows.Next() {
		var e model.JournalEntry
		if err := entryRows.Scan(&e.ID, &e.UserID, &e.EntryDate, &e.Description, &e.Source, &e.ReferenceType, &e.ReferenceID, &e.BaseCurrency, &e.CreatedAt); err != nil {
			return nil, err
		}
		e.Lines = []model.JournalLine{}
		entries = append(entries, e)
		entryOrder = append(entryOrder, e.ID)
		entryMap[e.ID] = &entries[len(entries)-1]
	}
	if err := entryRows.Err(); err != nil {
		return nil, err
	}
	if len(entries) == 0 {
		return entries, nil
	}

	lineQuery := `
		SELECT jl.id, jl.entry_id, jl.account_id, a.name, a.account_type, jl.debit, jl.credit, jl.base_debit, jl.base_credit, jl.description, jl.created_at
		FROM journal_lines jl
		JOIN accounts a ON a.id = jl.account_id
		JOIN journal_entries je ON je.id = jl.entry_id
		WHERE je.user_id = $1
		ORDER BY je.entry_date DESC, je.created_at DESC, jl.created_at ASC
	`

	lineRows, err := r.db.Pool.Query(ctx, lineQuery, userID)
	if err != nil {
		return nil, err
	}
	defer lineRows.Close()

	for lineRows.Next() {
		var line model.JournalLine
		if err := lineRows.Scan(&line.ID, &line.EntryID, &line.AccountID, &line.AccountName, &line.AccountType, &line.Debit, &line.Credit, &line.BaseDebit, &line.BaseCredit, &line.Description, &line.CreatedAt); err != nil {
			return nil, err
		}
		if entry, ok := entryMap[line.EntryID]; ok {
			entry.Lines = append(entry.Lines, line)
		}
	}
	if err := lineRows.Err(); err != nil {
		return nil, err
	}

	order := make(map[uuid.UUID]int, len(entryOrder))
	for i, id := range entryOrder {
		order[id] = i
	}
	sort.SliceStable(entries, func(i, j int) bool {
		return order[entries[i].ID] < order[entries[j].ID]
	})

	return entries, nil
}

func (r *AccountingRepository) getUserBaseCurrency(ctx context.Context, userID uuid.UUID) (string, error) {
	var baseCurrency string
	err := r.db.Pool.QueryRow(ctx, `SELECT base_currency FROM users WHERE id = $1`, userID).Scan(&baseCurrency)
	if err != nil {
		return "", err
	}
	return baseCurrency, nil
}

func (r *AccountingRepository) convertToBase(ctx context.Context, amount decimal.Decimal, fromCurrency, baseCurrency string) (decimal.Decimal, error) {
	if fromCurrency == baseCurrency {
		return amount, nil
	}

	var rate decimal.Decimal
	err := r.db.Pool.QueryRow(ctx, `SELECT rate FROM exchange_rates WHERE from_currency = $1 AND to_currency = $2`, fromCurrency, baseCurrency).Scan(&rate)
	if err == nil {
		return amount.Mul(rate).Round(2), nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return decimal.Zero, err
	}

	err = r.db.Pool.QueryRow(ctx, `SELECT rate FROM exchange_rates WHERE from_currency = $1 AND to_currency = $2`, baseCurrency, fromCurrency).Scan(&rate)
	if err == nil {
		if rate.IsZero() {
			return decimal.Zero, errors.New("invalid zero exchange rate")
		}
		return amount.Div(rate).Round(2), nil
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return decimal.Zero, errors.New("missing exchange rate to base currency")
	}

	return decimal.Zero, err
}
