CREATE TABLE exchange_rate_history (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    from_currency CHAR(3) NOT NULL REFERENCES currencies(code),
    to_currency CHAR(3) NOT NULL REFERENCES currencies(code),
    rate DECIMAL(20, 10) NOT NULL,
    fetched_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_exchange_rate_history_pair_time
    ON exchange_rate_history(from_currency, to_currency, fetched_at DESC);

INSERT INTO exchange_rate_history (from_currency, to_currency, rate, fetched_at)
SELECT from_currency, to_currency, rate, fetched_at
FROM exchange_rates;
