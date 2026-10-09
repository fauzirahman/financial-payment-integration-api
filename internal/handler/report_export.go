package handler

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/fauzirahman/financial-payment-integration-api/internal/model"
	"github.com/go-pdf/fpdf"
	"github.com/xuri/excelize/v2"
)

type reportTable struct {
	Headers        []string
	Rows           [][]string
	NumericColumns []int
}

func exportReport(format, title string, result any) ([]byte, string, string, error) {
	table, err := buildReportTable(result)
	if err != nil {
		return nil, "", "", err
	}
	switch format {
	case "csv":
		content, err := renderCSV(table)
		return content, "text/csv; charset=utf-8", "csv", err
	case "xlsx":
		content, err := renderXLSX(table)
		return content, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", "xlsx", err
	case "pdf":
		content, err := renderPDF(title, table)
		return content, "application/pdf", "pdf", err
	default:
		return nil, "", "", fmt.Errorf("unsupported report format %q", format)
	}
}

func buildReportTable(result any) (reportTable, error) {
	switch rows := result.(type) {
	case []model.PaymentSummary:
		table := reportTable{
			Headers:        []string{"Currency", "Total Payments", "Pending", "Successful", "Failed", "Total Amount", "Successful Amount"},
			NumericColumns: []int{1, 2, 3, 4, 5, 6},
		}
		for _, row := range rows {
			table.Rows = append(table.Rows, []string{
				row.Currency,
				strconv.FormatInt(row.TotalPayments, 10),
				strconv.FormatInt(row.PendingPayments, 10),
				strconv.FormatInt(row.SuccessfulPayments, 10),
				strconv.FormatInt(row.FailedPayments, 10),
				strconv.FormatInt(row.TotalAmount, 10),
				strconv.FormatInt(row.SuccessfulAmount, 10),
			})
		}
		return table, nil
	case []model.DailyPaymentReport:
		table := reportTable{
			Headers:        []string{"Date", "Currency", "Total Payments", "Successful", "Failed", "Total Amount", "Successful Amount"},
			NumericColumns: []int{2, 3, 4, 5, 6},
		}
		for _, row := range rows {
			table.Rows = append(table.Rows, []string{
				row.Date,
				row.Currency,
				strconv.FormatInt(row.TotalPayments, 10),
				strconv.FormatInt(row.SuccessfulPayments, 10),
				strconv.FormatInt(row.FailedPayments, 10),
				strconv.FormatInt(row.TotalAmount, 10),
				strconv.FormatInt(row.SuccessfulAmount, 10),
			})
		}
		return table, nil
	case []model.GeneralLedgerEntry:
		table := reportTable{
			Headers:        []string{"Journal ID", "Payment Reference", "Account Code", "Entry Type", "Debit", "Credit", "Running Balance", "Currency", "Posted At"},
			NumericColumns: []int{0, 4, 5, 6},
		}
		for _, row := range rows {
			table.Rows = append(table.Rows, []string{
				strconv.FormatInt(row.JournalID, 10),
				row.PaymentReference,
				row.AccountCode,
				row.EntryType,
				strconv.FormatInt(row.DebitAmount, 10),
				strconv.FormatInt(row.CreditAmount, 10),
				strconv.FormatInt(row.RunningBalance, 10),
				row.Currency,
				row.PostedAt.Format(time.RFC3339),
			})
		}
		return table, nil
	default:
		return reportTable{}, fmt.Errorf("unsupported report data type %T", result)
	}
}

func renderCSV(table reportTable) ([]byte, error) {
	var buffer bytes.Buffer
	buffer.Write([]byte{0xEF, 0xBB, 0xBF})
	writer := csv.NewWriter(&buffer)
	if err := writer.Write(table.Headers); err != nil {
		return nil, err
	}
	for _, row := range table.Rows {
		record := make([]string, len(row))
		for index, value := range row {
			if isNumericReportColumn(index, table.NumericColumns) {
				record[index] = value
			} else {
				record[index] = safeSpreadsheetText(value)
			}
		}
		if err := writer.Write(record); err != nil {
			return nil, err
		}
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

func renderXLSX(table reportTable) ([]byte, error) {
	file := excelize.NewFile()
	defer file.Close()
	const sheet = "Report"
	if err := file.SetSheetName("Sheet1", sheet); err != nil {
		return nil, err
	}
	for column, value := range table.Headers {
		cell, err := excelize.CoordinatesToCellName(column+1, 1)
		if err != nil {
			return nil, err
		}
		if err := file.SetCellValue(sheet, cell, value); err != nil {
			return nil, err
		}
	}
	for rowIndex, row := range table.Rows {
		for columnIndex, value := range row {
			cell, err := excelize.CoordinatesToCellName(columnIndex+1, rowIndex+2)
			if err != nil {
				return nil, err
			}
			var setErr error
			if isNumericReportColumn(columnIndex, table.NumericColumns) {
				number, parseErr := strconv.ParseInt(value, 10, 64)
				if parseErr != nil {
					return nil, parseErr
				}
				setErr = file.SetCellValue(sheet, cell, number)
			} else {
				setErr = file.SetCellStr(sheet, cell, safeSpreadsheetText(value))
			}
			if setErr != nil {
				return nil, setErr
			}
		}
	}
	lastHeader, err := excelize.CoordinatesToCellName(len(table.Headers), 1)
	if err != nil {
		return nil, err
	}
	style, err := file.NewStyle(&excelize.Style{Font: &excelize.Font{Bold: true}})
	if err != nil {
		return nil, err
	}
	if err := file.SetCellStyle(sheet, "A1", lastHeader, style); err != nil {
		return nil, err
	}
	lastColumn, err := excelize.ColumnNumberToName(len(table.Headers))
	if err != nil {
		return nil, err
	}
	if len(table.Rows) > 0 {
		if err := file.AutoFilter(sheet, fmt.Sprintf("A1:%s%d", lastColumn, len(table.Rows)+1), []excelize.AutoFilterOptions{}); err != nil {
			return nil, err
		}
	}
	for column := 1; column <= len(table.Headers); column++ {
		name, err := excelize.ColumnNumberToName(column)
		if err != nil {
			return nil, err
		}
		if err := file.SetColWidth(sheet, name, name, 20); err != nil {
			return nil, err
		}
	}
	var buffer bytes.Buffer
	if err := file.Write(&buffer); err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

func isNumericReportColumn(column int, numericColumns []int) bool {
	for _, numericColumn := range numericColumns {
		if column == numericColumn {
			return true
		}
	}
	return false
}

func renderPDF(title string, table reportTable) ([]byte, error) {
	pdf := fpdf.New("L", "mm", "A4", "")
	pdf.SetMargins(10, 10, 10)
	pdf.SetAutoPageBreak(true, 12)
	pdf.AddPage()
	pdf.SetFont("Arial", "B", 14)
	pdf.CellFormat(0, 9, title, "", 1, "L", false, 0, "")
	pdf.SetFont("Arial", "", 8)
	pdf.CellFormat(0, 6, "Generated: "+time.Now().UTC().Format(time.RFC3339), "", 1, "L", false, 0, "")
	pdf.Ln(3)

	columnWidth := (277.0) / float64(len(table.Headers))
	pdf.SetFont("Arial", "B", 7)
	for _, header := range table.Headers {
		pdf.CellFormat(columnWidth, 8, pdfText(header), "1", 0, "L", false, 0, "")
	}
	pdf.Ln(-1)
	pdf.SetFont("Arial", "", 7)
	for _, row := range table.Rows {
		for _, value := range row {
			pdf.CellFormat(columnWidth, 7, pdfText(value), "1", 0, "L", false, 0, "")
		}
		pdf.Ln(-1)
	}
	var buffer bytes.Buffer
	if err := pdf.Output(&buffer); err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

func safeSpreadsheetText(value string) string {
	trimmed := strings.TrimLeft(value, " \t\r\n")
	if trimmed == "" {
		return value
	}
	switch trimmed[0] {
	case '=', '+', '-', '@':
		return "'" + value
	default:
		return value
	}
}

func pdfText(value string) string {
	return strings.ToValidUTF8(value, "?")
}
