package model

import (
	"errors"
	"time"
)

const PaymentStatusPending = "PENDING"
const PaymentStatusProcessing = "PROCESSING"
const PaymentStatusSuccess = "SUCCESS"
const PaymentStatusFailed = "FAILED"

var ErrInvalidPaymentTransition = errors.New("invalid payment status transition")

type Payment struct {
	ID        string    `json:"id"`
	Reference string    `json:"reference"`
	Amount    int64     `json:"amount"`
	Currency  string    `json:"currency"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
