package model

import "time"

type Customer struct {
	ID             string    `json:"id"`
	CustomerNumber string    `json:"customer_number"`
	Name           string    `json:"name"`
	Email          string    `json:"email"`
	Phone          string    `json:"phone,omitempty"`
	Status         string    `json:"status"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}
