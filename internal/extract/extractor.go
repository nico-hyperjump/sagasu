// Package extract provides text extraction from various document formats.
package extract

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Extractor extracts plain text from document files.
type Extractor struct{}

// NewExtractor returns a new Extractor.
func NewExtractor() *Extractor {
	return &Extractor{}
}

// Extract reads the file at path and returns its text content.
// For plain text files (.txt, .md, .rst), content is returned as-is (UTF-8 validated).
// For PDF, DOCX, Excel, PPTX, ODP, and ODS, text is extracted from the binary format.
// Returns an error if the file cannot be read or the format is unsupported.
func (e *Extractor) Extract(path string) (string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read file: %w", err)
	}
	ext := strings.ToLower(filepath.Ext(path))
	return e.ExtractBytes(content, ext)
}

// ExtractBytes extracts text from content based on the given extension.
// ext should include the leading dot (e.g. ".pdf").
func (e *Extractor) ExtractBytes(content []byte, ext string) (string, error) {
	switch ext {
	case ".pdf":
		return extractPDF(content)
	case ".docx", ".odt", ".rtf":
		return extractDOCX(content)
	case ".xlsx":
		return extractExcel(content)
	case ".pptx":
		return extractPPTX(content)
	case ".odp":
		return extractODP(content)
	case ".ods":
		return extractODS(content)
	case ".txt", ".md", ".rst", "":
		return extractPlain(content)
	default:
		// Unknown extension: treat as plain text
		return extractPlain(content)
	}
}
