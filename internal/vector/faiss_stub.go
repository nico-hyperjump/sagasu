//go:build !faiss || !cgo
// +build !faiss !cgo

// Package vector provides a stub for FAISS when the faiss build tag is not set.
package vector

import (
	"context"
	"fmt"
)

// FAISSIndex is a stub that returns an error when FAISS is not available.
// Build with -tags=faiss to enable FAISS support.
type FAISSIndex struct{}

// NewFAISSIndex returns an error because FAISS is not available.
func NewFAISSIndex(dimensions int) (*FAISSIndex, error) {
	return nil, fmt.Errorf("FAISS not available: build with -tags=faiss and install FAISS library")
}

// Add is not implemented without FAISS.
func (f *FAISSIndex) Add(ctx context.Context, ids []string, vectors [][]float32) error {
	return fmt.Errorf("FAISS not available")
}

// Search is not implemented without FAISS.
func (f *FAISSIndex) Search(ctx context.Context, query []float32, k int) ([]*VectorResult, error) {
	return nil, fmt.Errorf("FAISS not available")
}

// Remove is not implemented without FAISS.
func (f *FAISSIndex) Remove(ctx context.Context, ids []string) error {
	return fmt.Errorf("FAISS not available")
}

// Save is not implemented without FAISS.
func (f *FAISSIndex) Save(path string) error {
	return fmt.Errorf("FAISS not available")
}

// Load is not implemented without FAISS.
func (f *FAISSIndex) Load(path string) error {
	return fmt.Errorf("FAISS not available")
}

// Size returns 0 without FAISS.
func (f *FAISSIndex) Size() int {
	return 0
}

// Close is a no-op without FAISS.
func (f *FAISSIndex) Close() error {
	return nil
}

// Type returns the index type identifier.
func (f *FAISSIndex) Type() string {
	return string(IndexTypeFAISS)
}
