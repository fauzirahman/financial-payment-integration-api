package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/fauzirahman/financial-payment-integration-api/internal/model"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrIdempotencyKeyExists = errors.New("idempotency key already exists")

type PaymentRepository interface {
	FindAll(ctx context.Context) ([]model.Payment, error)
	FindByID(ctx context.Context, id string) (*model.Payment, error)
	FindByReference(ctx context.Context, reference string) (*model.Payment, error)
	Create(ctx context.Context, payment *model.Payment) error
	CreateWithIdempotencyKey(ctx context.Context, payment *model.Payment, key *model.IdempotencyKey) error
	FindIdempotencyKey(ctx context.Context, key string) (*model.IdempotencyKey, error)
	ApplyWebhookEvent(ctx context.Context, event model.WebhookEvent, status string) (bool, error)
}

type PostgresPaymentRepository struct {
	db *pgxpool.Pool
}

func NewPostgresPaymentRepository(db *pgxpool.Pool) *PostgresPaymentRepository {
	return &PostgresPaymentRepository{
		db: db,
	}
}

func (r *PostgresPaymentRepository) FindAll(ctx context.Context) ([]model.Payment, error) {
	query := `
		SELECT
			id::text,
			reference,
			amount,
			TRIM(currency),
			status,
			created_at,
			updated_at
		FROM payments
		ORDER BY created_at ASC
	`

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	payments := make([]model.Payment, 0)

	for rows.Next() {
		var payment model.Payment

		err := rows.Scan(
			&payment.ID,
			&payment.Reference,
			&payment.Amount,
			&payment.Currency,
			&payment.Status,
			&payment.CreatedAt,
			&payment.UpdatedAt,
		)

		if err != nil {
			return nil, err
		}

		payments = append(payments, payment)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return payments, nil
}

func (r *PostgresPaymentRepository) FindByID(ctx context.Context, id string) (*model.Payment, error) {
	query := `
		SELECT
			id::text,
			reference,
			amount,
			TRIM(currency),
			status,
			created_at,
			updated_at
		FROM payments
		WHERE id = $1::bigint
	`

	var payment model.Payment
	err := r.db.QueryRow(ctx, query, id).Scan(
		&payment.ID,
		&payment.Reference,
		&payment.Amount,
		&payment.Currency,
		&payment.Status,
		&payment.CreatedAt,
		&payment.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	return &payment, nil
}

func (r *PostgresPaymentRepository) FindByReference(ctx context.Context, reference string) (*model.Payment, error) {
	query := `
		SELECT
			id::text,
			reference,
			amount,
			TRIM(currency),
			status,
			created_at,
			updated_at
		FROM payments
		WHERE reference = $1
	`

	var payment model.Payment
	err := r.db.QueryRow(ctx, query, reference).Scan(
		&payment.ID,
		&payment.Reference,
		&payment.Amount,
		&payment.Currency,
		&payment.Status,
		&payment.CreatedAt,
		&payment.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	return &payment, nil
}

func (r *PostgresPaymentRepository) Create(ctx context.Context, payment *model.Payment) error {
	query := `
		INSERT INTO payments (
			reference,
			amount,
			currency,
			status
		)
		VALUES ($1, $2, $3, $4)
		RETURNING
			id,
			created_at,
			updated_at
	`

	return r.db.QueryRow(
		ctx,
		query,
		payment.Reference,
		payment.Amount,
		payment.Currency,
		payment.Status,
	).Scan(
		&payment.ID,
		&payment.CreatedAt,
		&payment.UpdatedAt,
	)
}

func (r *PostgresPaymentRepository) FindIdempotencyKey(ctx context.Context, key string) (*model.IdempotencyKey, error) {
	query := `
		SELECT key, payment_reference, request_hash, created_at, updated_at
		FROM idempotency_keys
		WHERE key = $1
	`

	var record model.IdempotencyKey
	err := r.db.QueryRow(ctx, query, key).Scan(
		&record.Key,
		&record.PaymentReference,
		&record.RequestHash,
		&record.CreatedAt,
		&record.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &record, nil
}

func (r *PostgresPaymentRepository) CreateWithIdempotencyKey(ctx context.Context, payment *model.Payment, key *model.IdempotencyKey) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	err = tx.QueryRow(ctx, `
		INSERT INTO payments (reference, amount, currency, status)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at, updated_at
	`, payment.Reference, payment.Amount, payment.Currency, payment.Status).Scan(
		&payment.ID,
		&payment.CreatedAt,
		&payment.UpdatedAt,
	)
	if err != nil {
		return err
	}

	err = tx.QueryRow(ctx, `
		INSERT INTO idempotency_keys (key, payment_reference, request_hash)
		VALUES ($1, $2, $3)
		RETURNING created_at, updated_at
	`, key.Key, key.PaymentReference, key.RequestHash).Scan(&key.CreatedAt, &key.UpdatedAt)
	if err != nil {
		var postgresError *pgconn.PgError
		if errors.As(err, &postgresError) && postgresError.Code == "23505" {
			return fmt.Errorf("%w: %v", ErrIdempotencyKeyExists, err)
		}
		return err
	}

	return tx.Commit(ctx)
}

func (r *PostgresPaymentRepository) ApplyWebhookEvent(ctx context.Context, event model.WebhookEvent, status string) (bool, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return false, err
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	result, err := tx.Exec(ctx, `
		INSERT INTO webhook_events (event_id, payment_reference, event_type)
		VALUES ($1, $2, $3)
		ON CONFLICT (event_id) DO NOTHING
	`, event.EventID, event.Reference, event.Type)
	if err != nil {
		return false, err
	}
	if result.RowsAffected() == 0 {
		return false, nil
	}

	result, err = tx.Exec(ctx, `
		UPDATE payments
		SET status = $1, updated_at = NOW()
		WHERE reference = $2 AND status IN ($3, $4)
	`, status, event.Reference, model.PaymentStatusPending, model.PaymentStatusProcessing)
	if err != nil {
		return false, err
	}
	if result.RowsAffected() == 0 {
		var currentStatus string
		err := tx.QueryRow(ctx, `SELECT status FROM payments WHERE reference = $1`, event.Reference).Scan(&currentStatus)
		if err != nil {
			return false, err
		}
		return false, fmt.Errorf("%w: cannot transition from %s to %s", model.ErrInvalidPaymentTransition, currentStatus, status)
	}

	if status == model.PaymentStatusSuccess {
		var paymentAmount int64
		var paymentCurrency string
		err = tx.QueryRow(ctx, `SELECT amount, currency FROM payments WHERE reference = $1`, event.Reference).Scan(&paymentAmount, &paymentCurrency)
		if err != nil {
			return false, err
		}
		entries := []model.LedgerEntry{
			{AccountCode: "1010", EntryType: model.LedgerEntryTypeDebit, Amount: paymentAmount, Currency: paymentCurrency},
			{AccountCode: "2010", EntryType: model.LedgerEntryTypeCredit, Amount: paymentAmount, Currency: paymentCurrency},
		}
		var debitTotal, creditTotal int64
		for _, entry := range entries {
			if entry.EntryType == model.LedgerEntryTypeDebit {
				debitTotal += entry.Amount
			} else if entry.EntryType == model.LedgerEntryTypeCredit {
				creditTotal += entry.Amount
			}
		}
		if debitTotal <= 0 || debitTotal != creditTotal {
			return false, errors.New("ledger journal is not balanced")
		}

		result, err := tx.Exec(ctx, `
			INSERT INTO payment_journals (payment_reference, currency, debit_total, credit_total)
			VALUES ($1, $2, $3, $4)
			ON CONFLICT (payment_reference) DO NOTHING
		`, event.Reference, paymentCurrency, debitTotal, creditTotal)
		if err != nil {
			return false, err
		}
		if result.RowsAffected() == 1 {
			var journalID int64
			err = tx.QueryRow(ctx, `
				SELECT id FROM payment_journals WHERE payment_reference = $1
			`, event.Reference).Scan(&journalID)
			if err != nil {
				return false, err
			}
			for _, entry := range entries {
				_, err = tx.Exec(ctx, `
				INSERT INTO ledger_entries (journal_id, payment_reference, account_code, entry_type, amount, currency)
				VALUES ($1, $2, $3, $4, $5, $6)
			`, journalID, event.Reference, entry.AccountCode, entry.EntryType, entry.Amount, entry.Currency)
				if err != nil {
					return false, err
				}
			}
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return false, err
	}
	return true, nil
}
