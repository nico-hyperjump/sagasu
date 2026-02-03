package extract

import (
	"archive/zip"
	"bytes"
	"fmt"
	"regexp"
	"strings"
)

// pptxSlidePathPrefix is the path prefix for slide XML files inside a .pptx zip.
const pptxSlidePathPrefix = "ppt/slides/slide"

// atTag matches <a:t>text</a:t> or <a:t xml:space="preserve">text</a:t> (and any other attributes).
var atTag = regexp.MustCompile(`<a:t[^>]*>([^<]*)</a:t>`)

// extractPPTX extracts text from .pptx bytes. PPTX is a ZIP containing ppt/slides/slideN.xml
// (Office Open XML). We extract all <a:t>...</a:t> text nodes from each slide so content is searchable.
func extractPPTX(content []byte) (string, error) {
	zr, err := zip.NewReader(bytes.NewReader(content), int64(len(content)))
	if err != nil {
		return "", fmt.Errorf("extract PPTX: not a zip: %w", err)
	}
	var buf strings.Builder
	for _, f := range zr.File {
		if !strings.HasPrefix(f.Name, pptxSlidePathPrefix) || !strings.HasSuffix(f.Name, ".xml") {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return "", fmt.Errorf("extract PPTX: open %s: %w", f.Name, err)
		}
		var slideBuf bytes.Buffer
		if _, err := slideBuf.ReadFrom(rc); err != nil {
			_ = rc.Close()
			return "", fmt.Errorf("extract PPTX: read %s: %w", f.Name, err)
		}
		_ = rc.Close()
		parts := atTag.FindAllStringSubmatch(slideBuf.String(), -1)
		for _, p := range parts {
			if buf.Len() > 0 {
				buf.WriteByte(' ')
			}
			buf.WriteString(strings.TrimSpace(p[1]))
		}
	}
	return strings.TrimSpace(buf.String()), nil
}
