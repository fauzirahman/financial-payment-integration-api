package repository

import (
	"context"

	"github.com/fauzirahman/financial-payment-integration-api/internal/model"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PaymentRepository interface {
	FindAll(ctx context.Context) ([]model.Payment, error)
	Create(ctx context.Context, payment *model.Payment) error
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
			id,
			reference,
			amount,
			currency,
			status,
			created_at,
			updated_at
		FROM payments
		ORDER BY id ASC
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
