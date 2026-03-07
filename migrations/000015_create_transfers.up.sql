CREATE TABLE transfers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    from_asset_id UUID NOT NULL REFERENCES assets(id) ON DELETE RESTRICT,
    to_asset_id UUID NOT NULL REFERENCES assets(id) ON DELETE RESTRICT,
    from_amount DECIMAL(20, 2) NOT NULL,
    to_amount DECIMAL(20, 2) NOT NULL,
    from_currency CHAR(3) NOT NULL REFERENCES currencies(code),
    to_currency CHAR(3) NOT NULL REFERENCES currencies(code),
    exchange_rate DECIMAL(20, 10) NOT NULL,
    transfer_date TIMESTAMP WITH TIME ZONE NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    CHECK (from_asset_id <> to_asset_id),
    CHECK (from_amount > 0),
    CHECK (to_amount > 0),
    CHECK (exchange_rate > 0)
);

CREATE INDEX idx_transfers_user_date ON transfers(user_id, transfer_date DESC);
CREATE INDEX idx_transfers_from_asset ON transfers(from_asset_id);
CREATE INDEX idx_transfers_to_asset ON transfers(to_asset_id);
