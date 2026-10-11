package handler

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/fauzirahman/financial-payment-integration-api/internal/model"
	"github.com/fauzirahman/financial-payment-integration-api/internal/repository"
	"github.com/fauzirahman/financial-payment-integration-api/internal/service"
	"github.com/jackc/pgx/v5"
)

type SalesInventoryHandler struct {
	service *service.SalesInventoryService
}

func NewSalesInventoryHandler(service *service.SalesInventoryService) *SalesInventoryHandler {
	return &SalesInventoryHandler{service: service}
}

func (h *SalesInventoryHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/v1/"), "/")
	parts := strings.Split(path, "/")
	if len(parts) == 0 {
		writeError(w, http.StatusNotFound, "resource not found")
		return
	}

	switch parts[0] {
	case "sales-orders":
		h.serveSalesOrders(w, r, parts[1:])
	case "inventory":
		h.serveInventory(w, r, parts[1:])
	case "products":
		h.serveProducts(w, r, parts[1:])
	case "warehouses":
		h.serveWarehouses(w, r, parts[1:])
	default:
		writeError(w, http.StatusNotFound, "resource not found")
	}
}

func (h *SalesInventoryHandler) serveSalesOrders(w http.ResponseWriter, r *http.Request, parts []string) {
	if len(parts) == 0 || parts[0] == "" {
		switch r.Method {
		case http.MethodGet:
			orders, err := h.service.ListSalesOrders(r.Context())
			if err != nil {
				writeError(w, http.StatusInternalServerError, "internal server error")
				return
			}
			writeJSON(w, http.StatusOK, orders)
		case http.MethodPost:
			var request model.CreateSalesOrderRequest
			if !decodeJSONBody(w, r, &request) {
				return
			}
			order, err := h.service.CreateSalesOrder(r.Context(), &request)
			if err != nil {
				writeSalesInventoryError(w, err)
				return
			}
			writeJSON(w, http.StatusCreated, order)
		default:
			methodNotAllowed(w, http.MethodGet, http.MethodPost)
		}
		return
	}
	if len(parts) == 2 && (parts[1] == "activity" || parts[1] == "deliveries") {
		if r.Method != http.MethodGet {
			methodNotAllowed(w, http.MethodGet)
			return
		}
		if parts[1] == "activity" {
			activities, err := h.service.ListSalesActivity(r.Context(), parts[0])
			if errors.Is(err, repository.ErrSalesOrderNotFound) {
				writeError(w, http.StatusNotFound, "sales order not found")
				return
			}
			if err != nil {
				writeSalesInventoryError(w, err)
				return
			}
			writeJSON(w, http.StatusOK, activities)
			return
		}
		deliveries, err := h.service.ListDeliveries(r.Context(), parts[0])
		if errors.Is(err, repository.ErrSalesOrderNotFound) {
			writeError(w, http.StatusNotFound, "sales order not found")
			return
		}
		if err != nil {
			writeSalesInventoryError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, deliveries)
		return
	}
	if len(parts) == 2 && (parts[1] == "confirm" || parts[1] == "deliver") {
		if r.Method != http.MethodPost {
			methodNotAllowed(w, http.MethodPost)
			return
		}
		var request model.ConfirmSalesOrderRequest
		if !decodeJSONBody(w, r, &request) {
			return
		}
		if parts[1] == "confirm" {
			if err := h.service.ConfirmSalesOrder(r.Context(), parts[0], request.WarehouseID); err != nil {
				writeSalesInventoryError(w, err)
				return
			}
			writeJSON(w, http.StatusOK, map[string]string{"status": "CONFIRMED"})
			return
		}
		delivery, err := h.service.DeliverSalesOrder(r.Context(), parts[0], request.WarehouseID)
		if err != nil {
			writeSalesInventoryError(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, delivery)
		return
	}
	if len(parts) != 1 || r.Method != http.MethodGet {
		methodNotAllowed(w, http.MethodGet)
		return
	}
	order, err := h.service.GetSalesOrder(r.Context(), parts[0])
	if errors.Is(err, repository.ErrSalesOrderNotFound) {
		writeError(w, http.StatusNotFound, "sales order not found")
		return
	}
	if err != nil {
		writeSalesInventoryError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, order)
}

func (h *SalesInventoryHandler) serveInventory(w http.ResponseWriter, r *http.Request, parts []string) {
	if len(parts) == 0 || parts[0] == "" {
		if r.Method != http.MethodGet {
			methodNotAllowed(w, http.MethodGet)
			return
		}
		balances, err := h.service.ListInventory(r.Context())
		if err != nil {
			writeError(w, http.StatusInternalServerError, "internal server error")
			return
		}
		writeJSON(w, http.StatusOK, balances)
		return
	}
	if len(parts) != 1 || parts[0] != "movements" {
		methodNotAllowed(w, http.MethodGet, http.MethodPost)
		return
	}
	if r.Method == http.MethodGet {
		limit := 50
		if rawLimit := r.URL.Query().Get("limit"); rawLimit != "" {
			parsedLimit, err := strconv.Atoi(rawLimit)
			if err != nil || parsedLimit < 1 || parsedLimit > 200 {
				writeError(w, http.StatusBadRequest, "limit must be between 1 and 200")
				return
			}
			limit = parsedLimit
		}
		movements, err := h.service.ListInventoryMovements(r.Context(), r.URL.Query().Get("product_id"), r.URL.Query().Get("warehouse_id"), limit)
		if err != nil {
			writeSalesInventoryError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, movements)
		return
	}
	if r.Method != http.MethodPost {
		methodNotAllowed(w, http.MethodGet, http.MethodPost)
		return
	}
	var request model.InventoryMovementRequest
	if !decodeJSONBody(w, r, &request) {
		return
	}
	if err := h.service.RecordInventoryMovement(r.Context(), r.URL.Query().Get("type"), &request); err != nil {
		writeSalesInventoryError(w, err)
		return
	}
	w.WriteHeader(http.StatusCreated)
}

func (h *SalesInventoryHandler) serveProducts(w http.ResponseWriter, r *http.Request, parts []string) {
	if len(parts) != 0 && parts[0] != "" {
		writeError(w, http.StatusNotFound, "product not found")
		return
	}
	switch r.Method {
	case http.MethodGet:
		products, err := h.service.ListProducts(r.Context())
		if err != nil {
			writeError(w, http.StatusInternalServerError, "internal server error")
			return
		}
		writeJSON(w, http.StatusOK, products)
	case http.MethodPost:
		var product model.Product
		if !decodeJSONBody(w, r, &product) {
			return
		}
		if err := h.service.CreateProduct(r.Context(), &product); err != nil {
			writeSalesInventoryError(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, product)
	default:
		methodNotAllowed(w, http.MethodGet, http.MethodPost)
	}
}

func (h *SalesInventoryHandler) serveWarehouses(w http.ResponseWriter, r *http.Request, parts []string) {
	if len(parts) != 0 && parts[0] != "" {
		writeError(w, http.StatusNotFound, "warehouse not found")
		return
	}
	switch r.Method {
	case http.MethodGet:
		warehouses, err := h.service.ListWarehouses(r.Context())
		if err != nil {
			writeError(w, http.StatusInternalServerError, "internal server error")
			return
		}
		writeJSON(w, http.StatusOK, warehouses)
	case http.MethodPost:
		var warehouse model.Warehouse
		if !decodeJSONBody(w, r, &warehouse) {
			return
		}
		if err := h.service.CreateWarehouse(r.Context(), &warehouse); err != nil {
			writeSalesInventoryError(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, warehouse)
	default:
		methodNotAllowed(w, http.MethodGet, http.MethodPost)
	}
}

func decodeJSONBody(w http.ResponseWriter, r *http.Request, target any) bool {
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return false
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		writeError(w, http.StatusBadRequest, "request body must contain a single JSON object")
		return false
	}
	return true
}

func methodNotAllowed(w http.ResponseWriter, methods ...string) {
	w.Header().Set("Allow", strings.Join(methods, ", "))
	writeError(w, http.StatusMethodNotAllowed, "method not allowed")
}

func writeSalesInventoryError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, service.ErrInvalidSalesOrder), errors.Is(err, service.ErrInvalidInventory):
		writeError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, repository.ErrSalesOrderNotFound), errors.Is(err, repository.ErrWarehouseNotFound), errors.Is(err, repository.ErrProductNotFound), errors.Is(err, pgx.ErrNoRows):
		writeError(w, http.StatusNotFound, err.Error())
	case errors.Is(err, repository.ErrSalesOrderState), errors.Is(err, repository.ErrInsufficientStock):
		writeError(w, http.StatusConflict, err.Error())
	default:
		writeError(w, http.StatusInternalServerError, "internal server error")
	}
}
