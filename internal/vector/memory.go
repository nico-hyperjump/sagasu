// Package vector provides an in-memory vector index for testing and when FAISS is not available.
package vector

import (
	"context"
	"encoding/binary"
	"fmt"
	"math"
	"os"
	"path/filepath"
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

// Type returns the index type identifier.
func (m *MemoryIndex) Type() string {
	return string(IndexTypeMemory)
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

// Save persists the index to path. Directory is created if needed. Format: dimension (4), n (4),
// then per vector: idLen (4), id bytes, vector (dimension*4 bytes).
func (m *MemoryIndex) Save(path string) error {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if path == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("create index dir: %w", err)
	}
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create index file: %w", err)
	}
	defer f.Close()
	if err := binary.Write(f, binary.LittleEndian, uint32(m.dimensions)); err != nil {
		return fmt.Errorf("write dimensions: %w", err)
	}
	n := uint32(len(m.ids))
	if err := binary.Write(f, binary.LittleEndian, n); err != nil {
		return fmt.Errorf("write count: %w", err)
	}
	for i, id := range m.ids {
		idBytes := []byte(id)
		if err := binary.Write(f, binary.LittleEndian, uint32(len(idBytes))); err != nil {
			return fmt.Errorf("write id len: %w", err)
		}
		if _, err := f.Write(idBytes); err != nil {
			return fmt.Errorf("write id: %w", err)
		}
		if _, err := f.Write(float32SliceToBytes(m.vectors[i])); err != nil {
			return fmt.Errorf("write vector: %w", err)
		}
	}
	return nil
}

// Load reads the index from path and replaces the in-memory contents. Dimensions must match.
// If the file does not exist, no error is returned and the index is unchanged.
func (m *MemoryIndex) Load(path string) error {
	if path == "" {
		return nil
	}
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("open index file: %w", err)
	}
	defer f.Close()
	var dim, n uint32
	if err := binary.Read(f, binary.LittleEndian, &dim); err != nil {
		return fmt.Errorf("read dimensions: %w", err)
	}
	if int(dim) != m.dimensions {
		return fmt.Errorf("dimension mismatch: file has %d, index expects %d", dim, m.dimensions)
	}
	if err := binary.Read(f, binary.LittleEndian, &n); err != nil {
		return fmt.Errorf("read count: %w", err)
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ids = make([]string, 0, n)
	m.vectors = make([][]float32, 0, n)
	buf := make([]byte, m.dimensions*4)
	for i := uint32(0); i < n; i++ {
		var idLen uint32
		if err := binary.Read(f, binary.LittleEndian, &idLen); err != nil {
			return fmt.Errorf("read id len: %w", err)
		}
		idBytes := make([]byte, idLen)
		if _, err := f.Read(idBytes); err != nil {
			return fmt.Errorf("read id: %w", err)
		}
		if _, err := f.Read(buf); err != nil {
			return fmt.Errorf("read vector: %w", err)
		}
		m.ids = append(m.ids, string(idBytes))
		m.vectors = append(m.vectors, bytesToFloat32Slice(buf))
	}
	return nil
}

func float32SliceToBytes(s []float32) []byte {
	const size = 4
	out := make([]byte, len(s)*size)
	for i, v := range s {
		binary.LittleEndian.PutUint32(out[i*size:(i+1)*size], math.Float32bits(v))
	}
	return out
}

func bytesToFloat32Slice(b []byte) []float32 {
	const size = 4
	out := make([]float32, len(b)/size)
	for i := range out {
		out[i] = math.Float32frombits(binary.LittleEndian.Uint32(b[i*size : (i+1)*size]))
	}
	return out
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
