package model

import "time"

type PaymentSummary struct {
	Currency           string `json:"currency"`
	TotalPayments      int64  `json:"total_payments"`
	PendingPayments    int64  `json:"pending_payments"`
	SuccessfulPayments int64  `json:"successful_payments"`
	FailedPayments     int64  `json:"failed_payments"`
	TotalAmount        int64  `json:"total_amount"`
	SuccessfulAmount   int64  `json:"successful_amount"`
}

type DailyPaymentReport struct {
	Date               string `json:"date"`
	Currency           string `json:"currency"`
	TotalPayments      int64  `json:"total_payments"`
	SuccessfulPayments int64  `json:"successful_payments"`
	FailedPayments     int64  `json:"failed_payments"`
	TotalAmount        int64  `json:"total_amount"`
	SuccessfulAmount   int64  `json:"successful_amount"`
}

type GeneralLedgerEntry struct {
	FinancialTransactionID string    `json:"financial_transaction_id"`
	PaymentNumber          string    `json:"payment_number"`
	AccountCode            string    `json:"account_code"`
	EntryType              string    `json:"entry_type"`
	DebitAmount            float64   `json:"debit_amount"`
	CreditAmount           float64   `json:"credit_amount"`
	RunningBalance         float64   `json:"running_balance"`
	Currency               string    `json:"currency"`
	PostedAt               time.Time `json:"posted_at"`
}
