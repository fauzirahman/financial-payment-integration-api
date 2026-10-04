package model

import "time"

type FinancialTransaction struct {
	ID              string    `json:"id"`
	PaymentID       string    `json:"payment_id,omitempty"`
	AccountID       string    `json:"account_id"`
	TransactionType string    `json:"transaction_type"`
	Direction       string    `json:"direction"`
	Amount          int64     `json:"amount"`
	Currency        string    `json:"currency"`
	Reference       string    `json:"reference"`
	Status          string    `json:"status"`
	ProcessedAt     time.Time `json:"processed_at,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
}
