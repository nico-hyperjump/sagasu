package extract

import (
	"archive/zip"
	"bytes"
	"fmt"
	"regexp"
	"strings"
)

// odpContentPath is the path to the main content inside an .odp zip (OpenDocument Presentation).
const odpContentPath = "content.xml"

// odpTextTags match OpenDocument text elements (with optional attributes). We use separate patterns
// so opening and closing tags match (e.g. <text:p>...</text:p> only).
var (
	odpTextP   = regexp.MustCompile(`<text:p[^>]*>([^<]*)</text:p>`)
	odpTextSpan = regexp.MustCompile(`<text:span[^>]*>([^<]*)</text:span>`)
	odpTextH   = regexp.MustCompile(`<text:h[^>]*>([^<]*)</text:h>`)
)

// extractODP extracts text from .odp bytes. ODP is a ZIP containing content.xml (OpenDocument).
// We extract all text from text:p, text:span, and text:h elements so content is searchable.
func extractODP(content []byte) (string, error) {
	zr, err := zip.NewReader(bytes.NewReader(content), int64(len(content)))
	if err != nil {
		return "", fmt.Errorf("extract ODP: not a zip: %w", err)
	}
	var contentXML []byte
	for _, f := range zr.File {
		if f.Name != odpContentPath {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return "", fmt.Errorf("extract ODP: open %s: %w", f.Name, err)
		}
		var buf bytes.Buffer
		if _, err := buf.ReadFrom(rc); err != nil {
			_ = rc.Close()
			return "", fmt.Errorf("extract ODP: read %s: %w", f.Name, err)
		}
		_ = rc.Close()
		contentXML = buf.Bytes()
		break
	}
	if contentXML == nil {
		return "", fmt.Errorf("extract ODP: %s not found", odpContentPath)
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
	appendMatches(odpTextP.FindAllStringSubmatch(s, -1))
	appendMatches(odpTextSpan.FindAllStringSubmatch(s, -1))
	appendMatches(odpTextH.FindAllStringSubmatch(s, -1))
	return strings.TrimSpace(b.String()), nil
}
