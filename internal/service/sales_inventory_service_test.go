package service

import (
	"context"
	"errors"
	"testing"

	"github.com/fauzirahman/financial-payment-integration-api/internal/model"
)

type salesInventoryRepositoryStub struct {
	createdOrder *model.SalesOrder
	movementType string
	movement     *model.InventoryMovementRequest
}

func (r *salesInventoryRepositoryStub) CreateSalesOrder(_ context.Context, order *model.SalesOrder) error {
	r.createdOrder = order
	return nil
}
func (*salesInventoryRepositoryStub) ListSalesOrders(context.Context) ([]model.SalesOrder, error) {
	return nil, nil
}
func (*salesInventoryRepositoryStub) GetSalesOrder(context.Context, string) (*model.SalesOrder, error) {
	return nil, nil
}
func (*salesInventoryRepositoryStub) ListSalesActivity(context.Context, string) ([]model.SalesActivity, error) {
	return nil, nil
}
func (*salesInventoryRepositoryStub) ListDeliveries(context.Context, string) ([]model.DeliveryOrder, error) {
	return nil, nil
}
func (*salesInventoryRepositoryStub) ConfirmSalesOrder(context.Context, string, string) error {
	return nil
}
func (*salesInventoryRepositoryStub) DeliverSalesOrder(context.Context, string, string) (*model.DeliveryOrder, error) {
	return nil, nil
}
func (*salesInventoryRepositoryStub) ListInventory(context.Context) ([]model.InventoryBalance, error) {
	return nil, nil
}
func (*salesInventoryRepositoryStub) ListInventoryMovements(context.Context, string, string, int) ([]model.InventoryMovement, error) {
	return nil, nil
}
func (r *salesInventoryRepositoryStub) RecordInventoryMovement(_ context.Context, movementType string, request *model.InventoryMovementRequest) error {
	r.movementType = movementType
	r.movement = request
	return nil
}
func (*salesInventoryRepositoryStub) CreateProduct(context.Context, *model.Product) error { return nil }
func (*salesInventoryRepositoryStub) ListProducts(context.Context) ([]model.Product, error) {
	return nil, nil
}
func (*salesInventoryRepositoryStub) CreateWarehouse(context.Context, *model.Warehouse) error {
	return nil
}
func (*salesInventoryRepositoryStub) ListWarehouses(context.Context) ([]model.Warehouse, error) {
	return nil, nil
}

func TestCreateSalesOrderCalculatesTotals(t *testing.T) {
	repository := &salesInventoryRepositoryStub{}
	service := NewSalesInventoryService(repository)
	request := &model.CreateSalesOrderRequest{
		CustomerID: "550e8400-e29b-41d4-a716-446655440000",
		Currency:   "idr",
		Items: []model.SalesOrderItem{{
			ProductID:   "550e8400-e29b-41d4-a716-446655440001",
			Description: "Industrial pump",
			Unit:        "UNIT",
			Quantity:    2,
			UnitPrice:   125.50,
			TaxRate:     11,
		}},
	}
	order, err := service.CreateSalesOrder(context.Background(), request)
	if err != nil {
		t.Fatalf("CreateSalesOrder() error = %v", err)
	}
	if order.Status != "DRAFT" || order.Currency != "IDR" || order.Subtotal != 251 || order.TaxTotal != 27.61 || order.Total != 278.61 {
		t.Fatalf("order totals/status = %+v", order)
	}
	if repository.createdOrder != order {
		t.Fatal("order was not passed to repository")
	}
}

func TestCreateSalesOrderRejectsInvalidItems(t *testing.T) {
	service := NewSalesInventoryService(&salesInventoryRepositoryStub{})
	tests := []struct {
		name string
		item model.SalesOrderItem
	}{
		{name: "zero quantity", item: model.SalesOrderItem{ProductID: "550e8400-e29b-41d4-a716-446655440001", Description: "Pump", Unit: "UNIT", Quantity: 0, UnitPrice: 1}},
		{name: "negative price", item: model.SalesOrderItem{ProductID: "550e8400-e29b-41d4-a716-446655440001", Description: "Pump", Unit: "UNIT", Quantity: 1, UnitPrice: -1}},
		{name: "tax over 100 percent", item: model.SalesOrderItem{ProductID: "550e8400-e29b-41d4-a716-446655440001", Description: "Pump", Unit: "UNIT", Quantity: 1, UnitPrice: 1, TaxRate: 101}},
		{name: "unsupported quantity precision", item: model.SalesOrderItem{ProductID: "550e8400-e29b-41d4-a716-446655440001", Description: "Pump", Unit: "UNIT", Quantity: 1.00001, UnitPrice: 1}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := service.CreateSalesOrder(context.Background(), &model.CreateSalesOrderRequest{
				CustomerID: "550e8400-e29b-41d4-a716-446655440000",
				Currency:   "IDR",
				Items:      []model.SalesOrderItem{test.item},
			})
			if !errors.Is(err, ErrInvalidSalesOrder) {
				t.Fatalf("error = %v, want ErrInvalidSalesOrder", err)
			}
		})
	}
}

func TestRecordInventoryMovementValidatesTypeAndQuantity(t *testing.T) {
	repository := &salesInventoryRepositoryStub{}
	service := NewSalesInventoryService(repository)
	request := &model.InventoryMovementRequest{
		ProductID:     "550e8400-e29b-41d4-a716-446655440001",
		WarehouseID:   "550e8400-e29b-41d4-a716-446655440002",
		QuantityDelta: 12,
		Reason:        "Supplier receipt",
	}
	if err := service.RecordInventoryMovement(context.Background(), "receipt", request); err != nil {
		t.Fatalf("RecordInventoryMovement() error = %v", err)
	}
	if repository.movementType != "RECEIPT" {
		t.Fatalf("movement type = %q, want RECEIPT", repository.movementType)
	}
	request.QuantityDelta = 0
	if err := service.RecordInventoryMovement(context.Background(), "RECEIPT", request); !errors.Is(err, ErrInvalidInventory) {
		t.Fatalf("error = %v, want ErrInvalidInventory", err)
	}
}
