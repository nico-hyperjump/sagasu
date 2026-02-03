package storage

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDiskUsageBytes(t *testing.T) {
	dir := t.TempDir()

	// Single file
	f1 := filepath.Join(dir, "f1.txt")
	if err := os.WriteFile(f1, []byte("hello"), 0644); err != nil {
		t.Fatal(err)
	}
	got, err := DiskUsageBytes(f1)
	if err != nil {
		t.Fatal(err)
	}
	if got != 5 {
		t.Errorf("single file: got %d bytes, want 5", got)
	}

	// Directory
	sub := filepath.Join(dir, "sub")
	if err := os.Mkdir(sub, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sub, "a"), []byte("ab"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sub, "b"), []byte("c"), 0644); err != nil {
		t.Fatal(err)
	}
	got, err = DiskUsageBytes(sub)
	if err != nil {
		t.Fatal(err)
	}
	if got != 3 {
		t.Errorf("dir: got %d bytes, want 3", got)
	}

	// Multiple paths (file + dir)
	got, err = DiskUsageBytes(f1, sub)
	if err != nil {
		t.Fatal(err)
	}
	if got != 8 {
		t.Errorf("file+dir: got %d bytes, want 8", got)
	}

	// Missing path is skipped
	got, err = DiskUsageBytes(f1, filepath.Join(dir, "nonexistent"), sub)
	if err != nil {
		t.Fatal(err)
	}
	if got != 8 {
		t.Errorf("with missing: got %d bytes, want 8", got)
	}

	// Empty path is skipped
	got, err = DiskUsageBytes("", f1)
	if err != nil {
		t.Fatal(err)
	}
	if got != 5 {
		t.Errorf("with empty path: got %d bytes, want 5", got)
	}
}
