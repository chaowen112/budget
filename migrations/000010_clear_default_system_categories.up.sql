-- Remove default system categories so categories are fully user-defined.
-- Keep categories already referenced by existing transactions/budgets.
DELETE FROM categories c
WHERE c.is_system = true
  AND NOT EXISTS (
    SELECT 1
    FROM transactions t
    WHERE t.category_id = c.id
  )
  AND NOT EXISTS (
    SELECT 1
    FROM budgets b
    WHERE b.category_id = c.id
  );
