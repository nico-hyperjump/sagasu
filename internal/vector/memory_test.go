package vector

import (
	"context"
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
