package handler

import (
	"bytes"
	"context"
	"encoding/csv"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/fauzirahman/financial-payment-integration-api/internal/model"
	"github.com/fauzirahman/financial-payment-integration-api/internal/repository"
	"github.com/fauzirahman/financial-payment-integration-api/internal/service"
	"github.com/xuri/excelize/v2"
)

type reportExportRepositoryStub struct{}

func (reportExportRepositoryStub) PaymentSummary(context.Context, repository.ReportFilter) ([]model.PaymentSummary, error) {
	return []model.PaymentSummary{{Currency: "IDR", TotalPayments: 2, SuccessfulAmount: 125000}}, nil
}

func (reportExportRepositoryStub) DailyPaymentReport(context.Context, repository.ReportFilter) ([]model.DailyPaymentReport, error) {
	return []model.DailyPaymentReport{{Date: "2026-10-09", Currency: "IDR", TotalPayments: 2}}, nil
}

func (reportExportRepositoryStub) GeneralLedgerReport(context.Context, repository.ReportFilter) ([]model.GeneralLedgerEntry, error) {
	return []model.GeneralLedgerEntry{{JournalID: 1, PaymentReference: "PAY-001", AccountCode: "1010", EntryType: "DEBIT", DebitAmount: 125000, Currency: "IDR", PostedAt: time.Date(2026, 10, 9, 10, 0, 0, 0, time.UTC)}}, nil
}

func TestReportExportsReturnDownloadableFiles(t *testing.T) {
	handler := NewReportingHandler(service.NewReportingService(reportExportRepositoryStub{}))
	tests := []struct {
		path      string
		format    string
		mime      string
		filename  string
		fileStart string
	}{
		{path: "/api/v1/reports/payment-summary", format: "csv", mime: "text/csv; charset=utf-8", filename: "payment-summary.csv", fileStart: "\xef\xbb\xbf"},
		{path: "/api/v1/reports/daily-payments", format: "xlsx", mime: "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", filename: "daily-payment-report.xlsx", fileStart: "PK"},
		{path: "/api/v1/reports/general-ledger", format: "pdf", mime: "application/pdf", filename: "general-ledger-report.pdf", fileStart: "%PDF-"},
	}

	for _, test := range tests {
		t.Run(test.format, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, test.path+"?format="+test.format, nil)
			response := httptest.NewRecorder()

			handler.ServeHTTP(response, request)

			if response.Code != http.StatusOK {
				t.Fatalf("status = %d, want %d; body = %s", response.Code, http.StatusOK, response.Body)
			}
			if got := response.Header().Get("Content-Type"); got != test.mime {
				t.Fatalf("Content-Type = %q, want %q", got, test.mime)
			}
			if got := response.Header().Get("Content-Disposition"); !strings.Contains(got, "attachment;") || !strings.Contains(got, test.filename) {
				t.Fatalf("Content-Disposition = %q, want attachment filename %q", got, test.filename)
			}
			if !strings.HasPrefix(response.Body.String(), test.fileStart) {
				t.Fatalf("body does not start with %q", test.fileStart)
			}
		})
	}
}

func TestReportExportDefaultsToJSONAndRejectsUnknownFormat(t *testing.T) {
	handler := NewReportingHandler(service.NewReportingService(reportExportRepositoryStub{}))

	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/reports/payment-summary", nil))
	if response.Code != http.StatusOK || response.Header().Get("Content-Type") != "application/json" {
		t.Fatalf("default response = %d %q, want JSON 200", response.Code, response.Header().Get("Content-Type"))
	}

	response = httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/reports/payment-summary?format=xml", nil))
	if response.Code != http.StatusBadRequest {
		t.Fatalf("unknown format status = %d, want %d", response.Code, http.StatusBadRequest)
	}
}

func TestCSVExportEscapesSpreadsheetFormulas(t *testing.T) {
	content, _, _, err := exportReport("csv", "Payment Summary", []model.PaymentSummary{{Currency: "=2+2"}})
	if err != nil {
		t.Fatalf("exportReport() error = %v", err)
	}
	rows, err := csv.NewReader(strings.NewReader(strings.TrimPrefix(string(content), "\xef\xbb\xbf"))).ReadAll()
	if err != nil {
		t.Fatalf("read CSV: %v", err)
	}
	if got := rows[1][0]; got != "'=2+2" {
		t.Fatalf("escaped currency = %q, want formula-safe value", got)
	}
}

func TestXLSXExportUsesNumericCells(t *testing.T) {
	content, _, _, err := exportReport("xlsx", "Payment Summary", []model.PaymentSummary{{Currency: "IDR", TotalPayments: 12}})
	if err != nil {
		t.Fatalf("exportReport() error = %v", err)
	}
	workbook, err := excelize.OpenReader(bytes.NewReader(content))
	if err != nil {
		t.Fatalf("open generated workbook: %v", err)
	}
	defer workbook.Close()

	cellType, err := workbook.GetCellType("Report", "B2")
	if err != nil {
		t.Fatalf("get numeric cell type: %v", err)
	}
	if cellType != excelize.CellTypeUnset {
		t.Fatalf("numeric report cell type = %v, want default numeric cell type %v", cellType, excelize.CellTypeUnset)
	}
	value, err := workbook.GetCellValue("Report", "B2")
	if err != nil {
		t.Fatalf("get numeric cell value: %v", err)
	}
	if value != "12" {
		t.Fatalf("numeric report cell value = %q, want %q", value, "12")
	}
}
