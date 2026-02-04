package vector

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestMemoryIndex_AddSearch(t *testing.T) {
	idx, err := NewMemoryIndex(3)
	if err != nil {
		t.Fatal(err)
	}
	defer idx.Close()
	ctx := context.Background()

	vecs := [][]float32{
		{1, 0, 0},
		{0.9, 0.1, 0},
		{0, 1, 0},
	}
	ids := []string{"a", "b", "c"}
	if err := idx.Add(ctx, ids, vecs); err != nil {
		t.Fatal(err)
	}
	if idx.Size() != 3 {
		t.Errorf("Size=%d", idx.Size())
	}

	results, err := idx.Search(ctx, []float32{1, 0, 0}, 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	if results[0].ID != "a" {
		t.Errorf("top result should be a, got %s", results[0].ID)
	}
}

func TestMemoryIndex_Remove(t *testing.T) {
	idx, _ := NewMemoryIndex(2)
	ctx := context.Background()
	_ = idx.Add(ctx, []string{"x", "y"}, [][]float32{{1, 0}, {0, 1}})
	if err := idx.Remove(ctx, []string{"x"}); err != nil {
		t.Fatal(err)
	}
	if idx.Size() != 1 {
		t.Errorf("expected size 1, got %d", idx.Size())
	}
}

func TestMemoryIndex_SaveLoad(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	path := filepath.Join(dir, "idx.bin")

	idx, err := NewMemoryIndex(3)
	if err != nil {
		t.Fatal(err)
	}
	defer idx.Close()
	ids := []string{"a", "b", "c"}
	vecs := [][]float32{{1, 0, 0}, {0, 1, 0}, {0, 0, 1}}
	if err := idx.Add(ctx, ids, vecs); err != nil {
		t.Fatal(err)
	}
	if err := idx.Save(path); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("index file not created: %v", err)
	}

	idx2, err := NewMemoryIndex(3)
	if err != nil {
		t.Fatal(err)
	}
	defer idx2.Close()
	if err := idx2.Load(path); err != nil {
		t.Fatalf("Load: %v", err)
	}
	if idx2.Size() != 3 {
		t.Errorf("after Load size=%d, want 3", idx2.Size())
	}
	results, err := idx2.Search(ctx, []float32{0, 0, 1}, 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 || results[0].ID != "c" {
		t.Errorf("Search after Load: got %v", results)
	}
}

func TestMemoryIndex_LoadMissingFile(t *testing.T) {
	idx, err := NewMemoryIndex(2)
	if err != nil {
		t.Fatal(err)
	}
	defer idx.Close()
	if err := idx.Load("/nonexistent/path/index.bin"); err != nil {
		t.Errorf("Load missing file should not error: %v", err)
	}
	if idx.Size() != 0 {
		t.Errorf("Load missing file should leave index empty: size=%d", idx.Size())
	}
}

func TestMemoryIndex_SaveEmptyPath(t *testing.T) {
	idx, _ := NewMemoryIndex(2)
	defer idx.Close()
	if err := idx.Save(""); err != nil {
		t.Errorf("Save empty path should be no-op: %v", err)
	}
}

func TestMemoryIndex_Type(t *testing.T) {
	idx, err := NewMemoryIndex(2)
	if err != nil {
		t.Fatal(err)
	}
	defer idx.Close()
	if got := idx.Type(); got != "memory" {
		t.Errorf("Type() = %q, want %q", got, "memory")
	}
}
