package keyword

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/hyperjump/sagasu/internal/models"
)

func TestBleveIndex_SearchFindsContent(t *testing.T) {
	dir := t.TempDir()
	indexPath := filepath.Join(dir, "bleve")

	idx, err := NewBleveIndex(indexPath)
	if err != nil {
		t.Fatalf("NewBleveIndex: %v", err)
	}
	defer func() {
		_ = idx.Close()
	}()

	ctx := context.Background()
	docID := "file:abc123"
	doc := &models.Document{
		ID:      docID,
		Title:   "Ausvet Monthly Report 17 - May 2023.docx",
		Content: "This report mentions Omnisyan and other findings. The Bayes app is also referenced.",
	}

	if err := idx.Index(ctx, doc.ID, doc); err != nil {
		t.Fatalf("Index: %v", err)
	}

	results, err := idx.Search(ctx, "Omnisyan", 10)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(results) == 0 {
		t.Fatal("expected at least one keyword result for \"Omnisyan\" in document content")
	}
	if results[0].ID != docID {
		t.Errorf("first result ID = %q, want %q", results[0].ID, docID)
	}

	// Standard analyzer (no stemming) so "bayes" matches "Bayes" in content
	results2, err := idx.Search(ctx, "bayes", 10)
	if err != nil {
		t.Fatalf("Search bayes: %v", err)
	}
	if len(results2) == 0 {
		t.Fatal("expected at least one keyword result for \"bayes\" in document content (standard analyzer, no stop/stem)")
	}
	if results2[0].ID != docID {
		t.Errorf("first result ID = %q, want %q", results2[0].ID, docID)
	}
}

func TestBleveIndex_SearchFindsTitle(t *testing.T) {
	dir := t.TempDir()
	indexPath := filepath.Join(dir, "bleve")

	idx, err := NewBleveIndex(indexPath)
	if err != nil {
		t.Fatalf("NewBleveIndex: %v", err)
	}
	defer func() {
		_ = idx.Close()
	}()

	ctx := context.Background()
	docID := "file:xyz"
	doc := &models.Document{
		ID:      docID,
		Title:   "Ausvet Monthly Report 17 - May 2023.docx",
		Content: "Some body text.",
	}

	if err := idx.Index(ctx, doc.ID, doc); err != nil {
		t.Fatalf("Index: %v", err)
	}

	// Query "Report" (English analyzer stems so "Report" matches title)
	results, err := idx.Search(ctx, "Report", 10)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(results) == 0 {
		t.Fatal("expected at least one keyword result for \"Report\" in title")
	}
	if results[0].ID != docID {
		t.Errorf("first result ID = %q, want %q", results[0].ID, docID)
	}
}

func TestBleveIndex_OpenExistingRecreatesIndex(t *testing.T) {
	dir := t.TempDir()
	indexPath := filepath.Join(dir, "bleve")

	idx1, err := NewBleveIndex(indexPath)
	if err != nil {
		t.Fatalf("NewBleveIndex: %v", err)
	}
	ctx := context.Background()
	doc := &models.Document{ID: "doc1", Title: "T", Content: "uniqueword"}
	if err := idx1.Index(ctx, doc.ID, doc); err != nil {
		t.Fatalf("Index: %v", err)
	}
	if err := idx1.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	// Opening an existing index recreates it (empty) so the mapping is correct; caller re-indexes.
	idx2, err := NewBleveIndex(indexPath)
	if err != nil {
		t.Fatalf("NewBleveIndex (open existing): %v", err)
	}
	defer func() {
		_ = idx2.Close()
	}()

	results, err := idx2.Search(ctx, "uniqueword", 10)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("after open existing, index is recreated empty; got %d results", len(results))
	}
}

func TestBleveIndex_Delete(t *testing.T) {
	dir := t.TempDir()
	indexPath := filepath.Join(dir, "bleve")

	idx, err := NewBleveIndex(indexPath)
	if err != nil {
		t.Fatalf("NewBleveIndex: %v", err)
	}
	defer func() {
		_ = idx.Close()
	}()

	ctx := context.Background()
	doc := &models.Document{ID: "doc1", Title: "T", Content: "onlyindoc1"}
	if err := idx.Index(ctx, doc.ID, doc); err != nil {
		t.Fatalf("Index: %v", err)
	}

	if err := idx.Delete(ctx, doc.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	results, err := idx.Search(ctx, "onlyindoc1", 10)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results after delete, got %d", len(results))
	}
}

func TestNewBleveIndex_createsDir(t *testing.T) {
	dir := t.TempDir()
	indexPath := filepath.Join(dir, "sub", "bleve")

	idx, err := NewBleveIndex(indexPath)
	if err != nil {
		t.Fatalf("NewBleveIndex: %v", err)
	}
	_ = idx.Close()

	if _, err := os.Stat(indexPath); err != nil {
		t.Errorf("index path should exist: %v", err)
	}
}
