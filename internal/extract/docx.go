package extract

import (
	"archive/zip"
	"bytes"
	"fmt"
	"regexp"
	"strings"
)

// docxDocumentXMLPath is the path to the main document body inside a .docx zip.
const docxDocumentXMLPath = "word/document.xml"

// w-tTag matches <w:t>text</w:t> or <w:t xml:space="preserve">text</w:t> (and any other attributes).
var wtTag = regexp.MustCompile(`<w:t[^>]*>([^<]*)</w:t>`)

// extractDOCX extracts text from .docx bytes. DOCX is a ZIP containing word/document.xml
// (OOXML). We extract all <w:t>...</w:t> text nodes so content is searchable regardless
// of paragraph/run attributes. We do not use lu4p/cat because its regex only matches
// <w:p>(.*)</w:p> without attributes, so real-world docs (e.g. <w:p w:rsidR="...">) yield empty.
func extractDOCX(content []byte) (string, error) {
	zr, err := zip.NewReader(bytes.NewReader(content), int64(len(content)))
	if err != nil {
		return "", fmt.Errorf("extract DOCX: not a zip: %w", err)
	}
	var docXML []byte
	for _, f := range zr.File {
		if f.Name != docxDocumentXMLPath {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return "", fmt.Errorf("extract DOCX: open %s: %w", f.Name, err)
		}
		var buf bytes.Buffer
		if _, err := buf.ReadFrom(rc); err != nil {
			_ = rc.Close()
			return "", fmt.Errorf("extract DOCX: read %s: %w", f.Name, err)
		}
		_ = rc.Close()
		docXML = buf.Bytes()
		break
	}
	if docXML == nil {
		return "", fmt.Errorf("extract DOCX: %s not found", docxDocumentXMLPath)
	}
	// Extract all <w:t>...</w:t> inner text and join with spaces; collapse newlines to space.
	parts := wtTag.FindAllStringSubmatch(string(docXML), -1)
	if len(parts) == 0 {
		return "", nil
	}
	var b strings.Builder
	for i, p := range parts {
		if i > 0 {
			b.WriteByte(' ')
		}
		b.WriteString(strings.TrimSpace(p[1]))
	}
	return strings.TrimSpace(b.String()), nil
}
