//go:build faiss && cgo
// +build faiss,cgo

// Package vector provides FAISS-based vector index for production scale.
package vector

/*
#cgo CFLAGS: -I/opt/homebrew/include -I/usr/local/include
#cgo LDFLAGS: -L/opt/homebrew/lib -L/usr/local/lib -lfaiss_c

#include <stdlib.h>
#include <faiss/c_api/Index_c.h>
#include <faiss/c_api/IndexFlat_c.h>
#include <faiss/c_api/index_io_c.h>
#include <faiss/c_api/error_c.h>
*/
import "C"

import (
	"context"
	"encoding/gob"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"unsafe"
)

// FAISSIndex is a vector index using FAISS for efficient similarity search.
// It uses IndexFlatIP (inner product) for normalized vectors, which is equivalent
// to cosine similarity. Suitable for production workloads with large datasets.
type FAISSIndex struct {
	index      *C.FaissIndexFlatIP
	dimensions int
	idToIntID  map[string]int64 // string ID -> FAISS internal int64 ID
	intIDToID  map[int64]string // FAISS internal int64 ID -> string ID
	nextID     int64
	mu         sync.RWMutex
}

// NewFAISSIndex creates a FAISS index with the given dimension using inner product.
func NewFAISSIndex(dimensions int) (*FAISSIndex, error) {
	if dimensions <= 0 {
		return nil, fmt.Errorf("dimensions must be positive")
	}

	var index *C.FaissIndexFlatIP
	ret := C.faiss_IndexFlatIP_new_with(&index, C.idx_t(dimensions))
	if ret != 0 {
		return nil, fmt.Errorf("failed to create FAISS index: %s", faissLastError())
	}

	return &FAISSIndex{
		index:      index,
		dimensions: dimensions,
		idToIntID:  make(map[string]int64),
		intIDToID:  make(map[int64]string),
		nextID:     0,
	}, nil
}

// faissLastError returns the last FAISS error message.
func faissLastError() string {
	cErr := C.faiss_get_last_error()
	if cErr == nil {
		return "unknown error"
	}
	return C.GoString(cErr)
}

// Add appends vectors with the given IDs.
func (f *FAISSIndex) Add(ctx context.Context, ids []string, vectors [][]float32) error {
	if len(ids) != len(vectors) {
		return fmt.Errorf("ids and vectors length mismatch")
	}
	if len(ids) == 0 {
		return nil
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	// Flatten vectors into contiguous array for FAISS
	n := len(vectors)
	flatVectors := make([]float32, n*f.dimensions)
	for i, vec := range vectors {
		if len(vec) != f.dimensions {
			return fmt.Errorf("vector dimension mismatch: got %d, expected %d", len(vec), f.dimensions)
		}
		copy(flatVectors[i*f.dimensions:(i+1)*f.dimensions], vec)
	}

	// Add to FAISS index
	ret := C.faiss_Index_add(
		f.index,
		C.idx_t(n),
		(*C.float)(unsafe.Pointer(&flatVectors[0])),
	)
	if ret != 0 {
		return fmt.Errorf("failed to add vectors to FAISS index: %s", faissLastError())
	}

	// Track ID mappings
	for _, id := range ids {
		f.idToIntID[id] = f.nextID
		f.intIDToID[f.nextID] = id
		f.nextID++
	}

	return nil
}

// Search returns the top-k vectors by inner product (assumes normalized vectors = cosine similarity).
func (f *FAISSIndex) Search(ctx context.Context, query []float32, k int) ([]*VectorResult, error) {
	if len(query) != f.dimensions {
		return nil, fmt.Errorf("query dimension mismatch: got %d, expected %d", len(query), f.dimensions)
	}

	f.mu.RLock()
	defer f.mu.RUnlock()

	if k <= 0 {
		return nil, nil
	}

	// Get actual index size
	ntotal := int(C.faiss_Index_ntotal(f.index))
	if ntotal == 0 {
		return nil, nil
	}

	// Limit k to actual size
	if k > ntotal {
		k = ntotal
	}

	// Allocate result arrays
	distances := make([]float32, k)
	labels := make([]int64, k)

	// Run search
	ret := C.faiss_Index_search(
		f.index,
		1, // nq (number of queries)
		(*C.float)(unsafe.Pointer(&query[0])),
		C.idx_t(k),
		(*C.float)(unsafe.Pointer(&distances[0])),
		(*C.idx_t)(unsafe.Pointer(&labels[0])),
	)
	if ret != 0 {
		return nil, fmt.Errorf("FAISS search failed: %s", faissLastError())
	}

	// Convert to results
	results := make([]*VectorResult, 0, k)
	for i := 0; i < k; i++ {
		label := labels[i]
		if label < 0 {
			continue // Invalid result
		}
		id, ok := f.intIDToID[label]
		if !ok {
			continue // ID was removed
		}
		results = append(results, &VectorResult{
			ID:    id,
			Score: float64(distances[i]),
		})
	}

	// Sort by score descending (FAISS already does this, but ensure consistency)
	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	return results, nil
}

// Remove removes vectors by ID. Note: FAISS IndexFlat doesn't support efficient removal,
// so we only remove from the ID mapping. The vectors remain in the index but are
// excluded from search results. For production, consider periodic rebuilding.
func (f *FAISSIndex) Remove(ctx context.Context, ids []string) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	for _, id := range ids {
		if intID, ok := f.idToIntID[id]; ok {
			delete(f.intIDToID, intID)
			delete(f.idToIntID, id)
		}
	}

	return nil
}

// faissIDMapping stores the ID mapping for persistence.
type faissIDMapping struct {
	IDToIntID map[string]int64
	IntIDToID map[int64]string
	NextID    int64
}

// Save persists the index and ID mappings to path.
func (f *FAISSIndex) Save(path string) error {
	f.mu.RLock()
	defer f.mu.RUnlock()

	if path == "" {
		return nil
	}

	// Create directory
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("create index dir: %w", err)
	}

	// Save FAISS index
	cPath := C.CString(path + ".faiss")
	defer C.free(unsafe.Pointer(cPath))

	ret := C.faiss_write_index_fname(f.index, cPath)
	if ret != 0 {
		return fmt.Errorf("failed to save FAISS index: %s", faissLastError())
	}

	// Save ID mappings
	mapping := faissIDMapping{
		IDToIntID: f.idToIntID,
		IntIDToID: f.intIDToID,
		NextID:    f.nextID,
	}

	mapFile, err := os.Create(path + ".idmap")
	if err != nil {
		return fmt.Errorf("create id map file: %w", err)
	}
	defer mapFile.Close()

	if err := gob.NewEncoder(mapFile).Encode(mapping); err != nil {
		return fmt.Errorf("encode id map: %w", err)
	}

	return nil
}

// Load reads the index and ID mappings from path.
// If the files do not exist, no error is returned and the index is unchanged.
func (f *FAISSIndex) Load(path string) error {
	if path == "" {
		return nil
	}

	faissPath := path + ".faiss"
	mapPath := path + ".idmap"

	// Check if files exist
	if _, err := os.Stat(faissPath); os.IsNotExist(err) {
		return nil
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	// Load FAISS index
	cPath := C.CString(faissPath)
	defer C.free(unsafe.Pointer(cPath))

	var newIndex *C.FaissIndex
	ret := C.faiss_read_index_fname(cPath, 0, &newIndex)
	if ret != 0 {
		return fmt.Errorf("failed to load FAISS index: %s", faissLastError())
	}

	// Close old index and use new one
	if f.index != nil {
		C.faiss_Index_free(f.index)
	}
	f.index = newIndex

	// Load ID mappings
	mapFile, err := os.Open(mapPath)
	if err != nil {
		if os.IsNotExist(err) {
			// Index exists but no mapping - reset mappings
			f.idToIntID = make(map[string]int64)
			f.intIDToID = make(map[int64]string)
			f.nextID = 0
			return nil
		}
		return fmt.Errorf("open id map file: %w", err)
	}
	defer mapFile.Close()

	var mapping faissIDMapping
	if err := gob.NewDecoder(mapFile).Decode(&mapping); err != nil {
		return fmt.Errorf("decode id map: %w", err)
	}

	f.idToIntID = mapping.IDToIntID
	f.intIDToID = mapping.IntIDToID
	f.nextID = mapping.NextID

	return nil
}

// Size returns the number of active vectors (excluding removed ones).
func (f *FAISSIndex) Size() int {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return len(f.idToIntID)
}

// Close frees the FAISS index resources.
func (f *FAISSIndex) Close() error {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.index != nil {
		C.faiss_Index_free(f.index)
		f.index = nil
	}
	return nil
}

// Type returns the index type identifier.
func (f *FAISSIndex) Type() string {
	return string(IndexTypeFAISS)
}
