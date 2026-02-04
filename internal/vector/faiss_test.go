//go:build faiss && cgo
// +build faiss,cgo

package vector

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestFAISSIndex_AddSearch(t *testing.T) {
	idx, err := NewFAISSIndex(3)
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
		t.Errorf("Size=%d, want 3", idx.Size())
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

func TestFAISSIndex_SearchEmpty(t *testing.T) {
	idx, err := NewFAISSIndex(3)
	if err != nil {
		t.Fatal(err)
	}
	defer idx.Close()
	ctx := context.Background()

	results, err := idx.Search(ctx, []float32{1, 0, 0}, 10)
	if err != nil {
		t.Fatal(err)
	}
	if results != nil && len(results) != 0 {
		t.Errorf("expected empty results, got %d", len(results))
	}
}

func TestFAISSIndex_Remove(t *testing.T) {
	idx, err := NewFAISSIndex(2)
	if err != nil {
		t.Fatal(err)
	}
	defer idx.Close()
	ctx := context.Background()

	_ = idx.Add(ctx, []string{"x", "y"}, [][]float32{{1, 0}, {0, 1}})
	if err := idx.Remove(ctx, []string{"x"}); err != nil {
		t.Fatal(err)
	}
	if idx.Size() != 1 {
		t.Errorf("expected size 1, got %d", idx.Size())
	}

	// Search should not return removed item
	results, err := idx.Search(ctx, []float32{1, 0}, 10)
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range results {
		if r.ID == "x" {
			t.Error("removed item 'x' should not appear in search results")
		}
	}
}

func TestFAISSIndex_SaveLoad(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	path := filepath.Join(dir, "idx")

	idx, err := NewFAISSIndex(3)
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
	if _, err := os.Stat(path + ".faiss"); err != nil {
		t.Fatalf("faiss index file not created: %v", err)
	}
	if _, err := os.Stat(path + ".idmap"); err != nil {
		t.Fatalf("idmap file not created: %v", err)
	}

	idx2, err := NewFAISSIndex(3)
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

func TestFAISSIndex_LoadMissingFile(t *testing.T) {
	idx, err := NewFAISSIndex(2)
	if err != nil {
		t.Fatal(err)
	}
	defer idx.Close()
	if err := idx.Load("/nonexistent/path/index"); err != nil {
		t.Errorf("Load missing file should not error: %v", err)
	}
	if idx.Size() != 0 {
		t.Errorf("Load missing file should leave index empty: size=%d", idx.Size())
	}
}

func TestFAISSIndex_SaveEmptyPath(t *testing.T) {
	idx, err := NewFAISSIndex(2)
	if err != nil {
		t.Fatal(err)
	}
	defer idx.Close()
	if err := idx.Save(""); err != nil {
		t.Errorf("Save empty path should be no-op: %v", err)
	}
}

func TestFAISSIndex_DimensionMismatch(t *testing.T) {
	idx, err := NewFAISSIndex(3)
	if err != nil {
		t.Fatal(err)
	}
	defer idx.Close()
	ctx := context.Background()

	// Add with wrong dimension
	err = idx.Add(ctx, []string{"a"}, [][]float32{{1, 0}})
	if err == nil {
		t.Error("expected error for dimension mismatch on Add")
	}

	// Search with wrong dimension
	_ = idx.Add(ctx, []string{"a"}, [][]float32{{1, 0, 0}})
	_, err = idx.Search(ctx, []float32{1, 0}, 1)
	if err == nil {
		t.Error("expected error for dimension mismatch on Search")
	}
}

func TestFAISSIndex_InvalidDimension(t *testing.T) {
	_, err := NewFAISSIndex(0)
	if err == nil {
		t.Error("expected error for zero dimension")
	}

	_, err = NewFAISSIndex(-1)
	if err == nil {
		t.Error("expected error for negative dimension")
	}
}

func TestFAISSIndex_LengthMismatch(t *testing.T) {
	idx, err := NewFAISSIndex(2)
	if err != nil {
		t.Fatal(err)
	}
	defer idx.Close()
	ctx := context.Background()

	err = idx.Add(ctx, []string{"a", "b"}, [][]float32{{1, 0}})
	if err == nil {
		t.Error("expected error for ids/vectors length mismatch")
	}
}

func TestFAISSIndex_AddEmpty(t *testing.T) {
	idx, err := NewFAISSIndex(2)
	if err != nil {
		t.Fatal(err)
	}
	defer idx.Close()
	ctx := context.Background()

	err = idx.Add(ctx, []string{}, [][]float32{})
	if err != nil {
		t.Errorf("Add empty should succeed: %v", err)
	}
	if idx.Size() != 0 {
		t.Errorf("Size should be 0, got %d", idx.Size())
	}
}

func TestFAISSIndex_Type(t *testing.T) {
	idx, err := NewFAISSIndex(2)
	if err != nil {
		t.Fatal(err)
	}
	defer idx.Close()
	if got := idx.Type(); got != "faiss" {
		t.Errorf("Type() = %q, want %q", got, "faiss")
	}
}
