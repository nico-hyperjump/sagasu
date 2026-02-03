package extract

import (
	"bytes"
	"fmt"

	"github.com/ledongthuc/pdf"
)

func extractPDF(content []byte) (string, error) {
	r, err := pdf.NewReader(bytes.NewReader(content), int64(len(content)))
	if err != nil {
		return "", fmt.Errorf("open PDF: %w", err)
	}
	var buf bytes.Buffer
	numPages := r.NumPage()
	for i := 0; i < numPages; i++ {
		page := r.Page(i + 1)
		if page.V.IsNull() {
			continue
		}
		text, err := page.GetPlainText(nil)
		if err != nil {
			return "", fmt.Errorf("extract page %d: %w", i+1, err)
		}
		if _, err := buf.WriteString(text); err != nil {
			return "", fmt.Errorf("write page %d: %w", i+1, err)
		}
		if i < numPages-1 {
			buf.WriteByte('\n')
		}
	}
	return buf.String(), nil
}
