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
			journal.id,
			journal.payment_reference,
			entry.account_code,
			entry.entry_type,
			CASE WHEN entry.entry_type = 'DEBIT' THEN entry.amount ELSE 0 END,
			CASE WHEN entry.entry_type = 'CREDIT' THEN entry.amount ELSE 0 END,
			SUM(CASE WHEN entry.entry_type = 'DEBIT' THEN entry.amount ELSE -entry.amount END)
				OVER (
					PARTITION BY entry.currency, entry.account_code
					ORDER BY journal.posted_at, journal.id, entry.id
					ROWS BETWEEN UNBOUNDED PRECEDING AND CURRENT ROW
				) AS running_balance,
			TRIM(entry.currency) AS currency,
			journal.posted_at,
			entry.id AS ledger_entry_id
		FROM payment_journals AS journal
		JOIN ledger_entries AS entry ON entry.journal_id = journal.id
		)
		SELECT
			id,
			payment_reference,
			account_code,
			entry_type,
			debit_amount,
			credit_amount,
			running_balance,
			currency,
			posted_at
		FROM ledger_balances
		WHERE ($1::timestamptz IS NULL OR posted_at >= $1)
		  AND ($2::timestamptz IS NULL OR posted_at < $2)
		  AND ($3 = '' OR account_code = $3)
		ORDER BY posted_at, id, ledger_entry_id
	`, filter.From, filter.ToExclusive, filter.AccountCode)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	results := make([]model.GeneralLedgerEntry, 0)
	for rows.Next() {
		var entry model.GeneralLedgerEntry
		if err := rows.Scan(
			&entry.JournalID,
			&entry.PaymentReference,
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
