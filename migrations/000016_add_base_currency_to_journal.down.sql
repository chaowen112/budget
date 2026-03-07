DROP INDEX IF EXISTS idx_journal_entries_base_currency;

ALTER TABLE journal_lines
DROP CONSTRAINT IF EXISTS journal_lines_base_side_check;

ALTER TABLE journal_lines
DROP COLUMN IF EXISTS base_credit,
DROP COLUMN IF EXISTS base_debit;

ALTER TABLE journal_entries
DROP COLUMN IF EXISTS base_currency;
