package extract

import (
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
