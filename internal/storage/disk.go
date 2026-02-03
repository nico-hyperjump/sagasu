// Package storage provides disk usage helpers for storage paths.
package storage

import (
	"os"
	"path/filepath"
)

// DiskUsageBytes returns the total size in bytes of the given paths.
// Each path may be a file or a directory (recursively summed).
// Missing or inaccessible paths are skipped (contribute 0); errors during walk are returned.
func DiskUsageBytes(paths ...string) (int64, error) {
	var total int64
	for _, p := range paths {
		if p == "" {
			continue
		}
		info, err := os.Stat(p)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return 0, err
		}
		if info.IsDir() {
			n, err := dirSize(p)
			if err != nil {
				return 0, err
			}
			total += n
		} else {
			total += info.Size()
		}
	}
	return total, nil
}

func dirSize(dir string) (int64, error) {
	var total int64
	err := filepath.Walk(dir, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info != nil && !info.IsDir() {
			total += info.Size()
		}
		return nil
	})
	return total, err
}
