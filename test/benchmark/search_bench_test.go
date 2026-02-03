package benchmark

import (
	"context"
	"testing"

	"github.com/hyperjump/sagasu/internal/embedding"
	"github.com/hyperjump/sagasu/internal/search"
	"github.com/hyperjump/sagasu/internal/vector"
)

func BenchmarkFuse(b *testing.B) {
	kw := make(map[string]float64)
	sem := make(map[string]float64)
	for i := 0; i < 100; i++ {
		id := string(rune('a' + i%26))
		kw[id] = float64(i) / 100
		sem[id] = float64(100-i) / 100
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = search.Fuse(kw, sem, 0.5, 0.5)
	}
}

func BenchmarkMemoryIndexSearch(b *testing.B) {
	idx, _ := vector.NewMemoryIndex(384)
	ctx := context.Background()
	vecs := make([][]float32, 1000)
	ids := make([]string, 1000)
	for i := 0; i < 1000; i++ {
		vecs[i] = make([]float32, 384)
		vecs[i][0] = float32(i) / 1000
		ids[i] = string(rune('a' + i%26))
	}
	_ = idx.Add(ctx, ids, vecs)
	query := make([]float32, 384)
	query[0] = 1.0
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = idx.Search(ctx, query, 10)
	}
}

func BenchmarkMockEmbedder_Embed(b *testing.B) {
	e := embedding.NewMockEmbedder(384)
	ctx := context.Background()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = e.Embed(ctx, "benchmark query text for embedding")
	}
}
