-- Currencies table
CREATE TABLE currencies (
    code CHAR(3) PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    symbol VARCHAR(10) NOT NULL,
    is_active BOOLEAN DEFAULT true
);

-- Exchange rates
CREATE TABLE exchange_rates (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    from_currency CHAR(3) NOT NULL REFERENCES currencies(code),
    to_currency CHAR(3) NOT NULL REFERENCES currencies(code),
    rate DECIMAL(20, 10) NOT NULL,
    fetched_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(from_currency, to_currency)
);

CREATE INDEX idx_exchange_rates_currencies ON exchange_rates(from_currency, to_currency);

-- Seed common currencies
INSERT INTO currencies (code, name, symbol) VALUES
    ('SGD', 'Singapore Dollar', 'S$'),
    ('USD', 'US Dollar', '$'),
    ('EUR', 'Euro', '€'),
    ('GBP', 'British Pound', '£'),
    ('JPY', 'Japanese Yen', '¥'),
    ('CNY', 'Chinese Yuan', '¥'),
    ('AUD', 'Australian Dollar', 'A$'),
    ('CAD', 'Canadian Dollar', 'C$'),
    ('CHF', 'Swiss Franc', 'CHF'),
    ('HKD', 'Hong Kong Dollar', 'HK$'),
    ('MYR', 'Malaysian Ringgit', 'RM'),
    ('THB', 'Thai Baht', '฿'),
    ('INR', 'Indian Rupee', '₹'),
    ('KRW', 'South Korean Won', '₩'),
    ('TWD', 'Taiwan Dollar', 'NT$'),
    ('PHP', 'Philippine Peso', '₱'),
    ('IDR', 'Indonesian Rupiah', 'Rp'),
    ('VND', 'Vietnamese Dong', '₫'),
    ('NZD', 'New Zealand Dollar', 'NZ$'),
    ('SEK', 'Swedish Krona', 'kr');
