package model

import "time"

const (
	LedgerEntryTypeDebit  = "DEBIT"
	LedgerEntryTypeCredit = "CREDIT"
)

type LedgerEntry struct {
	ID               int64     `json:"id"`
	JournalID        int64     `json:"journal_id"`
	PaymentReference string    `json:"payment_reference"`
	AccountCode      string    `json:"account_code"`
	EntryType        string    `json:"entry_type"`
	Amount           int64     `json:"amount"`
	Currency         string    `json:"currency"`
	PostedAt         time.Time `json:"posted_at"`
	CreatedAt        time.Time `json:"created_at"`
}
