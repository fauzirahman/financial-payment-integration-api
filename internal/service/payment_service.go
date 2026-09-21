package service

import (
	"context"

	"github.com/fauzirahman/financial-payment-integration-api/internal/model"
	"github.com/fauzirahman/financial-payment-integration-api/internal/repository"
)

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

func (s *PaymentService) CreatePayment(ctx context.Context, payment *model.Payment) error {
	payment.Status = "PENDING"

	return s.repository.Create(ctx, payment)
}
