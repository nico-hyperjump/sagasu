// Package fileid provides a deterministic document ID from a file path for watched files.
package fileid

import (
	"crypto/sha256"
	"encoding/hex"
	"path/filepath"
)

const prefix = "file:"

// FileDocID returns a stable document ID for the given absolute path.
// Same path always yields the same ID. Used for index/update/delete by path.
func FileDocID(absolutePath string) string {
	normalized := filepath.Clean(absolutePath)
	hash := sha256.Sum256([]byte(normalized))
	return prefix + hex.EncodeToString(hash[:])
}
