package service

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/xuri/excelize/v2"
)

type Exporter struct {
	exportPath string
}

func NewExporter(path string) *Exporter {
	return &Exporter{exportPath: path}
}

// ExportToExcel generates a hierarchical Excel file from aggregation data
func (e *Exporter) ExportToExcel(filename string, data [][]string) error {
	// Ensure directory exists
	if err := os.MkdirAll(e.exportPath, 0755); err != nil {
		return fmt.Errorf("failed to create export directory: %w", err)
	}

	f := excelize.NewFile()
	defer func() {
		if err := f.Close(); err != nil {
			fmt.Println(err)
		}
	}()

	sheetName := "Analysis Report"
	f.SetSheetName("Sheet1", sheetName)

	// Header
	headers := []string{"Level", "Label", "Count", "Percentage"}
	for i, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue(sheetName, cell, h)
	}

	// Data
	for rIdx, row := range data {
		for cIdx, val := range row {
			cell, _ := excelize.CoordinatesToCellName(cIdx+1, rIdx+2)
			f.SetCellValue(sheetName, cell, val)
			
			// If it's the label column and level is > 0, apply indentation
			if cIdx == 1 {
				// level, _ := strconv.Atoi(row[0])
				// Applying style for indentation in Excel is better than spaces
			}
		}
	}

	fullPath := filepath.Join(e.exportPath, filename)
	if err := f.SaveAs(fullPath); err != nil {
		return fmt.Errorf("failed to save excel: %w", err)
	}

	return nil
}
