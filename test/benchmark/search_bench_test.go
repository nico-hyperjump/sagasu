package benchmark

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"testing"

	"github.com/hyperjump/sagasu/internal/embedding"
	"github.com/hyperjump/sagasu/internal/vector"
)

const benchDimensions = 384

// generateRandomVectors creates n random normalized vectors of given dimension.
func generateRandomVectors(n, dim int, seed int64) [][]float32 {
	rng := rand.New(rand.NewSource(seed))
	vecs := make([][]float32, n)
	for i := 0; i < n; i++ {
		vec := make([]float32, dim)
		var norm float32
		for j := 0; j < dim; j++ {
			vec[j] = rng.Float32()*2 - 1 // -1 to 1
			norm += vec[j] * vec[j]
		}
		// Normalize for cosine similarity
		norm = float32(1.0 / float64(norm))
		for j := 0; j < dim; j++ {
			vec[j] *= norm
		}
		vecs[i] = vec
	}
	return vecs
}

// generateIDs creates n string IDs.
func generateIDs(n int) []string {
	ids := make([]string, n)
	for i := 0; i < n; i++ {
		ids[i] = fmt.Sprintf("doc-%d", i)
	}
	return ids
}

// setupMemoryIndex creates and populates a MemoryIndex with n vectors.
func setupMemoryIndex(b *testing.B, n int) (*vector.MemoryIndex, []float32) {
	b.Helper()
	idx, err := vector.NewMemoryIndex(benchDimensions)
	if err != nil {
		b.Fatal(err)
	}

	vecs := generateRandomVectors(n, benchDimensions, 42)
	ids := generateIDs(n)

	ctx := context.Background()
	if err := idx.Add(ctx, ids, vecs); err != nil {
		b.Fatal(err)
	}

	// Generate a random query vector
	query := generateRandomVectors(1, benchDimensions, 123)[0]
	return idx, query
}

// ============================================================================
// Scale Benchmarks - MemoryIndex
// ============================================================================

func BenchmarkMemoryIndex_Search_1k(b *testing.B) {
	idx, query := setupMemoryIndex(b, 1000)
	defer idx.Close()
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = idx.Search(ctx, query, 10)
	}
}

func BenchmarkMemoryIndex_Search_10k(b *testing.B) {
	idx, query := setupMemoryIndex(b, 10000)
	defer idx.Close()
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = idx.Search(ctx, query, 10)
	}
}

func BenchmarkMemoryIndex_Search_100k(b *testing.B) {
	if testing.Short() {
		b.Skip("skipping 100k benchmark in short mode")
	}
	idx, query := setupMemoryIndex(b, 100000)
	defer idx.Close()
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = idx.Search(ctx, query, 10)
	}
}

func BenchmarkMemoryIndex_Add_1k(b *testing.B) {
	vecs := generateRandomVectors(1000, benchDimensions, 42)
	ids := generateIDs(1000)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		idx, _ := vector.NewMemoryIndex(benchDimensions)
		_ = idx.Add(ctx, ids, vecs)
		_ = idx.Close()
	}
}

func BenchmarkMemoryIndex_Add_10k(b *testing.B) {
	vecs := generateRandomVectors(10000, benchDimensions, 42)
	ids := generateIDs(10000)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		idx, _ := vector.NewMemoryIndex(benchDimensions)
		_ = idx.Add(ctx, ids, vecs)
		_ = idx.Close()
	}
}

// ============================================================================
// Concurrent Search Benchmarks
// ============================================================================

func BenchmarkMemoryIndex_ConcurrentSearch_1k_4goroutines(b *testing.B) {
	idx, query := setupMemoryIndex(b, 1000)
	defer idx.Close()
	ctx := context.Background()
	numGoroutines := 4

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _ = idx.Search(ctx, query, 10)
		}
	})
	_ = numGoroutines // Document intent
}

func BenchmarkMemoryIndex_ConcurrentSearch_10k_4goroutines(b *testing.B) {
	idx, query := setupMemoryIndex(b, 10000)
	defer idx.Close()
	ctx := context.Background()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _ = idx.Search(ctx, query, 10)
		}
	})
}

func BenchmarkMemoryIndex_ConcurrentSearch_10k_8goroutines(b *testing.B) {
	idx, query := setupMemoryIndex(b, 10000)
	defer idx.Close()
	ctx := context.Background()

	b.SetParallelism(2) // 2x GOMAXPROCS
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _ = idx.Search(ctx, query, 10)
		}
	})
}

// BenchmarkMemoryIndex_MixedWorkload simulates a realistic workload with
// concurrent searches and occasional adds.
func BenchmarkMemoryIndex_MixedWorkload(b *testing.B) {
	idx, query := setupMemoryIndex(b, 5000)
	defer idx.Close()
	ctx := context.Background()

	// Pre-generate vectors for adding
	addVecs := generateRandomVectors(100, benchDimensions, 999)
	addIDs := make([]string, 100)
	for i := range addIDs {
		addIDs[i] = fmt.Sprintf("new-doc-%d", i)
	}

	var addMu sync.Mutex
	addIdx := 0

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		localIdx := 0
		for pb.Next() {
			// 95% searches, 5% adds
			if localIdx%20 == 0 {
				addMu.Lock()
				if addIdx < len(addIDs) {
					_ = idx.Add(ctx, addIDs[addIdx:addIdx+1], addVecs[addIdx:addIdx+1])
					addIdx++
				}
				addMu.Unlock()
			} else {
				_, _ = idx.Search(ctx, query, 10)
			}
			localIdx++
		}
	})
}

// ============================================================================
// FAISS Benchmarks (only run when FAISS is available)
// ============================================================================

func BenchmarkFAISSIndex_Search_1k(b *testing.B) {
	if !vector.IsFAISSAvailable() {
		b.Skip("FAISS not available (build with -tags=faiss)")
	}

	idx, err := vector.NewFAISSIndex(benchDimensions)
	if err != nil {
		b.Fatal(err)
	}
	defer idx.Close()

	vecs := generateRandomVectors(1000, benchDimensions, 42)
	ids := generateIDs(1000)
	ctx := context.Background()
	if err := idx.Add(ctx, ids, vecs); err != nil {
		b.Fatal(err)
	}

	query := generateRandomVectors(1, benchDimensions, 123)[0]

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = idx.Search(ctx, query, 10)
	}
}

func BenchmarkFAISSIndex_Search_10k(b *testing.B) {
	if !vector.IsFAISSAvailable() {
		b.Skip("FAISS not available (build with -tags=faiss)")
	}

	idx, err := vector.NewFAISSIndex(benchDimensions)
	if err != nil {
		b.Fatal(err)
	}
	defer idx.Close()

	vecs := generateRandomVectors(10000, benchDimensions, 42)
	ids := generateIDs(10000)
	ctx := context.Background()
	if err := idx.Add(ctx, ids, vecs); err != nil {
		b.Fatal(err)
	}

	query := generateRandomVectors(1, benchDimensions, 123)[0]

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = idx.Search(ctx, query, 10)
	}
}

func BenchmarkFAISSIndex_Search_100k(b *testing.B) {
	if !vector.IsFAISSAvailable() {
		b.Skip("FAISS not available (build with -tags=faiss)")
	}
	if testing.Short() {
		b.Skip("skipping 100k benchmark in short mode")
	}

	idx, err := vector.NewFAISSIndex(benchDimensions)
	if err != nil {
		b.Fatal(err)
	}
	defer idx.Close()

	vecs := generateRandomVectors(100000, benchDimensions, 42)
	ids := generateIDs(100000)
	ctx := context.Background()
	if err := idx.Add(ctx, ids, vecs); err != nil {
		b.Fatal(err)
	}

	query := generateRandomVectors(1, benchDimensions, 123)[0]

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = idx.Search(ctx, query, 10)
	}
}

// ============================================================================
// Index Factory Benchmarks
// ============================================================================

func BenchmarkNewVectorIndex_Memory(b *testing.B) {
	for i := 0; i < b.N; i++ {
		idx, _ := vector.NewVectorIndex("memory", benchDimensions)
		_ = idx.Close()
	}
}

func BenchmarkNewVectorIndex_FAISS(b *testing.B) {
	if !vector.IsFAISSAvailable() {
		b.Skip("FAISS not available (build with -tags=faiss)")
	}

	for i := 0; i < b.N; i++ {
		idx, _ := vector.NewVectorIndex("faiss", benchDimensions)
		_ = idx.Close()
	}
}

// ============================================================================
// Embedder Benchmarks
// ============================================================================

func BenchmarkMockEmbedder_Embed(b *testing.B) {
	e := embedding.NewMockEmbedder(benchDimensions)
	ctx := context.Background()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = e.Embed(ctx, "benchmark query text for embedding")
	}
}

func BenchmarkMockEmbedder_EmbedBatch_10(b *testing.B) {
	e := embedding.NewMockEmbedder(benchDimensions)
	ctx := context.Background()
	texts := make([]string, 10)
	for i := range texts {
		texts[i] = fmt.Sprintf("benchmark text %d for batch embedding", i)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = e.EmbedBatch(ctx, texts)
	}
}

func BenchmarkMockEmbedder_ConcurrentEmbed(b *testing.B) {
	e := embedding.NewMockEmbedder(benchDimensions)
	ctx := context.Background()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			text := fmt.Sprintf("concurrent benchmark text %d", i)
			_, _ = e.Embed(ctx, text)
			i++
		}
	})
}

// ============================================================================
// Memory Usage Benchmarks
// ============================================================================

func BenchmarkMemoryIndex_MemoryUsage_10k(b *testing.B) {
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		idx, _ := vector.NewMemoryIndex(benchDimensions)
		vecs := generateRandomVectors(10000, benchDimensions, 42)
		ids := generateIDs(10000)
		ctx := context.Background()
		_ = idx.Add(ctx, ids, vecs)
		_ = idx.Close()
	}
}

// ============================================================================
// Backward Compatibility - Keep original benchmark names
// ============================================================================

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
