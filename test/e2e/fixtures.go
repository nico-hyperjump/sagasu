// Package e2e provides end-to-end tests; this file builds minimal binary files for supported types.
package e2e

import (
	"archive/zip"
	"bytes"

	"github.com/xuri/excelize/v2"
)

// SupportedFileExtensions is the list of file extensions used in E2E file-based tests.
// Covers: plain text (.txt, .md, .rst), OOXML (.docx, .xlsx, .pptx), OpenDocument (.odp, .ods).
// The extractor also supports .pdf, .odt, .rtf; PDF is not generated here (no minimal PDF with
// extractable text); .odt/.rtf use the same code path as .docx.
var SupportedFileExtensions = []string{
	".txt", ".md", ".rst",
	".docx", ".xlsx", ".pptx", ".odp", ".ods",
}

// WriteMinimalFile writes a minimal file of the given extension with the given text content
// to the provided path (caller must create the file). Returns the content to write.
// For plain types (.txt, .md, .rst) the content is the raw text; for binary types it is the file bytes.
func WriteMinimalFile(ext, text string) ([]byte, error) {
	switch ext {
	case ".txt", ".md", ".rst":
		return []byte(text), nil
	case ".docx":
		return minimalDocx(text), nil
	case ".pptx":
		return minimalPptx(text), nil
	case ".odp":
		return minimalOdp(text), nil
	case ".ods":
		return minimalOds(text), nil
	case ".xlsx":
		return minimalXlsx(text), nil
	default:
		return []byte(text), nil
	}
}

func minimalDocx(text string) []byte {
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	fw, _ := w.Create("word/document.xml")
	_, _ = fw.Write([]byte(`<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"><w:body><w:p><w:r><w:t>` + text + `</w:t></w:r></w:p></w:body></w:document>`))
	_ = w.Close()
	return buf.Bytes()
}

func minimalPptx(text string) []byte {
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	fw, _ := w.Create("ppt/slides/slide1.xml")
	_, _ = fw.Write([]byte(`<p:sld xmlns:p="a" xmlns:a="b"><p:cSld><p:spTree><p:sp><p:txBody><a:p><a:r><a:t>` + text + `</a:t></a:r></a:p></p:txBody></p:sp></p:spTree></p:cSld></p:sld>`))
	_ = w.Close()
	return buf.Bytes()
}

func minimalOdp(text string) []byte {
	contentXML := `<office:document><office:body><draw:page><draw:text-box><text:p>` + text + `</text:p></draw:text-box></draw:page></office:body></office:document>`
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	fw, _ := w.Create("content.xml")
	_, _ = fw.Write([]byte(contentXML))
	_ = w.Close()
	return buf.Bytes()
}

func minimalOds(text string) []byte {
	contentXML := `<office:document><office:body><table:table><table:table-row><table:table-cell><text:p>` + text + `</text:p></table:table-cell></table:table-row></table:table></office:body></office:document>`
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	fw, _ := w.Create("content.xml")
	_, _ = fw.Write([]byte(contentXML))
	_ = w.Close()
	return buf.Bytes()
}

func minimalXlsx(text string) []byte {
	f := excelize.NewFile()
	defer f.Close()
	f.SetCellValue("Sheet1", "A1", text)
	var buf bytes.Buffer
	_, _ = f.WriteTo(&buf)
	return buf.Bytes()
}
