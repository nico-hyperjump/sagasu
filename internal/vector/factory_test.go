package vector

import (
	"context"
	"testing"
)

func TestNewVectorIndex_Memory(t *testing.T) {
	idx, err := NewVectorIndex("memory", 3)
	if err != nil {
		t.Fatalf("NewVectorIndex(memory): %v", err)
	}
	defer idx.Close()

	// Verify it's a working MemoryIndex
	ctx := context.Background()
	err = idx.Add(ctx, []string{"a"}, [][]float32{{1, 0, 0}})
	if err != nil {
		t.Fatalf("Add: %v", err)
	}
	if idx.Size() != 1 {
		t.Errorf("Size=%d, want 1", idx.Size())
	}
}

func TestNewVectorIndex_Empty(t *testing.T) {
	// Empty string should default to memory
	idx, err := NewVectorIndex("", 3)
	if err != nil {
		t.Fatalf("NewVectorIndex(''): %v", err)
	}
	defer idx.Close()

	if idx.Size() != 0 {
		t.Errorf("Size=%d, want 0", idx.Size())
	}
}

func TestNewVectorIndex_Unknown(t *testing.T) {
	_, err := NewVectorIndex("unknown", 3)
	if err == nil {
		t.Error("expected error for unknown index type")
	}
}

func TestNewVectorIndex_InvalidDimension(t *testing.T) {
	_, err := NewVectorIndex("memory", 0)
	if err == nil {
		t.Error("expected error for zero dimension")
	}
}

func TestIsFAISSAvailable(t *testing.T) {
	// This test just verifies the function doesn't panic
	// The result depends on build tags
	available := IsFAISSAvailable()
	t.Logf("FAISS available: %v", available)
}

func TestNewVectorIndex_FAISS(t *testing.T) {
	if !IsFAISSAvailable() {
		t.Skip("FAISS not available (build with -tags=faiss)")
	}

	idx, err := NewVectorIndex("faiss", 3)
	if err != nil {
		t.Fatalf("NewVectorIndex(faiss): %v", err)
	}
	defer idx.Close()

	// Verify it's a working FAISSIndex
	ctx := context.Background()
	err = idx.Add(ctx, []string{"a"}, [][]float32{{1, 0, 0}})
	if err != nil {
		t.Fatalf("Add: %v", err)
	}
	if idx.Size() != 1 {
		t.Errorf("Size=%d, want 1", idx.Size())
	}
}
