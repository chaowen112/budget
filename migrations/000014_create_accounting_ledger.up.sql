CREATE TYPE account_type AS ENUM ('asset', 'liability', 'equity', 'income', 'expense');

CREATE TABLE accounts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name VARCHAR(200) NOT NULL,
    account_type account_type NOT NULL,
    currency CHAR(3) NOT NULL REFERENCES currencies(code),
    opening_balance DECIMAL(20, 2) NOT NULL DEFAULT 0,
    asset_id UUID NULL REFERENCES assets(id) ON DELETE CASCADE,
    category_id UUID NULL REFERENCES categories(id) ON DELETE CASCADE,
    is_system BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX uq_accounts_asset_id ON accounts(asset_id) WHERE asset_id IS NOT NULL;
CREATE UNIQUE INDEX uq_accounts_category_id ON accounts(category_id) WHERE category_id IS NOT NULL;
CREATE INDEX idx_accounts_user_type ON accounts(user_id, account_type);

CREATE TABLE journal_entries (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    entry_date TIMESTAMP WITH TIME ZONE NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    source TEXT NOT NULL DEFAULT 'manual',
    reference_type TEXT NULL,
    reference_id UUID NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX uq_journal_entries_reference
    ON journal_entries(user_id, reference_type, reference_id)
    WHERE reference_type IS NOT NULL AND reference_id IS NOT NULL;

CREATE INDEX idx_journal_entries_user_date ON journal_entries(user_id, entry_date);

CREATE TABLE journal_lines (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    entry_id UUID NOT NULL REFERENCES journal_entries(id) ON DELETE CASCADE,
    account_id UUID NOT NULL REFERENCES accounts(id) ON DELETE RESTRICT,
    debit DECIMAL(20, 2) NOT NULL DEFAULT 0,
    credit DECIMAL(20, 2) NOT NULL DEFAULT 0,
    description TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    CHECK (debit >= 0),
    CHECK (credit >= 0),
    CHECK ((debit = 0 AND credit > 0) OR (credit = 0 AND debit > 0))
);

CREATE INDEX idx_journal_lines_entry_id ON journal_lines(entry_id);
CREATE INDEX idx_journal_lines_account_id ON journal_lines(account_id);

-- Create one balancing equity account per user.
INSERT INTO accounts (user_id, name, account_type, currency, opening_balance, is_system)
SELECT u.id, 'Balance Adjustments', 'equity', u.base_currency, 0, true
FROM users u
WHERE NOT EXISTS (
    SELECT 1 FROM accounts a WHERE a.user_id = u.id AND a.account_type = 'equity' AND a.is_system = true
);

-- Create one ledger account per asset, seeded with current value as opening balance.
INSERT INTO accounts (user_id, name, account_type, currency, opening_balance, asset_id, is_system)
SELECT
    a.user_id,
    a.name,
    CASE WHEN a.is_liability THEN 'liability'::account_type ELSE 'asset'::account_type END,
    a.currency,
    a.current_value,
    a.id,
    true
FROM assets a
WHERE NOT EXISTS (
    SELECT 1 FROM accounts acc WHERE acc.asset_id = a.id
);

-- Create one ledger account per user category.
INSERT INTO accounts (user_id, name, account_type, currency, opening_balance, category_id, is_system)
SELECT
    c.user_id,
    c.name,
    CASE WHEN c.type = 'income' THEN 'income'::account_type ELSE 'expense'::account_type END,
    u.base_currency,
    0,
    c.id,
    true
FROM categories c
JOIN users u ON u.id = c.user_id
WHERE c.user_id IS NOT NULL
  AND NOT EXISTS (
      SELECT 1 FROM accounts acc WHERE acc.category_id = c.id
  );

-- Backfill transactions into balanced journal entries.
INSERT INTO journal_entries (user_id, entry_date, description, source, reference_type, reference_id)
SELECT
    t.user_id,
    t.transaction_date,
    COALESCE(t.description, ''),
    'backfill_transaction',
    'transaction',
    t.id
FROM transactions t
JOIN transaction_asset_links tal ON tal.transaction_id = t.id
WHERE NOT EXISTS (
    SELECT 1
    FROM journal_entries je
    WHERE je.user_id = t.user_id
      AND je.reference_type = 'transaction'
      AND je.reference_id = t.id
);

INSERT INTO journal_lines (entry_id, account_id, debit, credit, description)
SELECT
    je.id,
    CASE WHEN t.type = 'income' THEN asset_acc.id ELSE category_acc.id END,
    t.amount,
    0,
    'backfill debit'
FROM transactions t
JOIN journal_entries je ON je.user_id = t.user_id AND je.reference_type = 'transaction' AND je.reference_id = t.id
JOIN transaction_asset_links tal ON tal.transaction_id = t.id
JOIN accounts asset_acc ON asset_acc.asset_id = tal.asset_id
JOIN accounts category_acc ON category_acc.category_id = t.category_id
WHERE NOT EXISTS (
    SELECT 1 FROM journal_lines jl WHERE jl.entry_id = je.id
);

INSERT INTO journal_lines (entry_id, account_id, debit, credit, description)
SELECT
    je.id,
    CASE WHEN t.type = 'income' THEN category_acc.id ELSE asset_acc.id END,
    0,
    t.amount,
    'backfill credit'
FROM transactions t
JOIN journal_entries je ON je.user_id = t.user_id AND je.reference_type = 'transaction' AND je.reference_id = t.id
JOIN transaction_asset_links tal ON tal.transaction_id = t.id
JOIN accounts asset_acc ON asset_acc.asset_id = tal.asset_id
JOIN accounts category_acc ON category_acc.category_id = t.category_id
WHERE EXISTS (
    SELECT 1 FROM journal_lines jl WHERE jl.entry_id = je.id
)
  AND (SELECT COUNT(*) FROM journal_lines jl WHERE jl.entry_id = je.id) = 1;
