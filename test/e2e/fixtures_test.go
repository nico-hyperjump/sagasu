package e2e

import (
	"strings"
	"testing"

	"github.com/hyperjump/sagasu/internal/extract"
)

func TestWriteMinimalFile_AllExtensionsExtractable(t *testing.T) {
	e := extract.NewExtractor()
	sample := "E2E searchable content"
	for _, ext := range SupportedFileExtensions {
		ext := ext
		t.Run(ext, func(t *testing.T) {
			content, err := WriteMinimalFile(ext, sample)
			if err != nil {
				t.Fatalf("WriteMinimalFile: %v", err)
			}
			if len(content) == 0 {
				t.Fatal("empty content")
			}
			got, err := e.ExtractBytes(content, ext)
			if err != nil {
				t.Fatalf("ExtractBytes: %v", err)
			}
			if !strings.Contains(got, sample) {
				t.Errorf("extracted text %q does not contain %q", got, sample)
			}
		})
	}
}
