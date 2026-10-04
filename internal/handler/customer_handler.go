package handler

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/fauzirahman/financial-payment-integration-api/internal/model"
	"github.com/fauzirahman/financial-payment-integration-api/internal/service"
)

type CustomerHandler struct {
	service *service.CustomerService
}

func NewCustomerHandler(service *service.CustomerService) *CustomerHandler {
	return &CustomerHandler{service: service}
}

// Swagger: @Summary Create a new customer
// Swagger: @Tags customers
// Swagger: @Accept json
// Swagger: @Produce json
// Swagger: @Param customer body model.Customer true "Customer data"
// Swagger: @Success 201 {object} model.Customer
// Swagger: @Failure 400 {object} map[string]string
// Swagger: @Router /api/v1/customers [post]
func (h *CustomerHandler) CreateCustomer(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var customer model.Customer
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&customer); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		writeError(w, http.StatusBadRequest, "request body must contain a single JSON object")
		return
	}

	if err := h.service.CreateCustomer(r.Context(), &customer); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, customer)
}

// Swagger: @Summary List all customers
// Swagger: @Tags customers
// Swagger: @Produce json
// Swagger: @Success 200 {array} model.Customer
// Swagger: @Failure 500 {object} map[string]string
// Swagger: @Router /api/v1/customers [get]
func (h *CustomerHandler) GetCustomers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	customers, err := h.service.ListCustomers(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	writeJSON(w, http.StatusOK, customers)
}

// Swagger: @Summary Get customer by ID
// Swagger: @Tags customers
// Swagger: @Produce json
// Swagger: @Param id path string true "Customer ID"
// Swagger: @Success 200 {object} model.Customer
// Swagger: @Failure 400 {object} map[string]string
// Swagger: @Failure 404 {object} map[string]string
// Swagger: @Router /api/v1/customers/{id} [get]
func (h *CustomerHandler) GetCustomerByID(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	id := strings.TrimPrefix(r.URL.Path, "/api/v1/customers/")
	if id == "" || strings.Contains(id, "/") {
		writeError(w, http.StatusBadRequest, "invalid customer id")
		return
	}

	customer, err := h.service.GetCustomerByID(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "customer not found")
		return
	}
	writeJSON(w, http.StatusOK, customer)
}
