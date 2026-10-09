package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/fauzirahman/financial-payment-integration-api/internal/model"
	"github.com/fauzirahman/financial-payment-integration-api/internal/repository"
)

var ErrInvalidReportDateRange = errors.New("invalid report date range")

type ReportingService struct {
	repository repository.ReportingRepository
}

func NewReportingService(repository repository.ReportingRepository) *ReportingService {
	return &ReportingService{repository: repository}
}

func (s *ReportingService) GetPaymentSummary(ctx context.Context, fromDate, toDate string) ([]model.PaymentSummary, error) {
	filter, err := buildReportFilter(fromDate, toDate, "")
	if err != nil {
		return nil, err
	}
	return s.repository.PaymentSummary(ctx, filter)
}

func (s *ReportingService) GetDailyPaymentReport(ctx context.Context, fromDate, toDate string) ([]model.DailyPaymentReport, error) {
	filter, err := buildReportFilter(fromDate, toDate, "")
	if err != nil {
		return nil, err
	}
	return s.repository.DailyPaymentReport(ctx, filter)
}

func (s *ReportingService) GetGeneralLedgerReport(ctx context.Context, fromDate, toDate, accountCode string) ([]model.GeneralLedgerEntry, error) {
	filter, err := buildReportFilter(fromDate, toDate, accountCode)
	if err != nil {
		return nil, err
	}
	return s.repository.GeneralLedgerReport(ctx, filter)
}

func buildReportFilter(fromDate, toDate, accountCode string) (repository.ReportFilter, error) {
	filter := repository.ReportFilter{AccountCode: strings.TrimSpace(accountCode)}
	if fromDate != "" {
		from, err := time.Parse("2006-01-02", fromDate)
		if err != nil {
			return repository.ReportFilter{}, ErrInvalidReportDateRange
		}
		filter.From = &from
	}
	if toDate != "" {
		to, err := time.Parse("2006-01-02", toDate)
		if err != nil {
			return repository.ReportFilter{}, ErrInvalidReportDateRange
		}
		to = to.AddDate(0, 0, 1)
		filter.ToExclusive = &to
	}
	if filter.From != nil && filter.ToExclusive != nil && !filter.From.Before(*filter.ToExclusive) {
		return repository.ReportFilter{}, ErrInvalidReportDateRange
	}
	return filter, nil
}
