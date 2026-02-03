// Package vector provides vector index and similarity search.
package vector

import "context"

// VectorIndex defines vector storage and similarity search.
type VectorIndex interface {
	Add(ctx context.Context, ids []string, vectors [][]float32) error
	Search(ctx context.Context, query []float32, k int) ([]*VectorResult, error)
	Remove(ctx context.Context, ids []string) error
	Save(path string) error
	Load(path string) error
	Size() int
	Close() error
}

// VectorResult is a single vector search hit (ID is chunk ID for semantic index).
type VectorResult struct {
	ID    string
	Score float64 // Inner product or cosine similarity (0-1 for normalized)
}
