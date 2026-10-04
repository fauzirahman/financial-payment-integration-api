package model

import "time"

const (
	ChartCategoryAsset    = "ASSET"
	ChartCategoryLiability = "LIABILITY"
	ChartCategoryEquity   = "EQUITY"
	ChartCategoryRevenue  = "REVENUE"
	ChartCategoryExpense  = "EXPENSE"
)

type ChartOfAccount struct {
	ID          string    `json:"id"`
	Code        string    `json:"code"`
	Name        string    `json:"name"`
	Category    string    `json:"category"`
	AccountType string    `json:"account_type"`
	IsActive    bool      `json:"is_active"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}
