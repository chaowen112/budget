-- Net worth goals table
-- Each user can have only one net worth goal at a time
CREATE TABLE net_worth_goals (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL UNIQUE REFERENCES users(id) ON DELETE CASCADE,
    name VARCHAR(200) NOT NULL,
    target_amount DECIMAL(15, 2) NOT NULL,
    currency CHAR(3) NOT NULL REFERENCES currencies(code),
    notes TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_net_worth_goals_user_id ON net_worth_goals(user_id);
