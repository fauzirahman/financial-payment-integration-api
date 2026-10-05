package service

import (
	"context"
	"errors"
	"testing"

	"github.com/fauzirahman/financial-payment-integration-api/internal/model"
	"github.com/fauzirahman/financial-payment-integration-api/internal/repository"
)

type paymentRepositoryStub struct {
	created                    *model.Payment
	idempotencyKey             *model.IdempotencyKey
	createWithIdempotencyCalls int
	createWithIdempotencyErr   error
	webhookCalls               int
}

func (r *paymentRepositoryStub) FindAll(context.Context) ([]model.Payment, error) {
	return nil, nil
}

func (r *paymentRepositoryStub) FindByID(context.Context, string) (*model.Payment, error) {
	return nil, nil
}

func (r *paymentRepositoryStub) FindByReference(_ context.Context, reference string) (*model.Payment, error) {
	if r.created != nil && r.created.Reference == reference {
		return r.created, nil
	}
	return nil, nil
}

func (r *paymentRepositoryStub) Create(_ context.Context, payment *model.Payment) error {
	copy := *payment
	r.created = &copy
	return nil
}

func (r *paymentRepositoryStub) CreateWithIdempotencyKey(_ context.Context, payment *model.Payment, key *model.IdempotencyKey) error {
	r.createWithIdempotencyCalls++
	if r.createWithIdempotencyErr != nil {
		return r.createWithIdempotencyErr
	}
	copy := *payment
	r.created = &copy
	r.idempotencyKey = key
	return nil
}

func (r *paymentRepositoryStub) FindIdempotencyKey(_ context.Context, key string) (*model.IdempotencyKey, error) {
	if r.idempotencyKey != nil && r.idempotencyKey.Key == key {
		return r.idempotencyKey, nil
	}
	return nil, nil
}

func (r *paymentRepositoryStub) ApplyWebhookEvent(_ context.Context, event model.WebhookEvent, status string) (bool, error) {
	r.webhookCalls++
	return true, nil
}

func TestCreatePaymentValidatesAndNormalizesPayment(t *testing.T) {
	repository := &paymentRepositoryStub{}
	service := NewPaymentService(repository)
	payment := &model.Payment{
		Reference: "  PAY-001 ",
		Amount:    150000,
		Currency:  " idr ",
		Status:    "SUCCESS",
	}

	if err := service.CreatePayment(context.Background(), payment); err != nil {
		t.Fatalf("CreatePayment() error = %v", err)
	}
	if payment.Reference != "PAY-001" || payment.Currency != "IDR" {
		t.Fatalf("payment was not normalized: %+v", payment)
	}
	if payment.Status != model.PaymentStatusPending {
		t.Fatalf("payment status = %q, want %q", payment.Status, model.PaymentStatusPending)
	}
	if repository.created == nil {
		t.Fatal("repository Create was not called")
	}
}

func TestCreatePaymentRejectsInvalidPayment(t *testing.T) {
	tests := []struct {
		name    string
		payment model.Payment
	}{
		{name: "missing reference", payment: model.Payment{Amount: 100, Currency: "IDR"}},
		{name: "non-positive amount", payment: model.Payment{Reference: "PAY-001", Amount: 0, Currency: "IDR"}},
		{name: "invalid currency", payment: model.Payment{Reference: "PAY-001", Amount: 100, Currency: "ID1"}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			repository := &paymentRepositoryStub{}
			service := NewPaymentService(repository)
			err := service.CreatePayment(context.Background(), &test.payment)
			if !errors.Is(err, ErrInvalidPayment) {
				t.Fatalf("CreatePayment() error = %v, want ErrInvalidPayment", err)
			}
			if repository.created != nil {
				t.Fatal("repository Create was called for invalid payment")
			}
		})
	}
}

func TestCreatePaymentWithIdempotencyKeyRejectsDuplicates(t *testing.T) {
	repository := &paymentRepositoryStub{}
	service := NewPaymentService(repository)
	payment := &model.Payment{Reference: "PAY-002", Amount: 1500, Currency: "IDR"}

	repository.idempotencyKey = &model.IdempotencyKey{Key: "idem-123", PaymentReference: "PAY-002"}

	if err := service.CreatePaymentWithIdempotency(context.Background(), payment, "idem-123"); !errors.Is(err, ErrDuplicateIdempotencyKey) {
		t.Fatalf("CreatePaymentWithIdempotency() error = %v, want ErrDuplicateIdempotencyKey", err)
	}
}

func TestCreatePaymentWithIdempotencyKeyUsesAtomicRepositoryOperation(t *testing.T) {
	repository := &paymentRepositoryStub{}
	service := NewPaymentService(repository)
	payment := &model.Payment{Reference: "PAY-ATOMIC", Amount: 2500, Currency: "IDR"}

	if err := service.CreatePaymentWithIdempotency(context.Background(), payment, "idem-atomic"); err != nil {
		t.Fatalf("CreatePaymentWithIdempotency() error = %v", err)
	}
	if repository.createWithIdempotencyCalls != 1 {
		t.Fatalf("atomic repository calls = %d, want 1", repository.createWithIdempotencyCalls)
	}
	if repository.created == nil || repository.idempotencyKey == nil {
		t.Fatal("atomic repository operation did not persist both payment and idempotency key")
	}
}

func TestCreatePaymentWithIdempotencyKeyDoesNotFallbackToPaymentOnlyOnError(t *testing.T) {
	repository := &paymentRepositoryStub{createWithIdempotencyErr: errors.New("key insert failed")}
	service := NewPaymentService(repository)
	payment := &model.Payment{Reference: "PAY-ATOMIC-FAIL", Amount: 2500, Currency: "IDR"}

	if err := service.CreatePaymentWithIdempotency(context.Background(), payment, "idem-atomic-fail"); err == nil {
		t.Fatal("CreatePaymentWithIdempotency() error = nil, want key insert failure")
	}
	if repository.created != nil {
		t.Fatal("payment-only repository operation was used after atomic operation failed")
	}
}

func TestCreatePaymentWithIdempotencyMapsConcurrentKeyConflict(t *testing.T) {
	repository := &paymentRepositoryStub{createWithIdempotencyErr: repository.ErrIdempotencyKeyExists}
	service := NewPaymentService(repository)
	payment := &model.Payment{Reference: "PAY-ATOMIC-RACE", Amount: 2500, Currency: "IDR"}

	err := service.CreatePaymentWithIdempotency(context.Background(), payment, "idem-race")
	if !errors.Is(err, ErrDuplicateIdempotencyKey) {
		t.Fatalf("CreatePaymentWithIdempotency() error = %v, want ErrDuplicateIdempotencyKey", err)
	}
}

func TestProcessWebhookEventAppliesSuccessOnce(t *testing.T) {
	repository := &paymentRepositoryStub{
		created: &model.Payment{Reference: "PAY-003", Amount: 2500, Currency: "IDR"},
	}
	service := NewPaymentService(repository)

	_, err := service.ProcessWebhookEvent(context.Background(), model.WebhookEvent{
		EventID:   "evt-1",
		Type:      "payment.success",
		Reference: "PAY-003",
	})
	if err != nil {
		t.Fatalf("ProcessWebhookEvent() error = %v", err)
	}
	if repository.webhookCalls != 1 {
		t.Fatalf("ApplyWebhookEvent() calls = %d, want 1", repository.webhookCalls)
	}
}
