package model

import (
	"encoding/json"
	"time"
)

type SalesOrderItem struct {
	ID          string  `json:"id,omitempty"`
	ProductID   string  `json:"product_id"`
	SKU         string  `json:"sku,omitempty"`
	Description string  `json:"description"`
	Quantity    float64 `json:"quantity"`
	Unit        string  `json:"unit"`
	UnitPrice   float64 `json:"unit_price"`
	TaxRate     float64 `json:"tax_rate"`
	LineTotal   float64 `json:"line_total,omitempty"`
}

type SalesOrder struct {
	ID              string           `json:"id"`
	OrderNumber     string           `json:"order_number"`
	CustomerID      string           `json:"customer_id"`
	QuotationID     string           `json:"quotation_id,omitempty"`
	Status          string           `json:"status"`
	Currency        string           `json:"currency"`
	Subtotal        float64          `json:"subtotal"`
	TaxTotal        float64          `json:"tax_total"`
	Total           float64          `json:"total"`
	PaymentTerms    string           `json:"payment_terms,omitempty"`
	DeliveryAddress json.RawMessage  `json:"delivery_address,omitempty"`
	Items           []SalesOrderItem `json:"items"`
	CreatedAt       time.Time        `json:"created_at"`
	UpdatedAt       time.Time        `json:"updated_at"`
}

type CreateSalesOrderRequest struct {
	CustomerID      string           `json:"customer_id"`
	Currency        string           `json:"currency"`
	PaymentTerms    string           `json:"payment_terms,omitempty"`
	DeliveryAddress json.RawMessage  `json:"delivery_address,omitempty"`
	Items           []SalesOrderItem `json:"items"`
}

type ConfirmSalesOrderRequest struct {
	WarehouseID string `json:"warehouse_id"`
}

type DeliveryOrder struct {
	ID             string              `json:"id"`
	DeliveryNumber string              `json:"delivery_number"`
	SalesOrderID   string              `json:"sales_order_id"`
	WarehouseID    string              `json:"warehouse_id"`
	Status         string              `json:"status"`
	ProcessedAt    time.Time           `json:"processed_at"`
	Items          []DeliveryOrderItem `json:"items,omitempty"`
}

type DeliveryOrderItem struct {
	SalesOrderItemID string  `json:"sales_order_item_id"`
	SKU              string  `json:"sku,omitempty"`
	Description      string  `json:"description"`
	Quantity         float64 `json:"quantity"`
}

type SalesActivity struct {
	ID         string          `json:"id"`
	EntityType string          `json:"entity_type"`
	EntityID   string          `json:"entity_id"`
	Action     string          `json:"action"`
	ActorName  string          `json:"actor_name,omitempty"`
	OldValues  json.RawMessage `json:"old_values,omitempty"`
	NewValues  json.RawMessage `json:"new_values,omitempty"`
	OccurredAt time.Time       `json:"occurred_at"`
}

type InventoryBalance struct {
	ProductID   string  `json:"product_id"`
	SKU         string  `json:"sku"`
	ProductName string  `json:"product_name"`
	WarehouseID string  `json:"warehouse_id"`
	Warehouse   string  `json:"warehouse"`
	OnHand      float64 `json:"on_hand"`
	Reserved    float64 `json:"reserved"`
	Available   float64 `json:"available"`
}

type InventoryMovement struct {
	ID            string    `json:"id"`
	ProductID     string    `json:"product_id"`
	SKU           string    `json:"sku"`
	ProductName   string    `json:"product_name"`
	WarehouseID   string    `json:"warehouse_id"`
	Warehouse     string    `json:"warehouse"`
	MovementType  string    `json:"movement_type"`
	QuantityDelta float64   `json:"quantity_delta"`
	ReferenceType string    `json:"reference_type"`
	ReferenceID   string    `json:"reference_id"`
	OccurredAt    time.Time `json:"occurred_at"`
}

type InventoryMovementRequest struct {
	ProductID     string  `json:"product_id"`
	WarehouseID   string  `json:"warehouse_id"`
	QuantityDelta float64 `json:"quantity_delta"`
	Reason        string  `json:"reason"`
}

type Product struct {
	ID          string    `json:"id,omitempty"`
	SKU         string    `json:"sku"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	Unit        string    `json:"unit"`
	IsStocked   bool      `json:"is_stocked"`
	IsActive    bool      `json:"is_active"`
	CreatedAt   time.Time `json:"created_at,omitempty"`
}

type Warehouse struct {
	ID        string          `json:"id,omitempty"`
	Code      string          `json:"code"`
	Name      string          `json:"name"`
	Address   json.RawMessage `json:"address,omitempty"`
	IsActive  bool            `json:"is_active"`
	CreatedAt time.Time       `json:"created_at,omitempty"`
}
