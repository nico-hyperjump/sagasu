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
		DefaultKeywordEnabled: true, DefaultSemanticEnabled: true,
	}
	engine := NewEngine(store, emb, vecIndex, kwIndex, cfg)
	idx := indexer.NewIndexer(store, emb, vecIndex, kwIndex, cfg, nil)

	if err := idx.IndexDocument(ctx, &models.DocumentInput{
		ID: "d1", Title: "T1", Content: "machine learning algorithms",
	}); err != nil {
		t.Fatal(err)
	}

	resp, err := engine.Search(ctx, &models.SearchQuery{
		Query: "machine learning", Limit: 5, KeywordEnabled: true, SemanticEnabled: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	total := resp.TotalNonSemantic + resp.TotalSemantic
	if total < 1 {
		t.Errorf("expected at least 1 result, got total non_semantic=%d semantic=%d", resp.TotalNonSemantic, resp.TotalSemantic)
	}
}

func TestResolveMinScores_and_filterByMinScore(t *testing.T) {
	ctx := context.Background()
	store, err := storage.NewSQLiteStorage(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	emb := embedding.NewMockEmbedder(4)
	defer emb.Close()
	vecIndex, _ := vector.NewMemoryIndex(4)
	defer vecIndex.Close()
	kwPath := t.TempDir() + "/bleve"
	kwIndex, _ := keyword.NewBleveIndex(kwPath)
	defer kwIndex.Close()

	cfg := &config.SearchConfig{
		TopKCandidates: 20, ChunkSize: 50, ChunkOverlap: 10,
		DefaultKeywordEnabled: true, DefaultSemanticEnabled: true,
		DefaultMinKeywordScore: 0.8, DefaultMinSemanticScore: 0.9,
	}
	engine := NewEngine(store, emb, vecIndex, kwIndex, cfg)
	idx := indexer.NewIndexer(store, emb, vecIndex, kwIndex, cfg, nil)
	if err := idx.IndexDocument(ctx, &models.DocumentInput{ID: "d1", Title: "T1", Content: "machine learning"}); err != nil {
		t.Fatal(err)
	}

	resp, err := engine.Search(ctx, &models.SearchQuery{
		Query: "machine learning", Limit: 5, KeywordEnabled: true, SemanticEnabled: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.TotalNonSemantic + resp.TotalSemantic

	resp2, err := engine.Search(ctx, &models.SearchQuery{
		Query: "machine learning", Limit: 5, MinScore: 0.1, KeywordEnabled: true, SemanticEnabled: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	_ = resp2

	resp3, err := engine.Search(ctx, &models.SearchQuery{
		Query: "machine learning", Limit: 5,
		MinKeywordScore: 0.05, MinSemanticScore: 0.05,
		KeywordEnabled: true, SemanticEnabled: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	_ = resp3
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
		DefaultKeywordEnabled: true, DefaultSemanticEnabled: true,
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

func TestEngine_VectorIndexType(t *testing.T) {
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
		DefaultKeywordEnabled: true, DefaultSemanticEnabled: true,
	}
	engine := NewEngine(store, emb, vecIndex, kwIndex, cfg)

	if got := engine.VectorIndexType(); got != "memory" {
		t.Errorf("VectorIndexType() = %q, want %q", got, "memory")
	}
}

func TestEngine_Search_FuzzyEnabled(t *testing.T) {
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
		DefaultKeywordEnabled: true, DefaultSemanticEnabled: true,
	}
	engine := NewEngine(store, emb, vecIndex, kwIndex, cfg)
	idx := indexer.NewIndexer(store, emb, vecIndex, kwIndex, cfg, nil)

	// Index a document with "proposal"
	if err := idx.IndexDocument(ctx, &models.DocumentInput{
		ID: "d1", Title: "Project Proposal", Content: "This proposal outlines the project scope.",
	}); err != nil {
		t.Fatal(err)
	}

	// Search with typo "propodal" without fuzzy - should NOT find results
	respNoFuzzy, err := engine.Search(ctx, &models.SearchQuery{
		Query: "propodal", Limit: 5, KeywordEnabled: true, SemanticEnabled: false, FuzzyEnabled: false,
	})
	if err != nil {
		t.Fatal(err)
	}
	if respNoFuzzy.TotalNonSemantic > 0 {
		t.Errorf("without fuzzy: expected 0 results for typo 'propodal', got %d", respNoFuzzy.TotalNonSemantic)
	}

	// Search with typo "propodal" with fuzzy - should find results
	respFuzzy, err := engine.Search(ctx, &models.SearchQuery{
		Query: "propodal", Limit: 5, KeywordEnabled: true, SemanticEnabled: false, FuzzyEnabled: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if respFuzzy.TotalNonSemantic == 0 {
		t.Error("with fuzzy: expected results for typo 'propodal' -> 'proposal'")
	}
}

func TestEngine_WithSpellChecker(t *testing.T) {
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
		DefaultKeywordEnabled: true, DefaultSemanticEnabled: true,
	}
	
	engine := NewEngine(store, emb, vecIndex, kwIndex, cfg).WithSpellChecker()
	
	// Spell checker should be initialized
	if engine.spellChecker == nil {
		t.Error("WithSpellChecker should initialize spellChecker")
	}
}

func TestEngine_Search_Suggestions(t *testing.T) {
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
		DefaultKeywordEnabled: true, DefaultSemanticEnabled: true,
	}
	engine := NewEngine(store, emb, vecIndex, kwIndex, cfg).WithSpellChecker()
	idx := indexer.NewIndexer(store, emb, vecIndex, kwIndex, cfg, nil)

	// Index a document with "proposal"
	if err := idx.IndexDocument(ctx, &models.DocumentInput{
		ID: "d1", Title: "Budget Proposal", Content: "The proposal for the budget was approved.",
	}); err != nil {
		t.Fatal(err)
	}

	// Refresh spell checker cache
	if err := engine.RefreshSpellChecker(); err != nil {
		t.Fatal(err)
	}

	// Search with typo and fuzzy enabled - should get suggestions
	resp, err := engine.Search(ctx, &models.SearchQuery{
		Query: "propodal", Limit: 5, KeywordEnabled: true, SemanticEnabled: false, FuzzyEnabled: true,
	})
	if err != nil {
		t.Fatal(err)
	}

	// Should have suggestions since "propodal" is a typo for "proposal"
	if len(resp.Suggestions) == 0 {
		t.Error("expected suggestions for typo 'propodal'")
	}
}

func TestEngine_RefreshSpellChecker_NilChecker(t *testing.T) {
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
	}
	
	// Don't call WithSpellChecker
	engine := NewEngine(store, emb, vecIndex, kwIndex, cfg)
	
	// Should not error when spellChecker is nil
	if err := engine.RefreshSpellChecker(); err != nil {
		t.Errorf("RefreshSpellChecker with nil checker should return nil, got %v", err)
	}
}
