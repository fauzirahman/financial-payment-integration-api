package repository

import (
	"context"

	"github.com/fauzirahman/financial-payment-integration-api/internal/model"
)

type SalesInventoryRepository interface {
	CreateSalesOrder(ctx context.Context, order *model.SalesOrder) error
	ListSalesOrders(ctx context.Context) ([]model.SalesOrder, error)
	GetSalesOrder(ctx context.Context, id string) (*model.SalesOrder, error)
	ListSalesActivity(ctx context.Context, id string) ([]model.SalesActivity, error)
	ListDeliveries(ctx context.Context, id string) ([]model.DeliveryOrder, error)
	ConfirmSalesOrder(ctx context.Context, id, warehouseID string) error
	DeliverSalesOrder(ctx context.Context, id, warehouseID string) (*model.DeliveryOrder, error)
	ListInventory(ctx context.Context) ([]model.InventoryBalance, error)
	ListInventoryMovements(ctx context.Context, productID, warehouseID string, limit int) ([]model.InventoryMovement, error)
	RecordInventoryMovement(ctx context.Context, movementType string, request *model.InventoryMovementRequest) error
	CreateProduct(ctx context.Context, product *model.Product) error
	ListProducts(ctx context.Context) ([]model.Product, error)
	CreateWarehouse(ctx context.Context, warehouse *model.Warehouse) error
	ListWarehouses(ctx context.Context) ([]model.Warehouse, error)
}
