CREATE TABLE IF NOT EXISTS webhook_events (
    event_id VARCHAR(150) PRIMARY KEY,
    payment_reference VARCHAR(100) NOT NULL,
    event_type VARCHAR(50) NOT NULL CHECK (event_type IN ('payment.success', 'payment.failed')),
    received_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_webhook_events_payment_reference
    ON webhook_events (payment_reference);