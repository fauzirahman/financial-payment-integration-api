package repository

import "github.com/fauzirahman/financial-payment-integration-api/internal/model"

type PaymentRepository interface {
	FindAll() []model.Payment
}

type InMemoryPaymentRepository struct {
	payments []model.Payment
}

func NewInMemoryPaymentRepository() *InMemoryPaymentRepository {
	return &InMemoryPaymentRepository{
		payments: []model.Payment{
			{
				ID:        1,
				Reference: "PAY-001",
				Amount:    150000,
				Currency:  "IDR",
				Status:    "PENDING",
			},
			{
				ID:        2,
				Reference: "PAY-002",
				Amount:    250000,
				Currency:  "IDR",
				Status:    "SUCCESS",
			},
		},
	}
}

func (r *InMemoryPaymentRepository) FindAll() []model.Payment {
	return r.payments
}