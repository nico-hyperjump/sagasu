// Package vector provides an in-memory vector index for testing and when FAISS is not available.
package vector

import (
	"context"
	"fmt"
	"math"
	"sort"
	"sync"
)

// MemoryIndex is an in-memory vector index using brute-force inner product search.
// Suitable for tests and small datasets when FAISS is not available.
type MemoryIndex struct {
	dimensions int
	ids        []string
	vectors    [][]float32
	mu         sync.RWMutex
}

// NewMemoryIndex creates an in-memory vector index with the given dimension.
func NewMemoryIndex(dimensions int) (*MemoryIndex, error) {
	if dimensions <= 0 {
		return nil, fmt.Errorf("dimensions must be positive")
	}
	return &MemoryIndex{
		dimensions: dimensions,
		ids:        make([]string, 0),
		vectors:    make([][]float32, 0),
	}, nil
}

// Add appends vectors with the given IDs.
func (m *MemoryIndex) Add(ctx context.Context, ids []string, vectors [][]float32) error {
	if len(ids) != len(vectors) {
		return fmt.Errorf("ids and vectors length mismatch")
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	for i, id := range ids {
		if len(vectors[i]) != m.dimensions {
			return fmt.Errorf("vector dimension mismatch: got %d, expected %d", len(vectors[i]), m.dimensions)
		}
		vec := make([]float32, m.dimensions)
		copy(vec, vectors[i])
		m.ids = append(m.ids, id)
		m.vectors = append(m.vectors, vec)
	}
	return nil
}

// Search returns the top-k vectors by inner product (assumes normalized vectors = cosine similarity).
func (m *MemoryIndex) Search(ctx context.Context, query []float32, k int) ([]*VectorResult, error) {
	if len(query) != m.dimensions {
		return nil, fmt.Errorf("query dimension mismatch: got %d, expected %d", len(query), m.dimensions)
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	if k <= 0 || len(m.ids) == 0 {
		return nil, nil
	}
	type scored struct {
		id    string
		score float64
	}
	scores := make([]scored, len(m.ids))
	for i, vec := range m.vectors {
		var dot float64
		for j := 0; j < m.dimensions; j++ {
			dot += float64(query[j] * vec[j])
		}
		scores[i] = scored{id: m.ids[i], score: dot}
	}
	sort.Slice(scores, func(i, j int) bool { return scores[i].score > scores[j].score })
	if k > len(scores) {
		k = len(scores)
	}
	result := make([]*VectorResult, k)
	for i := 0; i < k; i++ {
		result[i] = &VectorResult{ID: scores[i].id, Score: scores[i].score}
	}
	return result, nil
}

// Remove removes vectors by ID (marks as removed by clearing ID; search skips zero IDs).
// For MemoryIndex we actually remove the entries by rebuilding the slice.
func (m *MemoryIndex) Remove(ctx context.Context, ids []string) error {
	removeSet := make(map[string]bool)
	for _, id := range ids {
		removeSet[id] = true
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	newIDs := make([]string, 0, len(m.ids))
	newVectors := make([][]float32, 0, len(m.vectors))
	for i, id := range m.ids {
		if !removeSet[id] {
			newIDs = append(newIDs, id)
			newVectors = append(newVectors, m.vectors[i])
		}
	}
	m.ids = newIDs
	m.vectors = newVectors
	return nil
}

// Save is a no-op for MemoryIndex (optionally could persist to file).
func (m *MemoryIndex) Save(path string) error {
	return nil
}

// Load is a no-op for MemoryIndex.
func (m *MemoryIndex) Load(path string) error {
	return nil
}

// Size returns the number of vectors in the index.
func (m *MemoryIndex) Size() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.ids)
}

// Close is a no-op for MemoryIndex.
func (m *MemoryIndex) Close() error {
	return nil
}

// CosineSimilarity returns cosine similarity between two normalized vectors (0-1).
func CosineSimilarity(a, b []float32) float64 {
	if len(a) != len(b) || len(a) == 0 {
		return 0
	}
	var dot float64
	for i := range a {
		dot += float64(a[i] * b[i])
	}
	return math.Max(0, math.Min(1, dot))
}
