package embedding

import (
	"context"
	"math"
)

// MockEmbedder is a deterministic embedder for tests. It returns a fixed-dimension
// vector derived from the text hash so that the same text always gets the same embedding.
type MockEmbedder struct {
	dimensions int
}

// NewMockEmbedder returns an embedder that produces deterministic embeddings of the given dimensions.
func NewMockEmbedder(dimensions int) *MockEmbedder {
	if dimensions <= 0 {
		dimensions = 384
	}
	return &MockEmbedder{dimensions: dimensions}
}

// Embed returns a deterministic embedding based on the text hash.
func (e *MockEmbedder) Embed(ctx context.Context, text string) ([]float32, error) {
	h := HashString(text)
	emb := make([]float32, e.dimensions)
	for i := 0; i < e.dimensions; i++ {
		emb[i] = float32(math.Sin(float64(h*(i+1)))*0.1 + 0.01)
	}
	// Normalize to unit length for cosine similarity
	var sum float64
	for _, v := range emb {
		sum += float64(v * v)
	}
	if sum > 0 {
		norm := 1.0 / math.Sqrt(sum)
		for i := range emb {
			emb[i] *= float32(norm)
		}
	}
	return emb, nil
}

// EmbedBatch calls Embed for each text.
func (e *MockEmbedder) EmbedBatch(ctx context.Context, texts []string) ([][]float32, error) {
	embeddings := make([][]float32, len(texts))
	for i, text := range texts {
		emb, err := e.Embed(ctx, text)
		if err != nil {
			return nil, err
		}
		embeddings[i] = emb
	}
	return embeddings, nil
}

// Dimensions returns the embedding dimension.
func (e *MockEmbedder) Dimensions() int {
	return e.dimensions
}

// Close is a no-op for MockEmbedder.
func (e *MockEmbedder) Close() error {
	return nil
}
