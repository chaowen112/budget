DROP INDEX IF EXISTS idx_journal_lines_account_id;
DROP INDEX IF EXISTS idx_journal_lines_entry_id;
DROP TABLE IF EXISTS journal_lines;

DROP INDEX IF EXISTS idx_journal_entries_user_date;
DROP INDEX IF EXISTS uq_journal_entries_reference;
DROP TABLE IF EXISTS journal_entries;

DROP INDEX IF EXISTS idx_accounts_user_type;
DROP INDEX IF EXISTS uq_accounts_category_id;
DROP INDEX IF EXISTS uq_accounts_asset_id;
DROP TABLE IF EXISTS accounts;

DROP TYPE IF EXISTS account_type;
