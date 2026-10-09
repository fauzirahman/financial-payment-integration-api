BEGIN;

-- =========================================================
-- 1. LEDGER INTEGRITY
-- Each ledger entry must have exactly one positive side:
-- DEBIT or CREDIT.
-- =========================================================

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'ledger_entries_one_sided_amount_check'
          AND conrelid = 'public.ledger_entries'::regclass
    ) THEN
        ALTER TABLE public.ledger_entries
            ADD CONSTRAINT ledger_entries_one_sided_amount_check
            CHECK (
                (debit > 0 AND credit = 0)
                OR
                (credit > 0 AND debit = 0)
            )
            NOT VALID;
    END IF;
END;
$$;

-- Validate existing rows before committing.
ALTER TABLE public.ledger_entries
    VALIDATE CONSTRAINT ledger_entries_one_sided_amount_check;


-- =========================================================
-- 2. FINANCIAL TRANSACTION REPORT
-- One row per financial transaction.
-- =========================================================

CREATE OR REPLACE VIEW public.financial_transaction_ledger_report AS
SELECT
    ft.id AS transaction_id,
    ft.transaction_number,
    ft.type AS transaction_type,
    ft.reference_type,
    ft.reference_id,
    ft.description,
    ft.transaction_date,
    ft.status,

    COUNT(le.id)::BIGINT AS entry_count,

    COALESCE(SUM(le.debit), 0)::NUMERIC(20, 2)
        AS total_debit,

    COALESCE(SUM(le.credit), 0)::NUMERIC(20, 2)
        AS total_credit,

    (
        COUNT(le.id) >= 2
        AND COALESCE(SUM(le.debit), 0)
            = COALESCE(SUM(le.credit), 0)
    ) AS is_balanced

FROM public.financial_transactions AS ft
LEFT JOIN public.ledger_entries AS le
    ON le.financial_transaction_id = ft.id

GROUP BY
    ft.id,
    ft.transaction_number,
    ft.type,
    ft.reference_type,
    ft.reference_id,
    ft.description,
    ft.transaction_date,
    ft.status;


-- =========================================================
-- 3. ACCOUNT LEDGER SUMMARY
-- Aggregated debit and credit per chart of account.
-- =========================================================

CREATE OR REPLACE VIEW public.account_ledger_summary AS
SELECT
    coa.id AS account_id,
    coa.code AS account_code,
    coa.name AS account_name,
    coa.type AS account_type,
    coa.normal_balance,
    coa.is_active,

    COUNT(le.id)::BIGINT AS entry_count,

    COALESCE(SUM(le.debit), 0)::NUMERIC(20, 2)
        AS total_debit,

    COALESCE(SUM(le.credit), 0)::NUMERIC(20, 2)
        AS total_credit,

    (
        COALESCE(SUM(le.debit), 0)
        - COALESCE(SUM(le.credit), 0)
    )::NUMERIC(20, 2) AS net_debit_movement

FROM public.chart_of_accounts AS coa
LEFT JOIN public.ledger_entries AS le
    ON le.chart_of_account_id = coa.id

GROUP BY
    coa.id,
    coa.code,
    coa.name,
    coa.type,
    coa.normal_balance,
    coa.is_active;

COMMIT;
