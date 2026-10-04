package repository

import (
	"context"

	"github.com/fauzirahman/financial-payment-integration-api/internal/model"
	"github.com/jackc/pgx/v5/pgxpool"
)

type CustomerRepository interface {
	Create(ctx context.Context, customer *model.Customer) error
	FindByID(ctx context.Context, id string) (*model.Customer, error)
	FindByEmail(ctx context.Context, email string) (*model.Customer, error)
	List(ctx context.Context) ([]model.Customer, error)
}

type PostgresCustomerRepository struct {
	db *pgxpool.Pool
}

func NewPostgresCustomerRepository(db *pgxpool.Pool) *PostgresCustomerRepository {
	return &PostgresCustomerRepository{db: db}
}

func (r *PostgresCustomerRepository) Create(ctx context.Context, customer *model.Customer) error {
	query := `
		INSERT INTO customers (customer_number, name, email, phone, status)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at, updated_at
	`
	return r.db.QueryRow(
		ctx,
		query,
		customer.CustomerNumber,
		customer.Name,
		customer.Email,
		customer.Phone,
		customer.Status,
	).Scan(&customer.ID, &customer.CreatedAt, &customer.UpdatedAt)
}

func (r *PostgresCustomerRepository) FindByID(ctx context.Context, id string) (*model.Customer, error) {
	query := `
		SELECT id, customer_number, name, email, phone, status, created_at, updated_at
		FROM customers WHERE id = $1
	`
	var customer model.Customer
	err := r.db.QueryRow(ctx, query, id).Scan(
		&customer.ID,
		&customer.CustomerNumber,
		&customer.Name,
		&customer.Email,
		&customer.Phone,
		&customer.Status,
		&customer.CreatedAt,
		&customer.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &customer, nil
}

func (r *PostgresCustomerRepository) FindByEmail(ctx context.Context, email string) (*model.Customer, error) {
	query := `
		SELECT id, customer_number, name, email, phone, status, created_at, updated_at
		FROM customers WHERE email = $1
	`
	var customer model.Customer
	err := r.db.QueryRow(ctx, query, email).Scan(
		&customer.ID,
		&customer.CustomerNumber,
		&customer.Name,
		&customer.Email,
		&customer.Phone,
		&customer.Status,
		&customer.CreatedAt,
		&customer.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &customer, nil
}

func (r *PostgresCustomerRepository) List(ctx context.Context) ([]model.Customer, error) {
	query := `
		SELECT id, customer_number, name, email, phone, status, created_at, updated_at
		FROM customers ORDER BY created_at DESC
	`
	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var customers []model.Customer
	for rows.Next() {
		var customer model.Customer
		if err := rows.Scan(
			&customer.ID,
			&customer.CustomerNumber,
			&customer.Name,
			&customer.Email,
			&customer.Phone,
			&customer.Status,
			&customer.CreatedAt,
			&customer.UpdatedAt,
		); err != nil {
			return nil, err
		}
		customers = append(customers, customer)
	}
	return customers, rows.Err()
}
