package service

import (
	"context"
	"errors"
	"testing"

	"github.com/fauzirahman/financial-payment-integration-api/internal/model"
)

type customerRepositoryStub struct {
	created *model.Customer
}

func (r *customerRepositoryStub) Create(_ context.Context, customer *model.Customer) error {
	r.created = customer
	return nil
}

func (r *customerRepositoryStub) FindByID(context.Context, string) (*model.Customer, error) {
	return nil, nil
}

func (r *customerRepositoryStub) FindByEmail(context.Context, string) (*model.Customer, error) {
	return nil, nil
}

func (r *customerRepositoryStub) List(context.Context) ([]model.Customer, error) {
	return nil, nil
}

type accountRepositoryStub struct {
	created *model.Account
}

func (r *accountRepositoryStub) Create(_ context.Context, account *model.Account) error {
	r.created = account
	return nil
}

func (r *accountRepositoryStub) FindByID(context.Context, string) (*model.Account, error) {
	return nil, nil
}

func (r *accountRepositoryStub) FindByCustomerID(context.Context, string) ([]model.Account, error) {
	return nil, nil
}

func (r *accountRepositoryStub) FindByAccountNumber(context.Context, string) (*model.Account, error) {
	return nil, nil
}

func TestCreateCustomerValidatesAndNormalizes(t *testing.T) {
	repository := &customerRepositoryStub{}
	service := NewCustomerService(repository)
	customer := &model.Customer{
		CustomerNumber: " CUST_001 ",
		Name:           " Ada Lovelace ",
		Email:          " ADA@example.com ",
		Phone:          "+62 812-3456",
	}

	if err := service.CreateCustomer(context.Background(), customer); err != nil {
		t.Fatalf("CreateCustomer() error = %v", err)
	}
	if customer.CustomerNumber != "CUST_001" || customer.Name != "Ada Lovelace" || customer.Email != "ada@example.com" || customer.Status != "ACTIVE" {
		t.Fatalf("customer was not normalized/defaulted: %+v", customer)
	}
	if repository.created != customer {
		t.Fatal("repository Create was not called")
	}
}

func TestCreateCustomerRejectsInvalidValues(t *testing.T) {
	tests := []struct {
		name     string
		customer model.Customer
	}{
		{name: "invalid customer number", customer: model.Customer{CustomerNumber: "bad number", Name: "Ada", Email: "ada@example.com"}},
		{name: "invalid email", customer: model.Customer{CustomerNumber: "CUST-1", Name: "Ada", Email: "not-an-email"}},
		{name: "invalid phone", customer: model.Customer{CustomerNumber: "CUST-1", Name: "Ada", Email: "ada@example.com", Phone: "+- ()"}},
		{name: "invalid status", customer: model.Customer{CustomerNumber: "CUST-1", Name: "Ada", Email: "ada@example.com", Status: "CLOSED"}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			repository := &customerRepositoryStub{}
			service := NewCustomerService(repository)
			if err := service.CreateCustomer(context.Background(), &test.customer); !errors.Is(err, ErrCustomerInvalid) {
				t.Fatalf("CreateCustomer() error = %v, want ErrCustomerInvalid", err)
			}
			if repository.created != nil {
				t.Fatal("repository Create was called for invalid customer")
			}
		})
	}
}

func TestCreateAccountValidatesAndNormalizes(t *testing.T) {
	repository := &accountRepositoryStub{}
	service := NewAccountService(repository)
	account := &model.Account{
		CustomerID:    "  550e8400-e29b-41d4-a716-446655440000 ",
		AccountNumber: " ACC_001 ",
		Currency:      " idr ",
	}

	if err := service.CreateAccount(context.Background(), account); err != nil {
		t.Fatalf("CreateAccount() error = %v", err)
	}
	if account.AccountNumber != "ACC_001" || account.Currency != "IDR" || account.Status != "ACTIVE" {
		t.Fatalf("account was not normalized/defaulted: %+v", account)
	}
	if repository.created != account {
		t.Fatal("repository Create was not called")
	}
}

func TestCreateAccountRejectsInvalidValues(t *testing.T) {
	tests := []struct {
		name    string
		account model.Account
	}{
		{name: "invalid customer id", account: model.Account{CustomerID: "not-a-uuid", AccountNumber: "ACC-1", Currency: "IDR"}},
		{name: "invalid account number", account: model.Account{CustomerID: "550e8400-e29b-41d4-a716-446655440000", AccountNumber: "bad number", Currency: "IDR"}},
		{name: "invalid currency", account: model.Account{CustomerID: "550e8400-e29b-41d4-a716-446655440000", AccountNumber: "ACC-1", Currency: "US1"}},
		{name: "negative balance", account: model.Account{CustomerID: "550e8400-e29b-41d4-a716-446655440000", AccountNumber: "ACC-1", Currency: "IDR", Balance: -1}},
		{name: "invalid status", account: model.Account{CustomerID: "550e8400-e29b-41d4-a716-446655440000", AccountNumber: "ACC-1", Currency: "IDR", Status: "CLOSED"}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			repository := &accountRepositoryStub{}
			service := NewAccountService(repository)
			if err := service.CreateAccount(context.Background(), &test.account); !errors.Is(err, ErrAccountInvalid) {
				t.Fatalf("CreateAccount() error = %v, want ErrAccountInvalid", err)
			}
			if repository.created != nil {
				t.Fatal("repository Create was called for invalid account")
			}
		})
	}
}
