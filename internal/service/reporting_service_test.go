package service

import (
	"context"
	"testing"
	"time"

	"github.com/fauzirahman/financial-payment-integration-api/internal/model"
	"github.com/fauzirahman/financial-payment-integration-api/internal/repository"
)

type reportingRepositoryStub struct {
	filter repository.ReportFilter
}

func (r *reportingRepositoryStub) PaymentSummary(_ context.Context, filter repository.ReportFilter) ([]model.PaymentSummary, error) {
	r.filter = filter
	return []model.PaymentSummary{}, nil
}

func (r *reportingRepositoryStub) DailyPaymentReport(_ context.Context, filter repository.ReportFilter) ([]model.DailyPaymentReport, error) {
	r.filter = filter
	return []model.DailyPaymentReport{}, nil
}

func (r *reportingRepositoryStub) GeneralLedgerReport(_ context.Context, filter repository.ReportFilter) ([]model.GeneralLedgerEntry, error) {
	r.filter = filter
	return []model.GeneralLedgerEntry{}, nil
}

func TestGetPaymentSummaryUsesInclusiveDateRange(t *testing.T) {
	repository := &reportingRepositoryStub{}
	service := NewReportingService(repository)

	if _, err := service.GetPaymentSummary(context.Background(), "2026-10-01", "2026-10-31"); err != nil {
		t.Fatalf("GetPaymentSummary() error = %v", err)
	}
	if repository.filter.From == nil || !repository.filter.From.Equal(time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("from date = %v, want 2026-10-01 UTC", repository.filter.From)
	}
	if repository.filter.ToExclusive == nil || !repository.filter.ToExclusive.Equal(time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("exclusive to date = %v, want 2026-11-01 UTC", repository.filter.ToExclusive)
	}
}

func TestGetGeneralLedgerReportRejectsInvalidDateRange(t *testing.T) {
	service := NewReportingService(&reportingRepositoryStub{})
	tests := []struct {
		name string
		from string
		to   string
	}{
		{name: "invalid date", from: "2026-02-30", to: "2026-03-01"},
		{name: "reversed dates", from: "2026-10-02", to: "2026-10-01"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := service.GetGeneralLedgerReport(context.Background(), test.from, test.to, "1010")
			if err != ErrInvalidReportDateRange {
				t.Fatalf("GetGeneralLedgerReport() error = %v, want ErrInvalidReportDateRange", err)
			}
		})
	}
}
