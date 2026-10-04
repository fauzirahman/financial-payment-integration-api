CREATE TABLE IF NOT EXISTS idempotency_keys (
    key VARCHAR(150) PRIMARY KEY,
    payment_reference VARCHAR(100) NOT NULL,
    request_hash VARCHAR(255) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_idempotency_keys_payment_reference
    ON idempotency_keys (payment_reference);

CREATE TABLE IF NOT EXISTS ledger_entries (
    id BIGSERIAL PRIMARY KEY,
    payment_reference VARCHAR(100) NOT NULL,
    account_code VARCHAR(30) NOT NULL,
    entry_type VARCHAR(10) NOT NULL CHECK (entry_type IN ('DEBIT', 'CREDIT')),
    amount BIGINT NOT NULL CHECK (amount > 0),
    currency VARCHAR(3) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_ledger_entries_payment_reference
    ON ledger_entries (payment_reference, created_at DESC);
