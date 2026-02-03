// Package integration provides end-to-end tests (requires real storage and indices).
package integration

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/hyperjump/sagasu/internal/config"
	"github.com/hyperjump/sagasu/internal/embedding"
	"github.com/hyperjump/sagasu/internal/indexer"
	"github.com/hyperjump/sagasu/internal/keyword"
	"github.com/hyperjump/sagasu/internal/models"
	"github.com/hyperjump/sagasu/internal/search"
	"github.com/hyperjump/sagasu/internal/storage"
	"github.com/hyperjump/sagasu/internal/vector"
)

func TestIntegration_Search(t *testing.T) {
	dir := t.TempDir()
	cfg := &config.Config{
		Storage: config.StorageConfig{
			DatabasePath:   filepath.Join(dir, "db.sqlite"),
			BleveIndexPath: filepath.Join(dir, "bleve"),
			FAISSIndexPath: filepath.Join(dir, "faiss"),
		},
		Embedding: config.EmbeddingConfig{Dimensions: 4, MaxTokens: 32, CacheSize: 100},
		Search: config.SearchConfig{
			ChunkSize: 10, ChunkOverlap: 2, TopKCandidates: 20,
			DefaultKeywordWeight: 0.5, DefaultSemanticWeight: 0.5,
		},
	}

	store, err := storage.NewSQLiteStorage(cfg.Storage.DatabasePath)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	embedder := embedding.NewMockEmbedder(cfg.Embedding.Dimensions)
	defer embedder.Close()

	vecIndex, err := vector.NewMemoryIndex(cfg.Embedding.Dimensions)
	if err != nil {
		t.Fatal(err)
	}
	defer vecIndex.Close()

	kwIndex, err := keyword.NewBleveIndex(cfg.Storage.BleveIndexPath)
	if err != nil {
		t.Fatal(err)
	}
	defer kwIndex.Close()

	engine := search.NewEngine(store, embedder, vecIndex, kwIndex, &cfg.Search)
	idx := indexer.NewIndexer(store, embedder, vecIndex, kwIndex, &cfg.Search)
	ctx := context.Background()

	if err := idx.IndexDocument(ctx, &models.DocumentInput{
		ID: "doc1", Title: "ML", Content: "Machine learning algorithms learn from data.",
	}); err != nil {
		t.Fatal(err)
	}
	if err := idx.IndexDocument(ctx, &models.DocumentInput{
		ID: "doc2", Title: "Search", Content: "Semantic search uses embeddings to find similar content.",
	}); err != nil {
		t.Fatal(err)
	}

	resp, err := engine.Search(ctx, &models.SearchQuery{
		Query: "machine learning", Limit: 5, KeywordWeight: 0.5, SemanticWeight: 0.5,
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Total < 1 {
		t.Errorf("expected at least 1 result, got %d", resp.Total)
	}
}
