package service

import (
	"context"
	"errors"
	"fmt"
	"net/mail"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/fauzirahman/financial-payment-integration-api/internal/model"
	"github.com/fauzirahman/financial-payment-integration-api/internal/repository"
)

var ErrCustomerInvalid = errors.New("invalid customer")

var customerNumberPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{1,30}$`)
var phonePattern = regexp.MustCompile(`^[+0-9() .-]{1,30}$`)

type CustomerService struct {
	repository repository.CustomerRepository
}

func NewCustomerService(repository repository.CustomerRepository) *CustomerService {
	return &CustomerService{repository: repository}
}

func (s *CustomerService) CreateCustomer(ctx context.Context, customer *model.Customer) error {
	if customer == nil {
		return fmt.Errorf("%w: customer is required", ErrCustomerInvalid)
	}
	customer.CustomerNumber = strings.TrimSpace(customer.CustomerNumber)
	customer.Name = strings.TrimSpace(customer.Name)
	customer.Email = strings.ToLower(strings.TrimSpace(customer.Email))
	customer.Phone = strings.TrimSpace(customer.Phone)
	customer.Status = strings.ToUpper(strings.TrimSpace(customer.Status))

	if !customerNumberPattern.MatchString(customer.CustomerNumber) {
		return fmt.Errorf("%w: customer_number must be 1-30 letters, digits, hyphens, or underscores", ErrCustomerInvalid)
	}
	if customer.Name == "" || utf8.RuneCountInString(customer.Name) > 150 || strings.IndexFunc(customer.Name, unicode.IsControl) >= 0 {
		return fmt.Errorf("%w: name must be 1-150 characters without control characters", ErrCustomerInvalid)
	}
	parsedEmail, err := mail.ParseAddress(customer.Email)
	if err != nil || parsedEmail.Address != customer.Email || len(customer.Email) > 150 {
		return fmt.Errorf("%w: email must be a valid address of at most 150 characters", ErrCustomerInvalid)
	}
	if customer.Phone != "" && (!phonePattern.MatchString(customer.Phone) || strings.IndexFunc(customer.Phone, unicode.IsDigit) < 0) {
		return fmt.Errorf("%w: phone may contain only digits, spaces, +, parentheses, dots, and hyphens (max 30 characters)", ErrCustomerInvalid)
	}
	if customer.Status == "" {
		customer.Status = "ACTIVE"
	} else if customer.Status != "ACTIVE" && customer.Status != "INACTIVE" && customer.Status != "SUSPENDED" {
		return fmt.Errorf("%w: status must be ACTIVE, INACTIVE, or SUSPENDED", ErrCustomerInvalid)
	}
	return s.repository.Create(ctx, customer)
}

func (s *CustomerService) GetCustomerByID(ctx context.Context, id string) (*model.Customer, error) {
	return s.repository.FindByID(ctx, id)
}

func (s *CustomerService) ListCustomers(ctx context.Context) ([]model.Customer, error) {
	return s.repository.List(ctx)
}
