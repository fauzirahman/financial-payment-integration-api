package handler

import (
	"errors"
	"net/http"

	"github.com/fauzirahman/financial-payment-integration-api/internal/service"
)

type ReportingHandler struct {
	service *service.ReportingService
}

func NewReportingHandler(service *service.ReportingService) *ReportingHandler {
	return &ReportingHandler{service: service}
}

func (h *ReportingHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	fromDate := r.URL.Query().Get("from")
	toDate := r.URL.Query().Get("to")
	var result any
	var err error
	switch r.URL.Path {
	case "/api/v1/reports/payment-summary":
		result, err = h.service.GetPaymentSummary(r.Context(), fromDate, toDate)
	case "/api/v1/reports/daily-payments":
		result, err = h.service.GetDailyPaymentReport(r.Context(), fromDate, toDate)
	case "/api/v1/reports/general-ledger":
		result, err = h.service.GetGeneralLedgerReport(r.Context(), fromDate, toDate, r.URL.Query().Get("account_code"))
	default:
		writeError(w, http.StatusNotFound, "report not found")
		return
	}
	if err != nil {
		if errors.Is(err, service.ErrInvalidReportDateRange) {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	writeJSON(w, http.StatusOK, result)
}
