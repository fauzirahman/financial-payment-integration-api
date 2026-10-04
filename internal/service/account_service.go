package service

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/fauzirahman/financial-payment-integration-api/internal/model"
	"github.com/fauzirahman/financial-payment-integration-api/internal/repository"
)

var ErrAccountInvalid = errors.New("invalid account")

var accountNumberPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{1,30}$`)
var currencyPattern = regexp.MustCompile(`^[A-Z]{3}$`)
var accountCustomerIDPattern = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

type AccountService struct {
	repository repository.AccountRepository
}

func NewAccountService(repository repository.AccountRepository) *AccountService {
	return &AccountService{repository: repository}
}

func (s *AccountService) CreateAccount(ctx context.Context, account *model.Account) error {
	if account == nil {
		return fmt.Errorf("%w: account is required", ErrAccountInvalid)
	}
	account.CustomerID = strings.TrimSpace(account.CustomerID)
	account.AccountNumber = strings.TrimSpace(account.AccountNumber)
	account.Currency = strings.ToUpper(strings.TrimSpace(account.Currency))
	account.Status = strings.ToUpper(strings.TrimSpace(account.Status))
	if !accountCustomerIDPattern.MatchString(account.CustomerID) {
		return fmt.Errorf("%w: customer_id must be a UUID", ErrAccountInvalid)
	}
	if !accountNumberPattern.MatchString(account.AccountNumber) {
		return fmt.Errorf("%w: account_number must be 1-30 letters, digits, hyphens, or underscores", ErrAccountInvalid)
	}
	if !currencyPattern.MatchString(account.Currency) {
		return fmt.Errorf("%w: currency must be a 3-letter code", ErrAccountInvalid)
	}
	if account.Balance < 0 {
		return fmt.Errorf("%w: balance cannot be negative", ErrAccountInvalid)
	}
	if account.Status == "" {
		account.Status = "ACTIVE"
	} else if account.Status != "ACTIVE" && account.Status != "INACTIVE" && account.Status != "SUSPENDED" {
		return fmt.Errorf("%w: status must be ACTIVE, INACTIVE, or SUSPENDED", ErrAccountInvalid)
	}
	return s.repository.Create(ctx, account)
}

func (s *AccountService) GetAccountByID(ctx context.Context, id string) (*model.Account, error) {
	return s.repository.FindByID(ctx, id)
}

func (s *AccountService) GetAccountsByCustomerID(ctx context.Context, customerID string) ([]model.Account, error) {
	return s.repository.FindByCustomerID(ctx, customerID)
}
