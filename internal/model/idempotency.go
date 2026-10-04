package model

import "time"

// IdempotencyKey stores a client-provided request key and the associated
// payment reference so duplicate requests can be safely rejected.
type IdempotencyKey struct {
	Key             string    `json:"key"`
	PaymentReference string   `json:"payment_reference"`
	RequestHash     string    `json:"request_hash"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}
