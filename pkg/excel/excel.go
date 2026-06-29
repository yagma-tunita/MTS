package excel

import (
	"fmt"
	"mime/multipart"
	"strconv"

	"github.com/xuri/excelize/v2"
)

// ReadSheet reads the first sheet of an uploaded Excel file and returns rows as [][]string.
// The first row is assumed to be the header.
func ReadSheet(file multipart.File, fileSize int64) ([][]string, error) {
	f, err := excelize.OpenReader(file)
	if err != nil {
		return nil, fmt.Errorf("failed to open excel file: %w", err)
	}
	defer f.Close()

	sheetName := f.GetSheetName(0)
	rows, err := f.GetRows(sheetName)
	if err != nil {
		return nil, fmt.Errorf("failed to read rows: %w", err)
	}
	if len(rows) < 2 {
		return nil, fmt.Errorf("excel file must have at least one header row and one data row")
	}
	return rows, nil
}

// WriteSheet creates an Excel file with given headers and data rows.
func WriteSheet(headers []string, data [][]string) ([]byte, error) {
	f := excelize.NewFile()
	sheetName := "Sheet1"
	// Write headers
	for i, header := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue(sheetName, cell, header)
	}
	// Write data rows
	for rowIdx, row := range data {
		for colIdx, cellVal := range row {
			cell, _ := excelize.CoordinatesToCellName(colIdx+1, rowIdx+2)
			f.SetCellValue(sheetName, cell, cellVal)
		}
	}
	// Auto-fit columns
	for i := 1; i <= len(headers); i++ {
		colName, _ := excelize.ColumnNumberToName(i)
		f.SetColWidth(sheetName, colName, colName, 15)
	}
	buf, err := f.WriteToBuffer()
	if err != nil {
		return nil, fmt.Errorf("failed to write excel buffer: %w", err)
	}
	return buf.Bytes(), nil
}

// Helper: parse string to float64
func ParseFloat(s string) float64 {
	v, _ := strconv.ParseFloat(s, 64)
	return v
}

// Helper: parse string to int64
func ParseInt(s string) int64 {
	v, _ := strconv.ParseInt(s, 10, 64)
	return v
}
