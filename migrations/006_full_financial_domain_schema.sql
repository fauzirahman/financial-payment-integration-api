CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE IF NOT EXISTS customers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    customer_number VARCHAR(30) UNIQUE NOT NULL,
    name VARCHAR(150) NOT NULL,
    email VARCHAR(150) UNIQUE NOT NULL,
    phone VARCHAR(30) NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'ACTIVE',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS chart_of_accounts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code VARCHAR(30) UNIQUE NOT NULL,
    name VARCHAR(150) NOT NULL,
    category VARCHAR(30) NOT NULL CHECK (category IN ('ASSET', 'LIABILITY', 'EQUITY', 'REVENUE', 'EXPENSE')),
    account_type VARCHAR(25) NOT NULL CHECK (account_type IN ('BANK', 'CARD', 'SETTLEMENTMENT', 'REVENUE', 'EXPENSE', 'LIABILITY')),
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS accounts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    customer_id UUID NOT NULL,
    account_number VARCHAR(30) UNIQUE NOT NULL,
    currency CHAR(3) NOT NULL,
    balance NUMERIC(20,2) NOT NULL DEFAULT 0,
    status VARCHAR(20) NOT NULL DEFAULT 'ACTIVE',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_accounts_customer
        FOREIGN KEY (customer_id)
        REFERENCES customers(id)
        ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS payments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    customer_id UUID NOT NULL,
    account_id UUID NULL,
    reference VARCHAR(100) UNIQUE NOT NULL,
    amount NUMERIC(20,2) NOT NULL CHECK (amount > 0),
    currency CHAR(3) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'PENDING',
    gateway VARCHAR(50) NULL,
    metadata JSONB NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_payments_customer
        FOREIGN KEY (customer_id)
        REFERENCES customers(id)
        ON DELETE RESTRICT,
    CONSTRAINT fk_payments_account
        FOREIGN KEY (account_id)
        REFERENCES accounts(id)
        ON DELETE SET NULL
);

CREATE TABLE IF NOT EXISTS financial_transactions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    payment_id UUID NULL,
    account_id UUID NOT NULL,
    transaction_type VARCHAR(30) NOT NULL CHECK (transaction_type IN ('DEBIT', 'CREDIT', 'ADJUSTMENT', 'SETTLEMENTMENT')),
    direction VARCHAR(20) NOT NULL CHECK (direction IN ('IN', 'OUT')),
    amount NUMERIC(20,2) NOT NULL CHECK (amount > 0),
    currency CHAR(3) NOT NULL,
    reference VARCHAR(100) UNIQUE NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'POSTED',
    processed_at TIMESTAMPTZ NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_financial_transactions_payment
        FOREIGN KEY (payment_id)
        REFERENCES payments(id)
        ON DELETE SET NULL,
    CONSTRAINT fk_financial_transactions_account
        FOREIGN KEY (account_id)
        REFERENCES accounts(id)
        ON DELETE RESTRICT
);

CREATE TABLE IF NOT EXISTS ledger_entries (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    payment_id UUID NULL,
    account_id UUID NOT NULL,
    chart_of_account_id UUID NOT NULL,
    entry_type VARCHAR(10) NOT NULL CHECK (entry_type IN ('DEBIT', 'CREDIT')),
    amount NUMERIC(20,2) NOT NULL CHECK (amount > 0),
    currency CHAR(3) NOT NULL,
    description VARCHAR(255) NOT NULL,
    posted_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_ledger_entries_payment
        FOREIGN KEY (payment_id)
        REFERENCES payments(id)
        ON DELETE SET NULL,
    CONSTRAINT fk_ledger_entries_account
        FOREIGN KEY (account_id)
        REFERENCES accounts(id)
        ON DELETE RESTRICT,
    CONSTRAINT fk_ledger_entries_chart
        FOREIGN KEY (chart_of_account_id)
        REFERENCES chart_of_accounts(id)
        ON DELETE RESTRICT
);

CREATE TABLE IF NOT EXISTS idempotency_keys (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    key VARCHAR(150) UNIQUE NOT NULL,
    payment_reference VARCHAR(100) NOT NULL,
    request_hash VARCHAR(255) NOT NULL,
    is_consumed BOOLEAN NOT NULL DEFAULT FALSE,
    expires_at TIMESTAMPTZ NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS payment_webhook_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    payment_id UUID NULL,
    event_id VARCHAR(150) UNIQUE NOT NULL,
    event_type VARCHAR(50) NOT NULL CHECK (event_type IN ('payment.success', 'payment.failed', 'payment.pending', 'payment.refunded')),
    payload JSONB NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'RECEIVED',
    attempts INT NOT NULL DEFAULT 0,
    next_retry_at TIMESTAMPTZ NULL,
    received_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    processed_at TIMESTAMPTZ NULL,
    CONSTRAINT fk_payment_webhook_events_payment
        FOREIGN KEY (payment_id)
        REFERENCES payments(id)
        ON DELETE SET NULL
);

CREATE INDEX IF NOT EXISTS idx_accounts_customer_id
    ON accounts (customer_id);

CREATE INDEX IF NOT EXISTS idx_payments_customer_id
    ON payments (customer_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_payments_account_id
    ON payments (account_id);

CREATE INDEX IF NOT EXISTS idx_financial_transactions_account_id
    ON financial_transactions (account_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_financial_transactions_payment_id
    ON financial_transactions (payment_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_ledger_entries_account_id
    ON ledger_entries (account_id, posted_at DESC);

CREATE INDEX IF NOT EXISTS idx_ledger_entries_payment_id
    ON ledger_entries (payment_id, posted_at DESC);

CREATE INDEX IF NOT EXISTS idx_payment_webhook_events_payment_id
    ON payment_webhook_events (payment_id, received_at DESC);
