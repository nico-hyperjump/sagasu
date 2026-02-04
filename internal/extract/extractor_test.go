package extract

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/xuri/excelize/v2"
)

func TestExtractBytes_plain(t *testing.T) {
	e := NewExtractor()
	content := []byte("Hello world\nLine 2")
	got, err := e.ExtractBytes(content, ".txt")
	if err != nil {
		t.Fatalf("ExtractBytes: %v", err)
	}
	if got != "Hello world\nLine 2" {
		t.Errorf("got %q", got)
	}
}

func TestExtractBytes_plainUTF8(t *testing.T) {
	e := NewExtractor()
	content := []byte("caf\xc3\xa9") // valid UTF-8
	got, err := e.ExtractBytes(content, ".md")
	if err != nil {
		t.Fatalf("ExtractBytes: %v", err)
	}
	if got != "café" {
		t.Errorf("got %q", got)
	}
}

func TestExtractBytes_plainInvalidUTF8(t *testing.T) {
	e := NewExtractor()
	content := []byte("hello\x80world") // invalid UTF-8
	got, err := e.ExtractBytes(content, ".rst")
	if err != nil {
		t.Fatalf("ExtractBytes: %v", err)
	}
	if got != "hello\uFFFDworld" {
		t.Errorf("got %q", got)
	}
}

func TestExtractBytes_excel(t *testing.T) {
	f := excelize.NewFile()
	defer f.Close()
	f.SetCellValue("Sheet1", "A1", "Title")
	f.SetCellValue("Sheet1", "A2", "Value 1")
	f.SetCellValue("Sheet1", "B2", "Value 2")
	var buf bytes.Buffer
	if _, err := f.WriteTo(&buf); err != nil {
		t.Fatalf("WriteTo: %v", err)
	}

	e := NewExtractor()
	got, err := e.ExtractBytes(buf.Bytes(), ".xlsx")
	if err != nil {
		t.Fatalf("ExtractBytes: %v", err)
	}
	if got != "Title\nValue 1\tValue 2" {
		t.Errorf("got %q", got)
	}
}

func TestExtract_plainFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")
	if err := os.WriteFile(path, []byte("File content"), 0600); err != nil {
		t.Fatal(err)
	}

	e := NewExtractor()
	got, err := e.Extract(path)
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}
	if got != "File content" {
		t.Errorf("got %q", got)
	}
}

func TestExtract_excelFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "data.xlsx")
	f := excelize.NewFile()
	f.SetCellValue("Sheet1", "A1", "Searchable text")
	if err := f.SaveAs(path); err != nil {
		t.Fatalf("SaveAs: %v", err)
	}
	f.Close()

	e := NewExtractor()
	got, err := e.Extract(path)
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}
	if got != "Searchable text" {
		t.Errorf("got %q", got)
	}
}

func TestExtract_nonexistent(t *testing.T) {
	e := NewExtractor()
	_, err := e.Extract("/nonexistent/path/file.txt")
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}

func TestExtractBytes_unknownExtension(t *testing.T) {
	e := NewExtractor()
	content := []byte("raw content")
	got, err := e.ExtractBytes(content, ".xyz")
	if err != nil {
		t.Fatalf("ExtractBytes: %v", err)
	}
	// Unknown extension falls back to plain
	if got != "raw content" {
		t.Errorf("got %q", got)
	}
}

// minimalDocx returns a minimal .docx zip bytes with word/document.xml containing the given text in <w:t> tags.
func minimalDocx(text string) []byte {
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	fw, _ := w.Create("word/document.xml")
	_, _ = fw.Write([]byte(`<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"><w:body><w:p><w:r><w:t>` + text + `</w:t></w:r></w:p></w:body></w:document>`))
	_ = w.Close()
	return buf.Bytes()
}

// minimalDocxWithContentTypes returns a .docx zip with [Content_Types].xml pointing to a custom document path.
func minimalDocxWithContentTypes(text, docPath string) []byte {
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	// Create [Content_Types].xml pointing to custom document path
	ct, _ := w.Create("[Content_Types].xml")
	_, _ = ct.Write([]byte(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">
<Override PartName="/` + docPath + `" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.document.main+xml"/>
</Types>`))
	// Create the document at the custom path
	fw, _ := w.Create(docPath)
	_, _ = fw.Write([]byte(`<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"><w:body><w:p><w:r><w:t>` + text + `</w:t></w:r></w:p></w:body></w:document>`))
	_ = w.Close()
	return buf.Bytes()
}

func TestExtractBytes_docx(t *testing.T) {
	e := NewExtractor()
	content := minimalDocx("Searchable docx content")
	got, err := e.ExtractBytes(content, ".docx")
	if err != nil {
		t.Fatalf("ExtractBytes: %v", err)
	}
	if got != "Searchable docx content" {
		t.Errorf("got %q", got)
	}
}

func TestExtractBytes_docxWithDocument2(t *testing.T) {
	e := NewExtractor()
	// Simulate a DOCX with word/document2.xml instead of word/document.xml
	content := minimalDocxWithContentTypes("Content from document2", "word/document2.xml")
	got, err := e.ExtractBytes(content, ".docx")
	if err != nil {
		t.Fatalf("ExtractBytes: %v", err)
	}
	if got != "Content from document2" {
		t.Errorf("got %q", got)
	}
}

func TestExtractBytes_docxContentTypesReversedOrder(t *testing.T) {
	e := NewExtractor()
	// Test with ContentType attribute before PartName
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	ct, _ := w.Create("[Content_Types].xml")
	_, _ = ct.Write([]byte(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">
<Override ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.document.main+xml" PartName="/word/document3.xml"/>
</Types>`))
	fw, _ := w.Create("word/document3.xml")
	_, _ = fw.Write([]byte(`<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"><w:body><w:p><w:r><w:t>Reversed order test</w:t></w:r></w:p></w:body></w:document>`))
	_ = w.Close()

	got, err := e.ExtractBytes(buf.Bytes(), ".docx")
	if err != nil {
		t.Fatalf("ExtractBytes: %v", err)
	}
	if got != "Reversed order test" {
		t.Errorf("got %q", got)
	}
}

// minimalPptx returns minimal .pptx zip bytes with one slide containing the given text in <a:t> tags.
func minimalPptx(text string) []byte {
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	fw, _ := w.Create("ppt/slides/slide1.xml")
	_, _ = fw.Write([]byte(`<p:sld xmlns:p="a" xmlns:a="b"><p:cSld><p:spTree><p:sp><p:txBody><a:p><a:r><a:t>` + text + `</a:t></a:r></a:p></p:txBody></p:sp></p:spTree></p:cSld></p:sld>`))
	_ = w.Close()
	return buf.Bytes()
}

func TestExtractBytes_pptx(t *testing.T) {
	e := NewExtractor()
	content := minimalPptx("Searchable pptx content")
	got, err := e.ExtractBytes(content, ".pptx")
	if err != nil {
		t.Fatalf("ExtractBytes: %v", err)
	}
	if got != "Searchable pptx content" {
		t.Errorf("got %q", got)
	}
}

func TestExtractBytes_pptxMultipleSlides(t *testing.T) {
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	slide1, _ := w.Create("ppt/slides/slide1.xml")
	_, _ = slide1.Write([]byte(`<p:sld><p:cSld><p:spTree><p:sp><p:txBody><a:p><a:r><a:t>First slide</a:t></a:r></a:p></p:txBody></p:sp></p:spTree></p:cSld></p:sld>`))
	slide2, _ := w.Create("ppt/slides/slide2.xml")
	_, _ = slide2.Write([]byte(`<p:sld><p:cSld><p:spTree><p:sp><p:txBody><a:p><a:r><a:t>Second slide</a:t></a:r></a:p></p:txBody></p:sp></p:spTree></p:cSld></p:sld>`))
	_ = w.Close()

	e := NewExtractor()
	got, err := e.ExtractBytes(buf.Bytes(), ".pptx")
	if err != nil {
		t.Fatalf("ExtractBytes: %v", err)
	}
	if got != "First slide Second slide" {
		t.Errorf("got %q", got)
	}
}

// minimalOdp returns minimal .odp zip bytes with content.xml containing text in text:p/text:span/text:h.
func minimalOdp(contentXML string) []byte {
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	fw, _ := w.Create("content.xml")
	_, _ = fw.Write([]byte(contentXML))
	_ = w.Close()
	return buf.Bytes()
}

func TestExtractBytes_odp(t *testing.T) {
	contentXML := `<office:document><office:body><draw:page><draw:text-box><text:p>Searchable odp content</text:p></draw:text-box></draw:page></office:body></office:document>`
	e := NewExtractor()
	got, err := e.ExtractBytes(minimalOdp(contentXML), ".odp")
	if err != nil {
		t.Fatalf("ExtractBytes: %v", err)
	}
	if got != "Searchable odp content" {
		t.Errorf("got %q", got)
	}
}

func TestExtractBytes_odpTextH(t *testing.T) {
	contentXML := `<office:document><office:body><draw:page><text:h>Slide title</text:h><text:p>Body text</text:p></draw:page></office:body></office:document>`
	e := NewExtractor()
	got, err := e.ExtractBytes(minimalOdp(contentXML), ".odp")
	if err != nil {
		t.Fatalf("ExtractBytes: %v", err)
	}
	// Order is p, span, h so we get "Body text" then "Slide title"
	if got != "Body text Slide title" {
		t.Errorf("got %q", got)
	}
}

// minimalOds returns minimal .ods zip bytes with content.xml containing text in text:p/text:span.
func minimalOds(contentXML string) []byte {
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	fw, _ := w.Create("content.xml")
	_, _ = fw.Write([]byte(contentXML))
	_ = w.Close()
	return buf.Bytes()
}

func TestExtractBytes_ods(t *testing.T) {
	contentXML := `<office:document><office:body><table:table><table:table-row><table:table-cell><text:p>Searchable ods content</text:p></table:table-cell></table:table-row></table:table></office:body></office:document>`
	e := NewExtractor()
	got, err := e.ExtractBytes(minimalOds(contentXML), ".ods")
	if err != nil {
		t.Fatalf("ExtractBytes: %v", err)
	}
	if got != "Searchable ods content" {
		t.Errorf("got %q", got)
	}
}

func TestExtractBytes_odsMultipleCells(t *testing.T) {
	contentXML := `<office:document><office:body><table:table><table:table-row><table:table-cell><text:p>Cell A</text:p></table:table-cell><table:table-cell><text:span>Cell B</text:span></table:table-cell></table:table-row></table:table></office:body></office:document>`
	e := NewExtractor()
	got, err := e.ExtractBytes(minimalOds(contentXML), ".ods")
	if err != nil {
		t.Fatalf("ExtractBytes: %v", err)
	}
	if got != "Cell A Cell B" {
		t.Errorf("got %q", got)
	}
}

func TestExtractBytes_pptxEmpty(t *testing.T) {
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	_, _ = w.Create("ppt/slides/other.xml")
	_, _ = w.Create("docProps/core.xml")
	_ = w.Close()
	e := NewExtractor()
	got, err := e.ExtractBytes(buf.Bytes(), ".pptx")
	if err != nil {
		t.Fatalf("ExtractBytes: %v", err)
	}
	if got != "" {
		t.Errorf("got %q", got)
	}
}

func TestExtract_pptxFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "deck.pptx")
	if err := os.WriteFile(path, minimalPptx("Searchable from file"), 0600); err != nil {
		t.Fatal(err)
	}
	e := NewExtractor()
	got, err := e.Extract(path)
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}
	if got != "Searchable from file" {
		t.Errorf("got %q", got)
	}
}

func TestExtract_odpFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "pres.odp")
	content := minimalOdp(`<office:document><office:body><draw:page><text:p>From file</text:p></draw:page></office:body></office:document>`)
	if err := os.WriteFile(path, content, 0600); err != nil {
		t.Fatal(err)
	}
	e := NewExtractor()
	got, err := e.Extract(path)
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}
	if got != "From file" {
		t.Errorf("got %q", got)
	}
}

func TestExtract_odsFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sheet.ods")
	content := minimalOds(`<office:document><office:body><table:table><table:table-row><table:table-cell><text:p>From file</text:p></table:table-cell></table:table-row></table:table></office:body></office:document>`)
	if err := os.WriteFile(path, content, 0600); err != nil {
		t.Fatal(err)
	}
	e := NewExtractor()
	got, err := e.Extract(path)
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}
	if got != "From file" {
		t.Errorf("got %q", got)
	}
}

func TestExtract_pptxNotZip(t *testing.T) {
	e := NewExtractor()
	_, err := e.ExtractBytes([]byte("not a zip"), ".pptx")
	if err == nil {
		t.Error("expected error for invalid pptx")
	}
}

func TestExtract_odpContentNotFound(t *testing.T) {
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	_, _ = w.Create("other.xml")
	_ = w.Close()
	e := NewExtractor()
	_, err := e.ExtractBytes(buf.Bytes(), ".odp")
	if err == nil {
		t.Error("expected error when content.xml missing")
	}
}

func TestExtract_odsContentNotFound(t *testing.T) {
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	_, _ = w.Create("other.xml")
	_ = w.Close()
	e := NewExtractor()
	_, err := e.ExtractBytes(buf.Bytes(), ".ods")
	if err == nil {
		t.Error("expected error when content.xml missing")
	}
}
