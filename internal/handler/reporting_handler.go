package handler

import (
	"errors"
	"net/http"
	"strings"

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
	format := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("format")))
	if format == "" {
		format = "json"
	}
	if format == "excel" {
		format = "xlsx"
	}
	if format != "json" && format != "csv" && format != "xlsx" && format != "pdf" {
		writeError(w, http.StatusBadRequest, "format must be json, csv, excel, xlsx, or pdf")
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
	if format == "json" {
		writeJSON(w, http.StatusOK, result)
		return
	}

	title, filename := reportDownloadName(r.URL.Path)
	body, contentType, extension, err := exportReport(format, title, result)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not generate report")
		return
	}
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Disposition", "attachment; filename=\""+filename+"."+extension+"\"")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(body)
}

func reportDownloadName(path string) (string, string) {
	switch path {
	case "/api/v1/reports/payment-summary":
		return "Payment Summary", "payment-summary"
	case "/api/v1/reports/daily-payments":
		return "Daily Payment Report", "daily-payment-report"
	default:
		return "General Ledger Report", "general-ledger-report"
	}
}
