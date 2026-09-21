CREATE INDEX IF NOT EXISTS idx_payments_status
    ON payments (status);

CREATE INDEX IF NOT EXISTS idx_payments_created_at
    ON payments (created_at DESC);
