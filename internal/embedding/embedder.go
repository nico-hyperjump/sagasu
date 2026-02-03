// Package embedding provides text embedding via ONNX and caching.
package embedding

import "context"

// Embedder produces vector embeddings for text.
type Embedder interface {
	Embed(ctx context.Context, text string) ([]float32, error)
	EmbedBatch(ctx context.Context, texts []string) ([][]float32, error)
	Dimensions() int
	Close() error
}
