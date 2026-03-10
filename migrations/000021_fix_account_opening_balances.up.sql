-- Fix double-counting of opening balances caused by migration 000014
-- The opening_balance was seeded with assets.current_value which already included
-- the historical transactions. Then we backfilled those transactions, causing them
-- to be double-counted.
-- We must adjust opening_balance by removing the net effect of ONLY the backfilled transactions.

WITH backfill_totals AS (
    SELECT 
        jl.account_id,
        COALESCE(SUM(jl.debit), 0) AS total_debit,
        COALESCE(SUM(jl.credit), 0) AS total_credit
    FROM journal_lines jl
    JOIN journal_entries je ON je.id = jl.entry_id
    WHERE je.source = 'backfill_transaction'
    GROUP BY jl.account_id
)
UPDATE accounts acc
SET opening_balance = 
    CASE 
        WHEN acc.account_type IN ('asset', 'expense')
            THEN acc.opening_balance - COALESCE(bt.total_debit, 0) + COALESCE(bt.total_credit, 0)
        ELSE 
            -- liability, equity, income increase with credit
            acc.opening_balance - COALESCE(bt.total_credit, 0) + COALESCE(bt.total_debit, 0)
    END
FROM backfill_totals bt
WHERE acc.id = bt.account_id
  AND acc.asset_id IS NOT NULL;
