package indexer

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/hyperjump/sagasu/internal/config"
	"github.com/hyperjump/sagasu/internal/embedding"
	"github.com/hyperjump/sagasu/internal/extract"
	"github.com/hyperjump/sagasu/internal/fileid"
	"github.com/hyperjump/sagasu/internal/keyword"
	"github.com/hyperjump/sagasu/internal/storage"
	"github.com/hyperjump/sagasu/internal/vector"
	"github.com/xuri/excelize/v2"
)

func TestExtensionAllowed(t *testing.T) {
	tests := []struct {
		ext     string
		allowed []string
		want    bool
	}{
		{".txt", []string{".txt", ".md"}, true},
		{".TXT", []string{".txt"}, true},
		{".md", []string{".txt", ".md"}, true},
		{".go", []string{".txt"}, false},
		{"", []string{".txt"}, false},
		{".rst", []string{".txt", ".md", ".rst"}, true},
	}
	for _, tt := range tests {
		got := extensionAllowed(tt.ext, tt.allowed)
		if got != tt.want {
			t.Errorf("extensionAllowed(%q, %v) = %v, want %v", tt.ext, tt.allowed, got, tt.want)
		}
	}
}

func testIndexerWithStorage(t *testing.T, dir string) (*Indexer, storage.Storage) {
	t.Helper()
	cfg := &config.SearchConfig{
		ChunkSize: 10, ChunkOverlap: 2, TopKCandidates: 20,
		DefaultKeywordWeight: 0.5, DefaultSemanticWeight: 0.5,
	}
	store, err := storage.NewSQLiteStorage(filepath.Join(dir, "db.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	embedder := embedding.NewMockEmbedder(4)
	t.Cleanup(func() { _ = embedder.Close() })
	vecIndex, err := vector.NewMemoryIndex(4)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = vecIndex.Close() })
	kwIndex, err := keyword.NewBleveIndex(filepath.Join(dir, "bleve"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = kwIndex.Close() })
	return NewIndexer(store, embedder, vecIndex, kwIndex, cfg, nil), store
}

func mustAbs(path string) string {
	a, err := filepath.Abs(path)
	if err != nil {
		panic(err)
	}
	return a
}

func TestIndexFile_createAndUpdate(t *testing.T) {
	dir := t.TempDir()
	idx, store := testIndexerWithStorage(t, dir)
	ctx := context.Background()

	fPath := filepath.Join(dir, "doc.txt")
	if err := os.WriteFile(fPath, []byte("Hello world content."), 0600); err != nil {
		t.Fatal(err)
	}
	if err := idx.IndexFile(ctx, fPath, []string{".txt", ".md"}); err != nil {
		t.Fatal(err)
	}
	docID := fileid.FileDocID(mustAbs(fPath))
	doc, err := store.GetDocument(ctx, docID)
	if err != nil {
		t.Fatal(err)
	}
	if doc.Title != "doc.txt" || doc.Content != "Hello world content." {
		t.Errorf("unexpected doc: title=%q content=%q", doc.Title, doc.Content)
	}
	if doc.Metadata["source_path"] != mustAbs(fPath) {
		t.Errorf("metadata source_path: got %v", doc.Metadata["source_path"])
	}

	if err := os.WriteFile(fPath, []byte("Updated content."), 0600); err != nil {
		t.Fatal(err)
	}
	if err := idx.IndexFile(ctx, fPath, []string{".txt"}); err != nil {
		t.Fatal(err)
	}
	doc2, err := store.GetDocument(ctx, docID)
	if err != nil {
		t.Fatal(err)
	}
	if doc2.Content != "Updated content." {
		t.Errorf("after update: content=%q", doc2.Content)
	}
}

func TestIndexFile_extensionFiltered(t *testing.T) {
	dir := t.TempDir()
	idx, _ := testIndexerWithStorage(t, dir)
	ctx := context.Background()

	fPath := filepath.Join(dir, "script.sh")
	if err := os.WriteFile(fPath, []byte("#!/bin/bash"), 0600); err != nil {
		t.Fatal(err)
	}
	err := idx.IndexFile(ctx, fPath, []string{".txt", ".md"})
	if err == nil {
		t.Error("expected error for disallowed extension")
	}
}

func TestIndexFile_deleteByPath(t *testing.T) {
	dir := t.TempDir()
	idx, store := testIndexerWithStorage(t, dir)
	ctx := context.Background()

	fPath := filepath.Join(dir, "note.md")
	if err := os.WriteFile(fPath, []byte("Note content."), 0600); err != nil {
		t.Fatal(err)
	}
	if err := idx.IndexFile(ctx, fPath, nil); err != nil {
		t.Fatal(err)
	}
	docID := fileid.FileDocID(mustAbs(fPath))
	if _, err := store.GetDocument(ctx, docID); err != nil {
		t.Fatal(err)
	}
	if err := idx.DeleteDocument(ctx, docID); err != nil {
		t.Fatal(err)
	}
	if _, err := store.GetDocument(ctx, docID); err == nil {
		t.Error("document should be deleted")
	}
}

func TestIndexFile_notRegularFile(t *testing.T) {
	dir := t.TempDir()
	idx, _ := testIndexerWithStorage(t, dir)
	ctx := context.Background()

	err := idx.IndexFile(ctx, dir, []string{".txt"})
	if err == nil {
		t.Error("expected error for directory")
	}
}

func TestIndexFile_nonexistent(t *testing.T) {
	dir := t.TempDir()
	idx, _ := testIndexerWithStorage(t, dir)
	ctx := context.Background()

	err := idx.IndexFile(ctx, filepath.Join(dir, "missing.txt"), nil)
	if err == nil {
		t.Error("expected error for missing file")
	}
}

func TestIndexFile_excelWithExtractor(t *testing.T) {
	dir := t.TempDir()
	cfg := &config.SearchConfig{
		ChunkSize: 10, ChunkOverlap: 2, TopKCandidates: 20,
		DefaultKeywordWeight: 0.5, DefaultSemanticWeight: 0.5,
	}
	store, err := storage.NewSQLiteStorage(filepath.Join(dir, "db.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	embedder := embedding.NewMockEmbedder(4)
	t.Cleanup(func() { _ = embedder.Close() })
	vecIndex, err := vector.NewMemoryIndex(4)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = vecIndex.Close() })
	kwIndex, err := keyword.NewBleveIndex(filepath.Join(dir, "bleve"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = kwIndex.Close() })
	idx := NewIndexer(store, embedder, vecIndex, kwIndex, cfg, extract.NewExtractor())

	fPath := filepath.Join(dir, "data.xlsx")
	f := excelize.NewFile()
	f.SetCellValue("Sheet1", "A1", "Excel searchable content")
	if err := f.SaveAs(fPath); err != nil {
		t.Fatalf("SaveAs: %v", err)
	}
	f.Close()

	ctx := context.Background()
	if err := idx.IndexFile(ctx, fPath, []string{".xlsx", ".txt"}); err != nil {
		t.Fatalf("IndexFile: %v", err)
	}
	docID := fileid.FileDocID(mustAbs(fPath))
	doc, err := store.GetDocument(ctx, docID)
	if err != nil {
		t.Fatal(err)
	}
	if doc.Title != "data.xlsx" || doc.Content != "Excel searchable content" {
		t.Errorf("unexpected doc: title=%q content=%q", doc.Title, doc.Content)
	}
}

func TestIndexDirectory(t *testing.T) {
	dir := t.TempDir()
	idx, _ := testIndexerWithStorage(t, dir)
	ctx := context.Background()

	sub := filepath.Join(dir, "sub")
	if err := os.Mkdir(sub, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "a.txt"), []byte("file a"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "b.txt"), []byte("file b"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sub, "c.txt"), []byte("file c"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "skip.xyz"), []byte("skip"), 0600); err != nil {
		t.Fatal(err)
	}

	n, err := idx.IndexDirectory(ctx, dir, []string{".txt"})
	if err != nil {
		t.Fatalf("IndexDirectory: %v", err)
	}
	if n != 3 {
		t.Errorf("IndexDirectory: indexed %d files, want 3", n)
	}
}
