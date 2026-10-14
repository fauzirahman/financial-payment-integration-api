package repository

import (
	"context"

	"github.com/fauzirahman/financial-payment-integration-api/internal/model"
	"github.com/jackc/pgx/v5/pgxpool"
)

type UserRepository interface {
	Create(context.Context, *model.AuthUser) error
	FindByEmail(context.Context, string) (*model.AuthUser, error)
}

type PostgresUserRepository struct { db *pgxpool.Pool }

func NewPostgresUserRepository(db *pgxpool.Pool) *PostgresUserRepository {
	return &PostgresUserRepository{db: db}
}

func (r *PostgresUserRepository) Create(ctx context.Context, user *model.AuthUser) error {
	return r.db.QueryRow(ctx, `INSERT INTO users (name, email, password_hash)
		VALUES ($1, $2, $3) RETURNING id, created_at, updated_at`,
		user.Name, user.Email, user.PasswordHash).Scan(&user.ID, &user.CreatedAt, &user.UpdatedAt)
}

func (r *PostgresUserRepository) FindByEmail(ctx context.Context, email string) (*model.AuthUser, error) {
	var user model.AuthUser
	err := r.db.QueryRow(ctx, `SELECT id, name, email, password_hash, created_at, updated_at
		FROM users WHERE LOWER(email) = LOWER($1)`, email).Scan(
		&user.ID, &user.Name, &user.Email, &user.PasswordHash, &user.CreatedAt, &user.UpdatedAt)
	if err != nil { return nil, err }
	return &user, nil
}
