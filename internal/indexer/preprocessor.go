package indexer

import (
	"strings"
	"unicode"
)

// Preprocess normalizes text for indexing (trim, collapse whitespace).
func Preprocess(text string) string {
	text = strings.TrimSpace(text)
	var b strings.Builder
	wasSpace := false
	for _, r := range text {
		if unicode.IsSpace(r) {
			if !wasSpace {
				b.WriteRune(' ')
				wasSpace = true
			}
		} else {
			b.WriteRune(r)
			wasSpace = false
		}
	}
	return b.String()
}
