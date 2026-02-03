package extract

import (
	"archive/zip"
	"bytes"
	"fmt"
	"regexp"
	"strings"
)

// odsContentPath is the path to the main content inside an .ods zip (OpenDocument Spreadsheet).
const odsContentPath = "content.xml"

// odsTextTags match OpenDocument text elements in spreadsheet cells (with optional attributes).
var (
	odsTextP   = regexp.MustCompile(`<text:p[^>]*>([^<]*)</text:p>`)
	odsTextSpan = regexp.MustCompile(`<text:span[^>]*>([^<]*)</text:span>`)
)

// extractODS extracts text from .ods bytes. ODS is a ZIP containing content.xml (OpenDocument).
// We extract all text from text:p and text:span elements so cell content is searchable.
func extractODS(content []byte) (string, error) {
	zr, err := zip.NewReader(bytes.NewReader(content), int64(len(content)))
	if err != nil {
		return "", fmt.Errorf("extract ODS: not a zip: %w", err)
	}
	var contentXML []byte
	for _, f := range zr.File {
		if f.Name != odsContentPath {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return "", fmt.Errorf("extract ODS: open %s: %w", f.Name, err)
		}
		var buf bytes.Buffer
		if _, err := buf.ReadFrom(rc); err != nil {
			_ = rc.Close()
			return "", fmt.Errorf("extract ODS: read %s: %w", f.Name, err)
		}
		_ = rc.Close()
		contentXML = buf.Bytes()
		break
	}
	if contentXML == nil {
		return "", fmt.Errorf("extract ODS: %s not found", odsContentPath)
	}
	s := string(contentXML)
	var b strings.Builder
	appendMatches := func(parts [][]string) {
		for _, p := range parts {
			if b.Len() > 0 {
				b.WriteByte(' ')
			}
			b.WriteString(strings.TrimSpace(p[1]))
		}
	}
	appendMatches(odsTextP.FindAllStringSubmatch(s, -1))
	appendMatches(odsTextSpan.FindAllStringSubmatch(s, -1))
	return strings.TrimSpace(b.String()), nil
}
