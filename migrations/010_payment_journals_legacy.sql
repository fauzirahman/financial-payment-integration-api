-- Reporting indexes for the UUID-based financial schema from migration 006.
-- The ledger is already linked to its financial transaction and account, so
-- this migration does not add a separate payment journal or rewrite ledger data.
BEGIN;

CREATE INDEX IF NOT EXISTS idx_ledger_entries_transaction_account_created
    ON public.ledger_entries (financial_transaction_id, chart_of_account_id, created_at, id);

CREATE INDEX IF NOT EXISTS idx_ledger_entries_account_currency_created
    ON public.ledger_entries (chart_of_account_id, currency, created_at, id);

COMMIT;
