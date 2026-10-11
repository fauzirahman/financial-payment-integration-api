package repository

import (
	"context"
	"time"

	"github.com/fauzirahman/financial-payment-integration-api/internal/model"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ReportFilter struct {
	From        *time.Time
	ToExclusive *time.Time
	AccountCode string
}

type ReportingRepository interface {
	PaymentSummary(ctx context.Context, filter ReportFilter) ([]model.PaymentSummary, error)
	DailyPaymentReport(ctx context.Context, filter ReportFilter) ([]model.DailyPaymentReport, error)
	GeneralLedgerReport(ctx context.Context, filter ReportFilter) ([]model.GeneralLedgerEntry, error)
}

type PostgresReportingRepository struct {
	db *pgxpool.Pool
}

func NewPostgresReportingRepository(db *pgxpool.Pool) *PostgresReportingRepository {
	return &PostgresReportingRepository{db: db}
}

func (r *PostgresReportingRepository) PaymentSummary(ctx context.Context, filter ReportFilter) ([]model.PaymentSummary, error) {
	rows, err := r.db.Query(ctx, `
		SELECT
			TRIM(currency),
			COUNT(*),
			COUNT(*) FILTER (WHERE status = 'PENDING'),
			COUNT(*) FILTER (WHERE status = 'SUCCESS'),
			COUNT(*) FILTER (WHERE status = 'FAILED'),
			COALESCE(SUM(amount), 0),
			COALESCE(SUM(amount) FILTER (WHERE status = 'SUCCESS'), 0)
		FROM payments
		WHERE ($1::timestamptz IS NULL OR created_at >= $1)
		  AND ($2::timestamptz IS NULL OR created_at < $2)
		GROUP BY currency
		ORDER BY currency
	`, filter.From, filter.ToExclusive)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	results := make([]model.PaymentSummary, 0)
	for rows.Next() {
		var report model.PaymentSummary
		if err := rows.Scan(
			&report.Currency,
			&report.TotalPayments,
			&report.PendingPayments,
			&report.SuccessfulPayments,
			&report.FailedPayments,
			&report.TotalAmount,
			&report.SuccessfulAmount,
		); err != nil {
			return nil, err
		}
		results = append(results, report)
	}
	return results, rows.Err()
}

func (r *PostgresReportingRepository) DailyPaymentReport(ctx context.Context, filter ReportFilter) ([]model.DailyPaymentReport, error) {
	rows, err := r.db.Query(ctx, `
		SELECT
			TO_CHAR((created_at AT TIME ZONE 'UTC')::date, 'YYYY-MM-DD'),
			TRIM(currency),
			COUNT(*),
			COUNT(*) FILTER (WHERE status = 'SUCCESS'),
			COUNT(*) FILTER (WHERE status = 'FAILED'),
			COALESCE(SUM(amount), 0),
			COALESCE(SUM(amount) FILTER (WHERE status = 'SUCCESS'), 0)
		FROM payments
		WHERE ($1::timestamptz IS NULL OR created_at >= $1)
		  AND ($2::timestamptz IS NULL OR created_at < $2)
		GROUP BY (created_at AT TIME ZONE 'UTC')::date, currency
		ORDER BY (created_at AT TIME ZONE 'UTC')::date DESC, currency
	`, filter.From, filter.ToExclusive)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	results := make([]model.DailyPaymentReport, 0)
	for rows.Next() {
		var report model.DailyPaymentReport
		if err := rows.Scan(
			&report.Date,
			&report.Currency,
			&report.TotalPayments,
			&report.SuccessfulPayments,
			&report.FailedPayments,
			&report.TotalAmount,
			&report.SuccessfulAmount,
		); err != nil {
			return nil, err
		}
		results = append(results, report)
	}
	return results, rows.Err()
}

func (r *PostgresReportingRepository) GeneralLedgerReport(ctx context.Context, filter ReportFilter) ([]model.GeneralLedgerEntry, error) {
	rows, err := r.db.Query(ctx, `
		WITH ledger_balances AS (
		SELECT
			ft.id::text AS financial_transaction_id,
			ft.reference AS payment_number,
			chart.code AS account_code,
			CASE WHEN entry.debit > 0 THEN 'DEBIT' ELSE 'CREDIT' END AS entry_type,
			entry.debit AS debit_amount,
			entry.credit AS credit_amount,
			SUM(entry.debit - entry.credit)
				OVER (
					PARTITION BY entry.currency, chart.code
					ORDER BY entry.created_at, ft.id, entry.id
					ROWS BETWEEN UNBOUNDED PRECEDING AND CURRENT ROW
				) AS running_balance,
			TRIM(entry.currency) AS currency,
			entry.created_at AS posted_at,
			entry.id AS ledger_entry_id
		FROM ledger_entries AS entry
		JOIN financial_transactions AS ft
			ON ft.id = entry.financial_transaction_id
		JOIN chart_of_accounts AS chart
			ON chart.id = entry.chart_of_account_id
		)
		SELECT
			financial_transaction_id,
			payment_number,
			account_code,
			entry_type,
			debit_amount,
			credit_amount,
			running_balance,
			currency,
			posted_at
		FROM ledger_balances
		WHERE ($1::timestamp IS NULL OR posted_at >= $1::timestamp)
		  AND ($2::timestamp IS NULL OR posted_at < $2::timestamp)
		  AND ($3 = '' OR account_code = $3)
		ORDER BY posted_at, financial_transaction_id, ledger_entry_id
	`, filter.From, filter.ToExclusive, filter.AccountCode)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	results := make([]model.GeneralLedgerEntry, 0)
	for rows.Next() {
		var entry model.GeneralLedgerEntry
		if err := rows.Scan(
			&entry.FinancialTransactionID,
			&entry.PaymentNumber,
			&entry.AccountCode,
			&entry.EntryType,
			&entry.DebitAmount,
			&entry.CreditAmount,
			&entry.RunningBalance,
			&entry.Currency,
			&entry.PostedAt,
		); err != nil {
			return nil, err
		}
		results = append(results, entry)
	}
	return results, rows.Err()
}
