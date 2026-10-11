package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/fauzirahman/financial-payment-integration-api/internal/model"
	"github.com/fauzirahman/financial-payment-integration-api/internal/repository"
	"github.com/fauzirahman/financial-payment-integration-api/internal/service"
)

type salesInventoryHandlerRepositoryStub struct {
	balances []model.InventoryBalance
}

func (*salesInventoryHandlerRepositoryStub) CreateSalesOrder(context.Context, *model.SalesOrder) error {
	return nil
}
func (*salesInventoryHandlerRepositoryStub) ListSalesOrders(context.Context) ([]model.SalesOrder, error) {
	return nil, nil
}
func (*salesInventoryHandlerRepositoryStub) GetSalesOrder(context.Context, string) (*model.SalesOrder, error) {
	return nil, repository.ErrSalesOrderNotFound
}
func (*salesInventoryHandlerRepositoryStub) ListSalesActivity(context.Context, string) ([]model.SalesActivity, error) {
	return nil, nil
}
func (*salesInventoryHandlerRepositoryStub) ListDeliveries(context.Context, string) ([]model.DeliveryOrder, error) {
	return nil, nil
}
func (*salesInventoryHandlerRepositoryStub) ConfirmSalesOrder(context.Context, string, string) error {
	return nil
}
func (*salesInventoryHandlerRepositoryStub) DeliverSalesOrder(context.Context, string, string) (*model.DeliveryOrder, error) {
	return nil, nil
}
func (r *salesInventoryHandlerRepositoryStub) ListInventory(context.Context) ([]model.InventoryBalance, error) {
	return r.balances, nil
}
func (*salesInventoryHandlerRepositoryStub) ListInventoryMovements(context.Context, string, string, int) ([]model.InventoryMovement, error) {
	return nil, nil
}
func (*salesInventoryHandlerRepositoryStub) RecordInventoryMovement(context.Context, string, *model.InventoryMovementRequest) error {
	return nil
}
func (*salesInventoryHandlerRepositoryStub) CreateProduct(context.Context, *model.Product) error {
	return nil
}
func (*salesInventoryHandlerRepositoryStub) ListProducts(context.Context) ([]model.Product, error) {
	return nil, nil
}
func (*salesInventoryHandlerRepositoryStub) CreateWarehouse(context.Context, *model.Warehouse) error {
	return nil
}
func (*salesInventoryHandlerRepositoryStub) ListWarehouses(context.Context) ([]model.Warehouse, error) {
	return nil, nil
}

func TestSalesOrderHandlerRejectsInvalidRequest(t *testing.T) {
	handler := NewSalesInventoryHandler(service.NewSalesInventoryService(&salesInventoryHandlerRepositoryStub{}))
	request := httptest.NewRequest(http.MethodPost, "/api/v1/sales-orders", strings.NewReader(`{"customer_id":"invalid"}`))
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d; body: %s", response.Code, http.StatusBadRequest, response.Body)
	}
}

func TestInventoryHandlerListsBalances(t *testing.T) {
	stub := &salesInventoryHandlerRepositoryStub{balances: []model.InventoryBalance{{SKU: "PUMP-1", Available: 7}}}
	handler := NewSalesInventoryHandler(service.NewSalesInventoryService(stub))
	request := httptest.NewRequest(http.MethodGet, "/api/v1/inventory", nil)
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"available":7`) {
		t.Fatalf("status/body = %d %s, want available stock", response.Code, response.Body)
	}
}

func TestInventoryMovementHandlerRejectsInvalidLimit(t *testing.T) {
	handler := NewSalesInventoryHandler(service.NewSalesInventoryService(&salesInventoryHandlerRepositoryStub{}))
	request := httptest.NewRequest(http.MethodGet, "/api/v1/inventory/movements?limit=201", nil)
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusBadRequest)
	}
}

func TestSalesOrderHandlerReturnsNotFound(t *testing.T) {
	handler := NewSalesInventoryHandler(service.NewSalesInventoryService(&salesInventoryHandlerRepositoryStub{}))
	request := httptest.NewRequest(http.MethodGet, "/api/v1/sales-orders/550e8400-e29b-41d4-a716-446655440000", nil)
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusNotFound)
	}
}
