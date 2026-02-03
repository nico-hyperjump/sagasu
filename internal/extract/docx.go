package extract

import (
	"fmt"

	"github.com/lu4p/cat"
)

func extractDOCX(content []byte) (string, error) {
	text, err := cat.FromBytes(content)
	if err != nil {
		return "", fmt.Errorf("extract DOCX: %w", err)
	}
	return text, nil
}
