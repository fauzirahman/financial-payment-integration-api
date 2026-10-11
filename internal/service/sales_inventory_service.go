package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"regexp"
	"strings"

	"github.com/fauzirahman/financial-payment-integration-api/internal/model"
	"github.com/fauzirahman/financial-payment-integration-api/internal/repository"
)

var (
	ErrInvalidSalesOrder      = errors.New("invalid sales order")
	ErrInvalidInventory       = errors.New("invalid inventory request")
	salesInventoryUUIDPattern = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)
)

type SalesInventoryService struct {
	repository repository.SalesInventoryRepository
}

func NewSalesInventoryService(repository repository.SalesInventoryRepository) *SalesInventoryService {
	return &SalesInventoryService{repository: repository}
}

func (s *SalesInventoryService) CreateSalesOrder(ctx context.Context, request *model.CreateSalesOrderRequest) (*model.SalesOrder, error) {
	if request == nil {
		return nil, fmt.Errorf("%w: request is required", ErrInvalidSalesOrder)
	}
	request.CustomerID = strings.TrimSpace(request.CustomerID)
	request.Currency = strings.ToUpper(strings.TrimSpace(request.Currency))
	request.PaymentTerms = strings.TrimSpace(request.PaymentTerms)
	if !salesInventoryUUIDPattern.MatchString(request.CustomerID) {
		return nil, fmt.Errorf("%w: customer_id must be a UUID", ErrInvalidSalesOrder)
	}
	if !currencyPattern.MatchString(request.Currency) {
		return nil, fmt.Errorf("%w: currency must be a 3-letter code", ErrInvalidSalesOrder)
	}
	if len(request.DeliveryAddress) > 0 && !isJSONObject(request.DeliveryAddress) {
		return nil, fmt.Errorf("%w: delivery_address must be a JSON object", ErrInvalidSalesOrder)
	}
	if len(request.Items) == 0 || len(request.Items) > 100 {
		return nil, fmt.Errorf("%w: order must contain 1-100 items", ErrInvalidSalesOrder)
	}
	order := &model.SalesOrder{
		CustomerID:      request.CustomerID,
		Currency:        request.Currency,
		PaymentTerms:    request.PaymentTerms,
		DeliveryAddress: request.DeliveryAddress,
		Status:          "DRAFT",
		Items:           make([]model.SalesOrderItem, len(request.Items)),
	}
	for index, item := range request.Items {
		item.ProductID = strings.TrimSpace(item.ProductID)
		item.SKU = strings.TrimSpace(item.SKU)
		item.Description = strings.TrimSpace(item.Description)
		item.Unit = strings.TrimSpace(item.Unit)
		if !salesInventoryUUIDPattern.MatchString(item.ProductID) || item.Description == "" || item.Unit == "" ||
			!finite(item.Quantity) || item.Quantity <= 0 || !hasPrecision(item.Quantity, 4) ||
			!finite(item.UnitPrice) || item.UnitPrice < 0 || !hasPrecision(item.UnitPrice, 2) ||
			!finite(item.TaxRate) || item.TaxRate < 0 || item.TaxRate > 100 || !hasPrecision(item.TaxRate, 4) {
			return nil, fmt.Errorf("%w: item %d has invalid product, quantity, price, or tax rate", ErrInvalidSalesOrder, index+1)
		}
		item.LineTotal = roundMoney(item.Quantity * item.UnitPrice)
		order.Subtotal += item.LineTotal
		order.TaxTotal += roundMoney(item.LineTotal * item.TaxRate / 100)
		order.Items[index] = item
	}
	order.Subtotal = roundMoney(order.Subtotal)
	order.TaxTotal = roundMoney(order.TaxTotal)
	order.Total = roundMoney(order.Subtotal + order.TaxTotal)
	if err := s.repository.CreateSalesOrder(ctx, order); err != nil {
		return nil, err
	}
	return order, nil
}

func (s *SalesInventoryService) ListSalesOrders(ctx context.Context) ([]model.SalesOrder, error) {
	return s.repository.ListSalesOrders(ctx)
}

func (s *SalesInventoryService) GetSalesOrder(ctx context.Context, id string) (*model.SalesOrder, error) {
	if !salesInventoryUUIDPattern.MatchString(id) {
		return nil, fmt.Errorf("%w: order id must be a UUID", ErrInvalidSalesOrder)
	}
	return s.repository.GetSalesOrder(ctx, id)
}

func (s *SalesInventoryService) ListSalesActivity(ctx context.Context, id string) ([]model.SalesActivity, error) {
	if !salesInventoryUUIDPattern.MatchString(id) {
		return nil, fmt.Errorf("%w: order id must be a UUID", ErrInvalidSalesOrder)
	}
	return s.repository.ListSalesActivity(ctx, id)
}

func (s *SalesInventoryService) ListDeliveries(ctx context.Context, id string) ([]model.DeliveryOrder, error) {
	if !salesInventoryUUIDPattern.MatchString(id) {
		return nil, fmt.Errorf("%w: order id must be a UUID", ErrInvalidSalesOrder)
	}
	return s.repository.ListDeliveries(ctx, id)
}

func (s *SalesInventoryService) ConfirmSalesOrder(ctx context.Context, id, warehouseID string) error {
	if !salesInventoryUUIDPattern.MatchString(id) || !salesInventoryUUIDPattern.MatchString(warehouseID) {
		return fmt.Errorf("%w: order and warehouse ids must be UUIDs", ErrInvalidSalesOrder)
	}
	return s.repository.ConfirmSalesOrder(ctx, id, warehouseID)
}

func (s *SalesInventoryService) DeliverSalesOrder(ctx context.Context, id, warehouseID string) (*model.DeliveryOrder, error) {
	if !salesInventoryUUIDPattern.MatchString(id) || !salesInventoryUUIDPattern.MatchString(warehouseID) {
		return nil, fmt.Errorf("%w: order and warehouse ids must be UUIDs", ErrInvalidSalesOrder)
	}
	return s.repository.DeliverSalesOrder(ctx, id, warehouseID)
}

func (s *SalesInventoryService) ListInventory(ctx context.Context) ([]model.InventoryBalance, error) {
	return s.repository.ListInventory(ctx)
}

func (s *SalesInventoryService) ListInventoryMovements(ctx context.Context, productID, warehouseID string, limit int) ([]model.InventoryMovement, error) {
	productID = strings.TrimSpace(productID)
	warehouseID = strings.TrimSpace(warehouseID)
	if (productID != "" && !salesInventoryUUIDPattern.MatchString(productID)) ||
		(warehouseID != "" && !salesInventoryUUIDPattern.MatchString(warehouseID)) || limit < 1 || limit > 200 {
		return nil, fmt.Errorf("%w: invalid movement filters", ErrInvalidInventory)
	}
	return s.repository.ListInventoryMovements(ctx, productID, warehouseID, limit)
}

func (s *SalesInventoryService) RecordInventoryMovement(ctx context.Context, movementType string, request *model.InventoryMovementRequest) error {
	if request == nil {
		return fmt.Errorf("%w: request is required", ErrInvalidInventory)
	}
	request.ProductID = strings.TrimSpace(request.ProductID)
	request.WarehouseID = strings.TrimSpace(request.WarehouseID)
	request.Reason = strings.TrimSpace(request.Reason)
	if !salesInventoryUUIDPattern.MatchString(request.ProductID) || !salesInventoryUUIDPattern.MatchString(request.WarehouseID) ||
		!finite(request.QuantityDelta) || request.QuantityDelta == 0 || !hasPrecision(request.QuantityDelta, 4) || request.Reason == "" {
		return fmt.Errorf("%w: valid product, warehouse, nonzero quantity_delta, and reason are required", ErrInvalidInventory)
	}
	movementType = strings.ToUpper(strings.TrimSpace(movementType))
	if movementType != "RECEIPT" && movementType != "ADJUSTMENT" {
		return fmt.Errorf("%w: movement type must be RECEIPT or ADJUSTMENT", ErrInvalidInventory)
	}
	if movementType == "RECEIPT" && request.QuantityDelta < 0 {
		return fmt.Errorf("%w: receipts must increase stock", ErrInvalidInventory)
	}
	return s.repository.RecordInventoryMovement(ctx, movementType, request)
}

func (s *SalesInventoryService) CreateProduct(ctx context.Context, product *model.Product) error {
	if product == nil {
		return fmt.Errorf("%w: product is required", ErrInvalidInventory)
	}
	product.SKU = strings.TrimSpace(product.SKU)
	product.Name = strings.TrimSpace(product.Name)
	product.Unit = strings.TrimSpace(product.Unit)
	if product.SKU == "" || len(product.SKU) > 60 || product.Name == "" || len(product.Name) > 200 || product.Unit == "" || len(product.Unit) > 20 {
		return fmt.Errorf("%w: SKU, name, and unit are required", ErrInvalidInventory)
	}
	product.IsStocked = true
	product.IsActive = true
	return s.repository.CreateProduct(ctx, product)
}

func (s *SalesInventoryService) ListProducts(ctx context.Context) ([]model.Product, error) {
	return s.repository.ListProducts(ctx)
}

func (s *SalesInventoryService) CreateWarehouse(ctx context.Context, warehouse *model.Warehouse) error {
	if warehouse == nil {
		return fmt.Errorf("%w: warehouse is required", ErrInvalidInventory)
	}
	warehouse.Code = strings.TrimSpace(warehouse.Code)
	warehouse.Name = strings.TrimSpace(warehouse.Name)
	if warehouse.Code == "" || len(warehouse.Code) > 30 || warehouse.Name == "" || len(warehouse.Name) > 150 {
		return fmt.Errorf("%w: warehouse code and name are required", ErrInvalidInventory)
	}
	if len(warehouse.Address) > 0 && !isJSONObject(warehouse.Address) {
		return fmt.Errorf("%w: address must be a JSON object", ErrInvalidInventory)
	}
	warehouse.IsActive = true
	return s.repository.CreateWarehouse(ctx, warehouse)
}

func (s *SalesInventoryService) ListWarehouses(ctx context.Context) ([]model.Warehouse, error) {
	return s.repository.ListWarehouses(ctx)
}

func finite(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0)
}

func roundMoney(value float64) float64 {
	return math.Round(value*100) / 100
}

func hasPrecision(value float64, decimals int) bool {
	scale := math.Pow10(decimals)
	return math.Abs(value*scale-math.Round(value*scale)) < 1e-8
}

func isJSONObject(value []byte) bool {
	var object map[string]json.RawMessage
	return json.Unmarshal(value, &object) == nil && object != nil
}
