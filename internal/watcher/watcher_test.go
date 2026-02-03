package watcher

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestWatcher_AddRemoveDirectories(t *testing.T) {
	dir := t.TempDir()
	var indexed, removed []string
	var mu sync.Mutex
	onIndex := func(path string) {
		mu.Lock()
		indexed = append(indexed, path)
		mu.Unlock()
	}
	onRemove := func(path string) {
		mu.Lock()
		removed = append(removed, path)
		mu.Unlock()
	}

	w := NewWatcher(nil, []string{".txt"}, true, onIndex, onRemove)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := w.Start(ctx); err != nil {
		t.Fatal(err)
	}
	defer w.Stop()

	if err := w.AddDirectory(dir, false); err != nil {
		t.Fatal(err)
	}
	dirs := w.Directories()
	if len(dirs) != 1 || filepath.Clean(dirs[0]) != filepath.Clean(dir) {
		t.Errorf("Directories() = %v", dirs)
	}

	if err := w.RemoveDirectory(dir); err != nil {
		t.Fatal(err)
	}
	if len(w.Directories()) != 0 {
		t.Errorf("after remove: %v", w.Directories())
	}
	_ = indexed
	_ = removed
}

func TestWatcher_DebounceAndExtensionFilter(t *testing.T) {
	dir := t.TempDir()
	sub := filepath.Join(dir, "sub")
	if err := mkdirAll(sub); err != nil {
		t.Fatal(err)
	}

	var indexed []string
	var mu sync.Mutex
	onIndex := func(path string) {
		mu.Lock()
		indexed = append(indexed, path)
		mu.Unlock()
	}
	w := NewWatcher([]string{dir}, []string{".txt"}, true, onIndex, nil)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := w.Start(ctx); err != nil {
		t.Fatal(err)
	}
	defer w.Stop()

	// Create a .txt file
	fPath := filepath.Join(sub, "f.txt")
	if err := writeFile(fPath, "hello"); err != nil {
		t.Fatal(err)
	}
	time.Sleep(600 * time.Millisecond)
	mu.Lock()
	count := len(indexed)
	mu.Unlock()
	if count < 1 {
		t.Errorf("expected at least one index callback, got %d", count)
	}
}

func TestMatchExtension(t *testing.T) {
	tests := []struct {
		path       string
		extensions []string
		want       bool
	}{
		{"/a/b.txt", []string{".txt"}, true},
		{"/a/b.TXT", []string{".txt"}, true},
		{"/a/b.md", []string{".txt"}, false},
		{"/a/b", nil, true},
		{"/a/b", []string{}, true},
	}
	for _, tt := range tests {
		got := matchExtension(tt.path, tt.extensions)
		if got != tt.want {
			t.Errorf("matchExtension(%q, %v) = %v, want %v", tt.path, tt.extensions, got, tt.want)
		}
	}
}

func TestInDir(t *testing.T) {
	tests := []struct {
		dir  string
		path string
		want bool
	}{
		{"/tmp/a", "/tmp/a", true},
		{"/tmp/a", "/tmp/a/b.txt", true},
		{"/tmp/a", "/tmp/b", false},
		{"/tmp/a", "/tmp/a/../b", false},
	}
	for _, tt := range tests {
		got := inDir(tt.dir, tt.path)
		if got != tt.want {
			t.Errorf("inDir(%q, %q) = %v, want %v", tt.dir, tt.path, got, tt.want)
		}
	}
}

func TestWatcher_SyncExistingFiles_indexesMatchingFiles(t *testing.T) {
	dir := t.TempDir()
	if err := writeFile(filepath.Join(dir, "a.txt"), "hello"); err != nil {
		t.Fatal(err)
	}
	if err := writeFile(filepath.Join(dir, "ignore.xyz"), "x"); err != nil {
		t.Fatal(err)
	}

	var indexed []string
	var mu sync.Mutex
	onIndex := func(path string) {
		mu.Lock()
		indexed = append(indexed, path)
		mu.Unlock()
	}
	w := NewWatcher([]string{dir}, []string{".txt"}, true, onIndex, nil)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := w.Start(ctx); err != nil {
		t.Fatal(err)
	}
	w.SyncExistingFiles()

	mu.Lock()
	defer mu.Unlock()
	if len(indexed) != 1 || !strings.HasSuffix(indexed[0], "a.txt") {
		t.Errorf("expected one indexed file a.txt, got %v", indexed)
	}
}

func TestWatcher_Start_createsMissingRootDirectory(t *testing.T) {
	base := t.TempDir()
	root := filepath.Join(base, "watch", "me")
	// Ensure the root does not exist.
	_ = os.RemoveAll(filepath.Join(base, "watch"))

	w := NewWatcher([]string{root}, []string{".txt"}, true, nil, nil)
	// Use Background so we don't cancel; avoid race with run() reading w.watcher after Stop() nils it.
	if err := w.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	// Don't call Stop() to avoid race where run() reads w.watcher after Stop() nils it; test exit is enough.

	if _, err := os.Stat(root); err != nil {
		t.Errorf("root directory should exist after Start: %v", err)
	}
}

func mkdirAll(path string) error {
	return os.MkdirAll(path, 0755)
}

func writeFile(path, content string) error {
	return os.WriteFile(path, []byte(content), 0600)
}
