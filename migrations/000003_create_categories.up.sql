-- Category types
CREATE TYPE category_type AS ENUM ('expense', 'income');

-- Categories table
CREATE TABLE categories (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,  -- NULL for system categories
    name VARCHAR(100) NOT NULL,
    type category_type NOT NULL,
    icon VARCHAR(50),
    is_system BOOLEAN DEFAULT false,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_categories_user_id ON categories(user_id);
CREATE INDEX idx_categories_type ON categories(type);

-- Seed system categories (expense)
INSERT INTO categories (name, type, icon, is_system) VALUES
    ('Food & Dining', 'expense', 'utensils', true),
    ('Groceries', 'expense', 'shopping-cart', true),
    ('Transportation', 'expense', 'car', true),
    ('Public Transport', 'expense', 'bus', true),
    ('Shopping', 'expense', 'shopping-bag', true),
    ('Entertainment', 'expense', 'film', true),
    ('Bills & Utilities', 'expense', 'file-text', true),
    ('Healthcare', 'expense', 'heart', true),
    ('Education', 'expense', 'book', true),
    ('Personal Care', 'expense', 'scissors', true),
    ('Home & Garden', 'expense', 'home', true),
    ('Gifts & Donations', 'expense', 'gift', true),
    ('Travel', 'expense', 'plane', true),
    ('Insurance', 'expense', 'shield', true),
    ('Subscriptions', 'expense', 'repeat', true),
    ('Taxes', 'expense', 'percent', true),
    ('Fees & Charges', 'expense', 'credit-card', true),
    ('Other Expense', 'expense', 'more-horizontal', true);

-- Seed system categories (income)
INSERT INTO categories (name, type, icon, is_system) VALUES
    ('Salary', 'income', 'briefcase', true),
    ('Bonus', 'income', 'award', true),
    ('Freelance', 'income', 'laptop', true),
    ('Investment Income', 'income', 'trending-up', true),
    ('Rental Income', 'income', 'home', true),
    ('Interest', 'income', 'percent', true),
    ('Dividends', 'income', 'dollar-sign', true),
    ('Refunds', 'income', 'rotate-ccw', true),
    ('Gifts Received', 'income', 'gift', true),
    ('Other Income', 'income', 'more-horizontal', true);
