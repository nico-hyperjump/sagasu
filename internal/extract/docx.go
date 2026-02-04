package extract

import (
	"archive/zip"
	"bytes"
	"fmt"
	"regexp"
	"strings"
)

// docxDocumentXMLPath is the default path to the main document body inside a .docx zip.
const docxDocumentXMLPath = "word/document.xml"

// contentTypesPath is the path to [Content_Types].xml in OOXML packages.
const contentTypesPath = "[Content_Types].xml"

// docxMainContentType is the content type for the main document in DOCX files.
const docxMainContentType = "application/vnd.openxmlformats-officedocument.wordprocessingml.document.main+xml"

// wtTag matches <w:t>text</w:t> or <w:t xml:space="preserve">text</w:t> (and any other attributes).
var wtTag = regexp.MustCompile(`<w:t[^>]*>([^<]*)</w:t>`)

// partNameRe extracts PartName from Override elements in [Content_Types].xml.
var partNameRe = regexp.MustCompile(`<Override[^>]+PartName="([^"]+)"[^>]+ContentType="` + regexp.QuoteMeta(docxMainContentType) + `"`)

// partNameRe2 handles the case where ContentType appears before PartName.
var partNameRe2 = regexp.MustCompile(`<Override[^>]+ContentType="` + regexp.QuoteMeta(docxMainContentType) + `"[^>]+PartName="([^"]+)"`)

// findDocxMainDocumentPath finds the main document path from [Content_Types].xml.
// Returns the path without leading slash, or empty string if not found.
func findDocxMainDocumentPath(zr *zip.Reader) string {
	for _, f := range zr.File {
		if f.Name != contentTypesPath {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return ""
		}
		var buf bytes.Buffer
		if _, err := buf.ReadFrom(rc); err != nil {
			_ = rc.Close()
			return ""
		}
		_ = rc.Close()

		content := buf.String()
		// Try both attribute orders
		if matches := partNameRe.FindStringSubmatch(content); len(matches) > 1 {
			return strings.TrimPrefix(matches[1], "/")
		}
		if matches := partNameRe2.FindStringSubmatch(content); len(matches) > 1 {
			return strings.TrimPrefix(matches[1], "/")
		}
		return ""
	}
	return ""
}

// extractDOCX extracts text from .docx bytes. DOCX is a ZIP containing word/document.xml
// (OOXML). We extract all <w:t>...</w:t> text nodes so content is searchable regardless
// of paragraph/run attributes. We do not use lu4p/cat because its regex only matches
// <w:p>(.*)</w:p> without attributes, so real-world docs (e.g. <w:p w:rsidR="...">) yield empty.
func extractDOCX(content []byte) (string, error) {
	zr, err := zip.NewReader(bytes.NewReader(content), int64(len(content)))
	if err != nil {
		return "", fmt.Errorf("extract DOCX: not a zip: %w", err)
	}

	// Find main document path from [Content_Types].xml, fall back to default
	docPath := findDocxMainDocumentPath(zr)
	if docPath == "" {
		docPath = docxDocumentXMLPath
	}

	var docXML []byte
	for _, f := range zr.File {
		if f.Name != docPath {
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
		return "", fmt.Errorf("extract DOCX: %s not found", docPath)
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
