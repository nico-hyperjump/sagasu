package extract

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/xuri/excelize/v2"
)

func extractExcel(content []byte) (string, error) {
	f, err := excelize.OpenReader(bytes.NewReader(content))
	if err != nil {
		return "", fmt.Errorf("open Excel: %w", err)
	}
	defer f.Close()

	var buf strings.Builder
	for _, sheet := range f.GetSheetList() {
		rows, err := f.GetRows(sheet)
		if err != nil {
			return "", fmt.Errorf("get rows for sheet %q: %w", sheet, err)
		}
		for _, row := range rows {
			buf.WriteString(strings.Join(row, "\t"))
			buf.WriteByte('\n')
		}
	}
	return strings.TrimSpace(buf.String()), nil
}
