package handler

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/fauzirahman/financial-payment-integration-api/internal/model"
	"github.com/fauzirahman/financial-payment-integration-api/internal/service"
	"github.com/jackc/pgx/v5"
)

type accountHandlerRepositoryStub struct {
	account *model.Account
	err     error
}

func (r *accountHandlerRepositoryStub) Create(context.Context, *model.Account) error {
	return nil
}

func (r *accountHandlerRepositoryStub) FindByID(context.Context, string) (*model.Account, error) {
	return r.account, r.err
}

func (r *accountHandlerRepositoryStub) FindByCustomerID(context.Context, string) ([]model.Account, error) {
	return nil, nil
}

func (r *accountHandlerRepositoryStub) FindByAccountNumber(context.Context, string) (*model.Account, error) {
	return nil, nil
}

func TestGetAccountByIDReturnsAccount(t *testing.T) {
	want := &model.Account{ID: "550e8400-e29b-41d4-a716-446655440000", AccountNumber: "ACC-001"}
	handler := NewAccountHandler(service.NewAccountService(&accountHandlerRepositoryStub{account: want}))
	request := httptest.NewRequest(http.MethodGet, "/api/v1/accounts/"+want.ID, nil)
	response := httptest.NewRecorder()

	handler.GetAccountByID(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", response.Code, http.StatusOK, response.Body)
	}
}

func TestGetAccountByIDRejectsInvalidUUID(t *testing.T) {
	handler := NewAccountHandler(service.NewAccountService(&accountHandlerRepositoryStub{}))
	request := httptest.NewRequest(http.MethodGet, "/api/v1/accounts/not-a-uuid", nil)
	response := httptest.NewRecorder()

	handler.GetAccountByID(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusBadRequest)
	}
}

func TestGetAccountByIDReturnsNotFound(t *testing.T) {
	handler := NewAccountHandler(service.NewAccountService(&accountHandlerRepositoryStub{err: pgx.ErrNoRows}))
	request := httptest.NewRequest(http.MethodGet, "/api/v1/accounts/550e8400-e29b-41d4-a716-446655440000", nil)
	response := httptest.NewRecorder()

	handler.GetAccountByID(response, request)

	if response.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusNotFound)
	}
}

func TestGetAccountByIDReturnsInternalError(t *testing.T) {
	handler := NewAccountHandler(service.NewAccountService(&accountHandlerRepositoryStub{err: errors.New("database unavailable")}))
	request := httptest.NewRequest(http.MethodGet, "/api/v1/accounts/550e8400-e29b-41d4-a716-446655440000", nil)
	response := httptest.NewRecorder()

	handler.GetAccountByID(response, request)

	if response.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusInternalServerError)
	}
}
