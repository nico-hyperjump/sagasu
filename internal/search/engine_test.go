package search

import (
	"context"
	"testing"

	"github.com/hyperjump/sagasu/internal/config"
	"github.com/hyperjump/sagasu/internal/embedding"
	"github.com/hyperjump/sagasu/internal/indexer"
	"github.com/hyperjump/sagasu/internal/keyword"
	"github.com/hyperjump/sagasu/internal/models"
	"github.com/hyperjump/sagasu/internal/storage"
	"github.com/hyperjump/sagasu/internal/vector"
)

func TestEngine_Search(t *testing.T) {
	ctx := context.Background()
	store, err := storage.NewSQLiteStorage(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	emb := embedding.NewMockEmbedder(4)
	defer emb.Close()

	vecIndex, err := vector.NewMemoryIndex(4)
	if err != nil {
		t.Fatal(err)
	}
	defer vecIndex.Close()

	kwPath := t.TempDir() + "/bleve"
	kwIndex, err := keyword.NewBleveIndex(kwPath)
	if err != nil {
		t.Fatal(err)
	}
	defer kwIndex.Close()

	cfg := &config.SearchConfig{
		TopKCandidates: 20, ChunkSize: 50, ChunkOverlap: 10,
		DefaultKeywordWeight: 0.5, DefaultSemanticWeight: 0.5,
	}
	engine := NewEngine(store, emb, vecIndex, kwIndex, cfg)
	idx := indexer.NewIndexer(store, emb, vecIndex, kwIndex, cfg, nil)

	if err := idx.IndexDocument(ctx, &models.DocumentInput{
		ID: "d1", Title: "T1", Content: "machine learning algorithms",
	}); err != nil {
		t.Fatal(err)
	}

	resp, err := engine.Search(ctx, &models.SearchQuery{
		Query: "machine learning", Limit: 5, KeywordWeight: 0.5, SemanticWeight: 0.5,
	})
	if err != nil {
		t.Fatal(err)
	}
	total := resp.TotalNonSemantic + resp.TotalSemantic
	if total < 1 {
		t.Errorf("expected at least 1 result, got total non_semantic=%d semantic=%d", resp.TotalNonSemantic, resp.TotalSemantic)
	}
}

func TestEngine_VectorIndexSize(t *testing.T) {
	ctx := context.Background()
	store, err := storage.NewSQLiteStorage(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	emb := embedding.NewMockEmbedder(4)
	defer emb.Close()

	vecIndex, err := vector.NewMemoryIndex(4)
	if err != nil {
		t.Fatal(err)
	}
	defer vecIndex.Close()

	kwPath := t.TempDir() + "/bleve"
	kwIndex, err := keyword.NewBleveIndex(kwPath)
	if err != nil {
		t.Fatal(err)
	}
	defer kwIndex.Close()

	cfg := &config.SearchConfig{
		TopKCandidates: 20, ChunkSize: 50, ChunkOverlap: 10,
		DefaultKeywordWeight: 0.5, DefaultSemanticWeight: 0.5,
	}
	engine := NewEngine(store, emb, vecIndex, kwIndex, cfg)
	idx := indexer.NewIndexer(store, emb, vecIndex, kwIndex, cfg, nil)

	if got := engine.VectorIndexSize(); got != 0 {
		t.Errorf("empty index: VectorIndexSize() = %d, want 0", got)
	}

	if err := idx.IndexDocument(ctx, &models.DocumentInput{
		ID: "d1", Title: "T1", Content: "short",
	}); err != nil {
		t.Fatal(err)
	}

	if got := engine.VectorIndexSize(); got < 1 {
		t.Errorf("after index: VectorIndexSize() = %d, want >= 1", got)
	}
}
