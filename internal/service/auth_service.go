package service

import (
	"context"
	"errors"
	"net/mail"
	"strings"
	"time"

	"github.com/fauzirahman/financial-payment-integration-api/internal/model"
	"github.com/fauzirahman/financial-payment-integration-api/internal/repository"
	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/bcrypt"
)

var ErrInvalidCredentials = errors.New("invalid email or password")

type AuthService struct { users repository.UserRepository }

func NewAuthService(users repository.UserRepository) *AuthService { return &AuthService{users: users} }

func (s *AuthService) Register(ctx context.Context, name, email, password string) (*model.User, error) {
	name, email = strings.TrimSpace(name), strings.ToLower(strings.TrimSpace(email))
	if name == "" || len(name) > 150 { return nil, errors.New("name is required and must be at most 150 characters") }
	parsed, err := mail.ParseAddress(email)
	if err != nil || parsed.Address != email || len(email) > 254 { return nil, errors.New("valid email is required") }
	if len(password) < 8 || len(password) > 72 { return nil, errors.New("password must be between 8 and 72 bytes") }
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil { return nil, err }
	user := &model.AuthUser{User: model.User{Name: name, Email: email}, PasswordHash: string(hash)}
	if err := s.users.Create(ctx, user); err != nil { return nil, err }
	return &user.User, nil
}

func (s *AuthService) Authenticate(ctx context.Context, email, password string) (*model.User, error) {
	user, err := s.users.FindByEmail(ctx, strings.ToLower(strings.TrimSpace(email)))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrInvalidCredentials
	}
	if err != nil { return nil, err }
	if bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)) != nil { return nil, ErrInvalidCredentials }
	return &user.User, nil
}

type TokenIssuer interface { Issue(userID string, expiresAt time.Time) (string, error) }
