package extract

import (
	"strings"
	"unicode/utf8"
)

// extractPlain returns content as string, validating it is valid UTF-8.
// Invalid UTF-8 sequences are replaced with the replacement character.
func extractPlain(content []byte) (string, error) {
	if !utf8.Valid(content) {
		content = []byte(strings.ToValidUTF8(string(content), "\ufffd"))
	}
	return string(content), nil
}
