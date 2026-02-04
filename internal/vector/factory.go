// Package vector provides vector index implementations and a factory for creating them.
package vector

import "fmt"

// IndexType represents the type of vector index to use.
type IndexType string

const (
	// IndexTypeMemory uses in-memory brute-force search. Good for small datasets (<10k vectors).
	IndexTypeMemory IndexType = "memory"
	// IndexTypeFAISS uses FAISS for efficient ANN search. Good for large datasets.
	// Requires FAISS library and build tag -tags=faiss.
	IndexTypeFAISS IndexType = "faiss"
)

// NewVectorIndex creates a vector index of the specified type.
// Supported types: "memory" (default), "faiss".
// FAISS requires building with -tags=faiss and having FAISS library installed.
func NewVectorIndex(indexType string, dimensions int) (VectorIndex, error) {
	switch IndexType(indexType) {
	case IndexTypeMemory, "":
		return NewMemoryIndex(dimensions)
	case IndexTypeFAISS:
		return NewFAISSIndex(dimensions)
	default:
		return nil, fmt.Errorf("unknown index type: %s (supported: memory, faiss)", indexType)
	}
}

// IsFAISSAvailable returns true if FAISS support is compiled in.
// This is determined by the build tag -tags=faiss.
func IsFAISSAvailable() bool {
	idx, err := NewFAISSIndex(1)
	if err != nil {
		return false
	}
	_ = idx.Close()
	return true
}
