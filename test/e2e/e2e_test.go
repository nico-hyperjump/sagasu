package e2e

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/hyperjump/sagasu/internal/config"
	"github.com/hyperjump/sagasu/internal/embedding"
	"github.com/hyperjump/sagasu/internal/extract"
	"github.com/hyperjump/sagasu/internal/fileid"
	"github.com/hyperjump/sagasu/internal/indexer"
	"github.com/hyperjump/sagasu/internal/keyword"
	"github.com/hyperjump/sagasu/internal/models"
	"github.com/hyperjump/sagasu/internal/search"
	"github.com/hyperjump/sagasu/internal/storage"
	"github.com/hyperjump/sagasu/internal/vector"
)

const (
	e2eSearchLimit = 30
	e2eDimensions  = 4
)

func TestE2E_SearchReturnsCorrectResults(t *testing.T) {
	dir := t.TempDir()
	cfg := &config.Config{
		Storage: config.StorageConfig{
			DatabasePath:   filepath.Join(dir, "db.sqlite"),
			BleveIndexPath: filepath.Join(dir, "bleve"),
			FAISSIndexPath: filepath.Join(dir, "faiss"),
		},
		Embedding: config.EmbeddingConfig{Dimensions: e2eDimensions, MaxTokens: 256, CacheSize: 500},
		Search: config.SearchConfig{
			ChunkSize:              64,
			ChunkOverlap:            8,
			TopKCandidates:          50,
			DefaultKeywordEnabled:   true,
			DefaultSemanticEnabled:  true,
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
	idx := indexer.NewIndexer(store, embedder, vecIndex, kwIndex, &cfg.Search, nil)
	ctx := context.Background()

	corpus := BuildCorpus()
	if corpus.TotalDocs == 0 {
		t.Fatal("corpus has no documents")
	}
	if corpus.TotalQueries == 0 {
		t.Fatal("corpus has no query test cases")
	}

	for _, input := range corpus.ToDocumentInputs() {
		if err := idx.IndexDocument(ctx, input); err != nil {
			t.Fatalf("index document %q: %v", input.ID, err)
		}
	}

	t.Logf("indexed %d documents; running %d query test cases", corpus.TotalDocs, corpus.TotalQueries)

	for _, tc := range corpus.TestCases {
		t.Run(tc.Description, func(t *testing.T) {
			resp, err := engine.Search(ctx, &models.SearchQuery{
				Query:          tc.Query,
				Limit:          e2eSearchLimit,
				KeywordEnabled: true,
				SemanticEnabled: true,
			})
			if err != nil {
				t.Fatalf("search failed: %v", err)
			}
			resultIDs := documentIDsFromResponse(resp)
			if !containsAny(resultIDs, tc.ExpectedDocIDs) {
				t.Errorf("query %q: expected at least one of %v in results, got %d results (ids: %v)",
					tc.Query, tc.ExpectedDocIDs, len(resultIDs), resultIDs)
			}
		})
	}
}

func documentIDsFromResponse(resp *models.SearchResponse) []string {
	ids := make([]string, 0, len(resp.NonSemanticResults)+len(resp.SemanticResults))
	for _, r := range resp.NonSemanticResults {
		if r.Document != nil {
			ids = append(ids, r.Document.ID)
		}
	}
	for _, r := range resp.SemanticResults {
		if r.Document != nil {
			ids = append(ids, r.Document.ID)
		}
	}
	return ids
}

func containsAny(got []string, expected []string) bool {
	set := make(map[string]bool)
	for _, id := range got {
		set[id] = true
	}
	for _, id := range expected {
		if set[id] {
			return true
		}
	}
	return false
}

// TestE2E_FileIndexingSearch indexes real files of all supported types (.txt, .md, .rst, .docx, .xlsx, .pptx, .odp, .ods)
// via IndexDirectory with extractor, then runs the same query test cases.
// Document IDs are derived from file paths (fileid.FileDocID).
// PDF extraction is covered by internal/extract tests; a minimal PDF with extractable text is not generated here.
func TestE2E_FileIndexingSearch(t *testing.T) {
	dir := t.TempDir()
	docDir := filepath.Join(dir, "docs")
	if err := os.MkdirAll(docDir, 0755); err != nil {
		t.Fatal(err)
	}

	corpus := BuildCorpus()
	exts := SupportedFileExtensions
	corpusIDToFileDocID := make(map[string]string)
	nFiles := 0
	for i, d := range corpus.Documents {
		if nFiles >= 50 {
			break
		}
		ext := exts[i%len(exts)]
		name := d.ID + ext
		path := filepath.Join(docDir, name)
		content := d.Title + "\n\n" + d.Content
		fileBytes, err := WriteMinimalFile(ext, content)
		if err != nil {
			t.Fatalf("write minimal file %s: %v", name, err)
		}
		if err := os.WriteFile(path, fileBytes, 0644); err != nil {
			t.Fatalf("write file %s: %v", path, err)
		}
		absPath, _ := filepath.Abs(path)
		corpusIDToFileDocID[d.ID] = fileid.FileDocID(absPath)
		nFiles++
	}

	cfg := &config.Config{
		Storage: config.StorageConfig{
			DatabasePath:   filepath.Join(dir, "db.sqlite"),
			BleveIndexPath: filepath.Join(dir, "bleve"),
			FAISSIndexPath: filepath.Join(dir, "faiss"),
		},
		Embedding: config.EmbeddingConfig{Dimensions: e2eDimensions, MaxTokens: 256, CacheSize: 500},
		Search: config.SearchConfig{
			ChunkSize:              64,
			ChunkOverlap:            8,
			TopKCandidates:          50,
			DefaultKeywordEnabled:   true,
			DefaultSemanticEnabled:  true,
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

	extractor := extract.NewExtractor()
	idx := indexer.NewIndexer(store, embedder, vecIndex, kwIndex, &cfg.Search, extractor)
	engine := search.NewEngine(store, embedder, vecIndex, kwIndex, &cfg.Search)
	ctx := context.Background()

	allowedExts := SupportedFileExtensions
	n, err := idx.IndexDirectory(ctx, docDir, allowedExts)
	if err != nil {
		t.Fatalf("index directory: %v", err)
	}
	if n != nFiles {
		t.Fatalf("expected %d files indexed, got %d", nFiles, n)
	}

	t.Logf("indexed %d files from %s; running query test cases (only for docs that were written as files)", n, docDir)

	var run int
	for _, tc := range corpus.TestCases {
		expectedFileDocIDs := make([]string, 0)
		for _, corpusID := range tc.ExpectedDocIDs {
			if fileDocID, ok := corpusIDToFileDocID[corpusID]; ok {
				expectedFileDocIDs = append(expectedFileDocIDs, fileDocID)
			}
		}
		if len(expectedFileDocIDs) == 0 {
			continue
		}
		run++
		t.Run(tc.Description, func(t *testing.T) {
			resp, err := engine.Search(ctx, &models.SearchQuery{
				Query:           tc.Query,
				Limit:           e2eSearchLimit,
				KeywordEnabled:  true,
				SemanticEnabled: true,
			})
			if err != nil {
				t.Fatalf("search failed: %v", err)
			}
			resultIDs := documentIDsFromResponse(resp)
			if !containsAny(resultIDs, expectedFileDocIDs) {
				t.Errorf("query %q: expected at least one of %v in results, got %d results (sample ids: %v)",
					tc.Query, expectedFileDocIDs, len(resultIDs), resultIDs)
			}
		})
	}
	if run == 0 {
		t.Fatal("no query test cases matched the file-based corpus")
	}
	t.Logf("ran %d query test cases for file-based index", run)
}
