package handler

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"regexp"
	"strings"

	"github.com/fauzirahman/financial-payment-integration-api/internal/model"
	"github.com/fauzirahman/financial-payment-integration-api/internal/service"
	"github.com/jackc/pgx/v5"
)

var accountIDPattern = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

type AccountHandler struct {
	service *service.AccountService
}

func NewAccountHandler(service *service.AccountService) *AccountHandler {
	return &AccountHandler{service: service}
}

// Swagger: @Summary Create a new account
// Swagger: @Tags accounts
// Swagger: @Accept json
// Swagger: @Produce json
// Swagger: @Param account body model.Account true "Account data"
// Swagger: @Success 201 {object} model.Account
// Swagger: @Failure 400 {object} map[string]string
// Swagger: @Router /api/v1/accounts [post]
func (h *AccountHandler) CreateAccount(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var account model.Account
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&account); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		writeError(w, http.StatusBadRequest, "request body must contain a single JSON object")
		return
	}

	if err := h.service.CreateAccount(r.Context(), &account); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, account)
}

// Swagger: @Summary Get account by ID
// Swagger: @Tags accounts
// Swagger: @Produce json
// Swagger: @Param id path string true "Account UUID"
// Swagger: @Success 200 {object} model.Account
// Swagger: @Failure 400 {object} map[string]string
// Swagger: @Failure 404 {object} map[string]string
// Swagger: @Router /api/v1/accounts/{id} [get]
func (h *AccountHandler) GetAccountByID(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	id := strings.TrimPrefix(r.URL.Path, "/api/v1/accounts/")
	if !accountIDPattern.MatchString(id) {
		writeError(w, http.StatusBadRequest, "invalid account id")
		return
	}

	account, err := h.service.GetAccountByID(r.Context(), id)
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(w, http.StatusNotFound, "account not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	writeJSON(w, http.StatusOK, account)
}

// Swagger: @Summary List accounts for a customer
// Swagger: @Tags accounts
// Swagger: @Produce json
// Swagger: @Param customer_id path string true "Customer ID"
// Swagger: @Success 200 {array} model.Account
// Swagger: @Failure 400 {object} map[string]string
// Swagger: @Failure 500 {object} map[string]string
// Swagger: @Router /api/v1/customers/{customer_id}/accounts [get]
func (h *AccountHandler) GetAccountsByCustomerID(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	customerID := strings.TrimPrefix(r.URL.Path, "/api/v1/customers/")
	customerID = strings.TrimSuffix(customerID, "/accounts")
	if customerID == "" || strings.Contains(customerID, "/") {
		writeError(w, http.StatusBadRequest, "invalid customer id")
		return
	}

	accounts, err := h.service.GetAccountsByCustomerID(r.Context(), customerID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	writeJSON(w, http.StatusOK, accounts)
}
