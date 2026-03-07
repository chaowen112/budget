ALTER TABLE journal_entries
ADD COLUMN base_currency CHAR(3) REFERENCES currencies(code);

UPDATE journal_entries je
SET base_currency = u.base_currency
FROM users u
WHERE je.user_id = u.id
  AND je.base_currency IS NULL;

ALTER TABLE journal_entries
ALTER COLUMN base_currency SET NOT NULL;

ALTER TABLE journal_lines
ADD COLUMN base_debit DECIMAL(20, 2) NOT NULL DEFAULT 0,
ADD COLUMN base_credit DECIMAL(20, 2) NOT NULL DEFAULT 0;

UPDATE journal_lines
SET base_debit = debit,
    base_credit = credit;

ALTER TABLE journal_lines
ADD CONSTRAINT journal_lines_base_side_check CHECK ((base_debit = 0 AND base_credit > 0) OR (base_credit = 0 AND base_debit > 0));

CREATE INDEX idx_journal_entries_base_currency ON journal_entries(base_currency);
