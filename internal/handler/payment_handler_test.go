package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/fauzirahman/financial-payment-integration-api/internal/model"
	"github.com/fauzirahman/financial-payment-integration-api/internal/service"
	"github.com/jackc/pgx/v5"
)

type paymentRepositoryStub struct {
	payment *model.Payment
}

func (r *paymentRepositoryStub) FindAll(context.Context) ([]model.Payment, error) {
	return []model.Payment{}, nil
}

func (r *paymentRepositoryStub) FindByID(_ context.Context, id int64) (*model.Payment, error) {
	if r.payment == nil || r.payment.ID != id {
		return nil, pgx.ErrNoRows
	}
	return r.payment, nil
}

func (r *paymentRepositoryStub) FindByReference(_ context.Context, reference string) (*model.Payment, error) {
	if r.payment == nil || r.payment.Reference != reference {
		return nil, pgx.ErrNoRows
	}
	return r.payment, nil
}

func (r *paymentRepositoryStub) Create(_ context.Context, payment *model.Payment) error {
	payment.ID = 1
	r.payment = payment
	return nil
}

func (r *paymentRepositoryStub) CreateWithIdempotencyKey(_ context.Context, payment *model.Payment, _ *model.IdempotencyKey) error {
	payment.ID = 1
	r.payment = payment
	return nil
}

func (r *paymentRepositoryStub) FindIdempotencyKey(context.Context, string) (*model.IdempotencyKey, error) {
	return nil, nil
}

func (r *paymentRepositoryStub) ApplyWebhookEvent(context.Context, model.WebhookEvent, string) (bool, error) {
	return true, nil
}

func TestCreatePaymentReturnsCreatedPayment(t *testing.T) {
	service := service.NewPaymentService(&paymentRepositoryStub{})
	handler := NewPaymentHandler(service)
	request := httptest.NewRequest(http.MethodPost, "/api/v1/payments", strings.NewReader(
		`{"reference":"PAY-001","amount":150000,"currency":"idr"}`,
	))
	response := httptest.NewRecorder()

	handler.CreatePayment(response, request)

	if response.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d; body: %s", response.Code, http.StatusCreated, response.Body)
	}
	var payment model.Payment
	if err := json.NewDecoder(response.Body).Decode(&payment); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payment.Reference != "PAY-001" || payment.Currency != "IDR" || payment.Status != model.PaymentStatusPending {
		t.Fatalf("unexpected payment response: %+v", payment)
	}
}

func TestGetPaymentByIDReturnsNotFound(t *testing.T) {
	service := service.NewPaymentService(&paymentRepositoryStub{})
	handler := NewPaymentHandler(service)
	request := httptest.NewRequest(http.MethodGet, "/api/v1/payments/99", nil)
	response := httptest.NewRecorder()

	handler.GetPaymentByID(response, request)

	if response.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusNotFound)
	}
}
