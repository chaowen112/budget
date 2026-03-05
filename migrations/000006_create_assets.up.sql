-- Asset category enum
CREATE TYPE asset_category AS ENUM ('cash', 'bank', 'investment', 'retirement', 'property', 'liability', 'custom');

-- Asset types table
CREATE TABLE asset_types (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(100) NOT NULL,
    category asset_category NOT NULL,
    is_system BOOLEAN DEFAULT false
);

-- Seed system asset types
INSERT INTO asset_types (name, category, is_system) VALUES
    ('Cash', 'cash', true),
    ('Savings Account', 'bank', true),
    ('Checking Account', 'bank', true),
    ('Fixed Deposit', 'bank', true),
    ('Stocks', 'investment', true),
    ('ETF', 'investment', true),
    ('Mutual Fund', 'investment', true),
    ('Bonds', 'investment', true),
    ('Cryptocurrency', 'investment', true),
    ('Robo-Advisor', 'investment', true),
    ('CPF OA', 'retirement', true),
    ('CPF SA', 'retirement', true),
    ('CPF MA', 'retirement', true),
    ('SRS', 'retirement', true),
    ('401k', 'retirement', true),
    ('IRA', 'retirement', true),
    ('Property', 'property', true),
    ('Vehicle', 'property', true),
    ('Credit Card', 'liability', true),
    ('Personal Loan', 'liability', true),
    ('Mortgage', 'liability', true),
    ('Student Loan', 'liability', true),
    ('Car Loan', 'liability', true),
    ('Other Asset', 'custom', true),
    ('Other Liability', 'liability', true);

-- Assets table
CREATE TABLE assets (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    asset_type_id UUID NOT NULL REFERENCES asset_types(id),
    name VARCHAR(200) NOT NULL,
    currency CHAR(3) NOT NULL REFERENCES currencies(code),
    current_value DECIMAL(20, 2) NOT NULL DEFAULT 0,
    is_liability BOOLEAN DEFAULT false,
    custom_fields JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_assets_user_id ON assets(user_id);
CREATE INDEX idx_assets_asset_type_id ON assets(asset_type_id);
CREATE INDEX idx_assets_is_liability ON assets(is_liability);

-- Asset snapshots for historical tracking
CREATE TABLE asset_snapshots (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    asset_id UUID NOT NULL REFERENCES assets(id) ON DELETE CASCADE,
    value DECIMAL(20, 2) NOT NULL,
    recorded_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_asset_snapshots_asset_id ON asset_snapshots(asset_id);
CREATE INDEX idx_asset_snapshots_recorded_at ON asset_snapshots(recorded_at);
