package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/fauzirahman/financial-payment-integration-api/internal/model"
	"github.com/fauzirahman/financial-payment-integration-api/internal/repository"
	"github.com/jackc/pgx/v5"
)

var ErrInvalidPayment = errors.New("invalid payment")
var ErrInvalidWebhookEvent = errors.New("invalid webhook event")
var ErrDuplicateIdempotencyKey = errors.New("duplicate idempotency key")

type PaymentService struct {
	repository repository.PaymentRepository
}

func NewPaymentService(repository repository.PaymentRepository) *PaymentService {
	return &PaymentService{
		repository: repository,
	}
}

func (s *PaymentService) GetAllPayments(ctx context.Context) ([]model.Payment, error) {
	return s.repository.FindAll(ctx)
}

func (s *PaymentService) GetPaymentByID(ctx context.Context, id string) (*model.Payment, error) {
	return s.repository.FindByID(ctx, id)
}

func (s *PaymentService) CreatePayment(ctx context.Context, payment *model.Payment) error {
	return s.CreatePaymentWithIdempotency(ctx, payment, "")
}

func (s *PaymentService) CreatePaymentWithIdempotency(ctx context.Context, payment *model.Payment, idempotencyKey string) error {
	payment.Reference = strings.TrimSpace(payment.Reference)
	if payment.Reference == "" {
		return fmt.Errorf("%w: reference is required", ErrInvalidPayment)
	}
	if payment.Amount <= 0 {
		return fmt.Errorf("%w: amount must be greater than 0", ErrInvalidPayment)
	}

	payment.Currency = strings.ToUpper(strings.TrimSpace(payment.Currency))
	if len(payment.Currency) != 3 {
		return fmt.Errorf("%w: currency must be a 3-letter code", ErrInvalidPayment)
	}
	for _, character := range payment.Currency {
		if character < 'A' || character > 'Z' {
			return fmt.Errorf("%w: currency must be a 3-letter code", ErrInvalidPayment)
		}
	}

	idempotencyKey = strings.TrimSpace(idempotencyKey)
	if idempotencyKey != "" {
		existing, err := s.repository.FindIdempotencyKey(ctx, idempotencyKey)
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return err
		}
		if existing != nil {
			if existing.PaymentReference != payment.Reference {
				return fmt.Errorf("%w: idempotency key already used for a different payment", ErrDuplicateIdempotencyKey)
			}
			return fmt.Errorf("%w: request already processed", ErrDuplicateIdempotencyKey)
		}
	}

	payment.ID = ""
	payment.Status = model.PaymentStatusPending

	if idempotencyKey == "" {
		return s.repository.Create(ctx, payment)
	}

	err := s.repository.CreateWithIdempotencyKey(ctx, payment, &model.IdempotencyKey{
		Key:              idempotencyKey,
		PaymentReference: payment.Reference,
		RequestHash:      fmt.Sprintf("payment:%s:%d:%s", payment.Reference, payment.Amount, payment.Currency),
	})
	if errors.Is(err, repository.ErrIdempotencyKeyExists) {
		return fmt.Errorf("%w: idempotency key already used", ErrDuplicateIdempotencyKey)
	}
	return err
}

func (s *PaymentService) ProcessWebhookEvent(ctx context.Context, event model.WebhookEvent) (bool, error) {
	event.EventID = strings.TrimSpace(event.EventID)
	event.Reference = strings.TrimSpace(event.Reference)
	event.Type = strings.TrimSpace(event.Type)
	if event.EventID == "" || event.Reference == "" {
		return false, ErrInvalidWebhookEvent
	}

	var status string
	switch event.Type {
	case "payment.success":
		status = model.PaymentStatusSuccess
	case "payment.failed":
		status = model.PaymentStatusFailed
	default:
		return false, fmt.Errorf("%w: unsupported event type", ErrInvalidWebhookEvent)
	}

	applied, err := s.repository.ApplyWebhookEvent(ctx, event, status)
	return applied, err
}
