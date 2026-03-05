-- CPF accounts table
CREATE TABLE cpf_accounts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID UNIQUE NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    oa_balance DECIMAL(15, 2) NOT NULL DEFAULT 0,
    sa_balance DECIMAL(15, 2) NOT NULL DEFAULT 0,
    ma_balance DECIMAL(15, 2) NOT NULL DEFAULT 0,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_cpf_accounts_user_id ON cpf_accounts(user_id);

-- CPF contributions table
CREATE TABLE cpf_contributions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    contribution_month CHAR(7) NOT NULL,  -- Format: YYYY-MM
    employee_amount DECIMAL(15, 2) NOT NULL DEFAULT 0,
    employer_amount DECIMAL(15, 2) NOT NULL DEFAULT 0,
    oa_amount DECIMAL(15, 2) NOT NULL DEFAULT 0,
    sa_amount DECIMAL(15, 2) NOT NULL DEFAULT 0,
    ma_amount DECIMAL(15, 2) NOT NULL DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(user_id, contribution_month)
);

CREATE INDEX idx_cpf_contributions_user_id ON cpf_contributions(user_id);
CREATE INDEX idx_cpf_contributions_month ON cpf_contributions(contribution_month);
