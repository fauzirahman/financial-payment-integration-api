package model

import "time"

type Payment struct {
	ID        int64     `json:"id"`
	Reference string    `json:"reference"`
	Amount    int64     `json:"amount"`
	Currency  string    `json:"currency"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
