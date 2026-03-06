-- Re-seed default system expense categories.
INSERT INTO categories (name, type, icon, is_system)
SELECT v.name, v.type::category_type, v.icon, true
FROM (
  VALUES
    ('Food & Dining', 'expense', 'utensils'),
    ('Groceries', 'expense', 'shopping-cart'),
    ('Transportation', 'expense', 'car'),
    ('Public Transport', 'expense', 'bus'),
    ('Shopping', 'expense', 'shopping-bag'),
    ('Entertainment', 'expense', 'film'),
    ('Bills & Utilities', 'expense', 'file-text'),
    ('Healthcare', 'expense', 'heart'),
    ('Education', 'expense', 'book'),
    ('Personal Care', 'expense', 'scissors'),
    ('Home & Garden', 'expense', 'home'),
    ('Gifts & Donations', 'expense', 'gift'),
    ('Travel', 'expense', 'plane'),
    ('Insurance', 'expense', 'shield'),
    ('Subscriptions', 'expense', 'repeat'),
    ('Taxes', 'expense', 'percent'),
    ('Fees & Charges', 'expense', 'credit-card'),
    ('Other Expense', 'expense', 'more-horizontal')
) AS v(name, type, icon)
WHERE NOT EXISTS (
  SELECT 1
  FROM categories c
  WHERE c.user_id IS NULL
    AND c.is_system = true
    AND c.type = v.type::category_type
    AND c.name = v.name
);

-- Re-seed default system income categories.
INSERT INTO categories (name, type, icon, is_system)
SELECT v.name, v.type::category_type, v.icon, true
FROM (
  VALUES
    ('Salary', 'income', 'briefcase'),
    ('Bonus', 'income', 'award'),
    ('Freelance', 'income', 'laptop'),
    ('Investment Income', 'income', 'trending-up'),
    ('Rental Income', 'income', 'home'),
    ('Interest', 'income', 'percent'),
    ('Dividends', 'income', 'dollar-sign'),
    ('Refunds', 'income', 'rotate-ccw'),
    ('Gifts Received', 'income', 'gift'),
    ('Other Income', 'income', 'more-horizontal')
) AS v(name, type, icon)
WHERE NOT EXISTS (
  SELECT 1
  FROM categories c
  WHERE c.user_id IS NULL
    AND c.is_system = true
    AND c.type = v.type::category_type
    AND c.name = v.name
);
