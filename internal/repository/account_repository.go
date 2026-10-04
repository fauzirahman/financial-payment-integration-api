package repository

import (
	"context"

	"github.com/fauzirahman/financial-payment-integration-api/internal/model"
	"github.com/jackc/pgx/v5/pgxpool"
)

type AccountRepository interface {
	Create(ctx context.Context, account *model.Account) error
	FindByID(ctx context.Context, id string) (*model.Account, error)
	FindByCustomerID(ctx context.Context, customerID string) ([]model.Account, error)
	FindByAccountNumber(ctx context.Context, number string) (*model.Account, error)
}

type PostgresAccountRepository struct {
	db *pgxpool.Pool
}

func NewPostgresAccountRepository(db *pgxpool.Pool) *PostgresAccountRepository {
	return &PostgresAccountRepository{db: db}
}

func (r *PostgresAccountRepository) Create(ctx context.Context, account *model.Account) error {
	query := `
		INSERT INTO accounts (customer_id, account_number, currency, balance, status)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at, updated_at
	`
	return r.db.QueryRow(
		ctx,
		query,
		account.CustomerID,
		account.AccountNumber,
		account.Currency,
		account.Balance,
		account.Status,
	).Scan(&account.ID, &account.CreatedAt, &account.UpdatedAt)
}

func (r *PostgresAccountRepository) FindByID(ctx context.Context, id string) (*model.Account, error) {
	query := `
		SELECT id, customer_id, account_number, currency, balance, status, created_at, updated_at
		FROM accounts WHERE id = $1
	`
	var account model.Account
	err := r.db.QueryRow(ctx, query, id).Scan(
		&account.ID,
		&account.CustomerID,
		&account.AccountNumber,
		&account.Currency,
		&account.Balance,
		&account.Status,
		&account.CreatedAt,
		&account.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &account, nil
}

func (r *PostgresAccountRepository) FindByCustomerID(ctx context.Context, customerID string) ([]model.Account, error) {
	query := `
		SELECT id, customer_id, account_number, currency, balance, status, created_at, updated_at
		FROM accounts WHERE customer_id = $1 ORDER BY created_at DESC
	`
	rows, err := r.db.Query(ctx, query, customerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	accounts := make([]model.Account, 0)
	for rows.Next() {
		var account model.Account
		if err := rows.Scan(
			&account.ID,
			&account.CustomerID,
			&account.AccountNumber,
			&account.Currency,
			&account.Balance,
			&account.Status,
			&account.CreatedAt,
			&account.UpdatedAt,
		); err != nil {
			return nil, err
		}
		accounts = append(accounts, account)
	}
	return accounts, rows.Err()
}

func (r *PostgresAccountRepository) FindByAccountNumber(ctx context.Context, number string) (*model.Account, error) {
	query := `
		SELECT id, customer_id, account_number, currency, balance, status, created_at, updated_at
		FROM accounts WHERE account_number = $1
	`
	var account model.Account
	err := r.db.QueryRow(ctx, query, number).Scan(
		&account.ID,
		&account.CustomerID,
		&account.AccountNumber,
		&account.Currency,
		&account.Balance,
		&account.Status,
		&account.CreatedAt,
		&account.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &account, nil
}
