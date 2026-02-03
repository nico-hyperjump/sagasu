package storage

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/hyperjump/sagasu/internal/models"
)

func TestSQLiteStorage_CRUD(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.db")
	store, err := NewSQLiteStorage(path)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	ctx := context.Background()

	doc := &models.Document{
		ID:       "doc1",
		Title:    "Title",
		Content:  "Content",
		Metadata: map[string]interface{}{"k": "v"},
	}
	if err := store.CreateDocument(ctx, doc); err != nil {
		t.Fatal(err)
	}
	if doc.CreatedAt.IsZero() {
		t.Error("CreatedAt should be set")
	}

	got, err := store.GetDocument(ctx, "doc1")
	if err != nil {
		t.Fatal(err)
	}
	if got.Title != "Title" || got.Content != "Content" {
		t.Errorf("got %+v", got)
	}

	doc.Title = "Updated"
	if err := store.UpdateDocument(ctx, doc); err != nil {
		t.Fatal(err)
	}
	got, _ = store.GetDocument(ctx, "doc1")
	if got.Title != "Updated" {
		t.Errorf("expected Updated, got %s", got.Title)
	}

	list, err := store.ListDocuments(ctx, 0, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 {
		t.Errorf("expected 1 doc, got %d", len(list))
	}

	if err := store.DeleteDocument(ctx, "doc1"); err != nil {
		t.Fatal(err)
	}
	_, err = store.GetDocument(ctx, "doc1")
	if err == nil {
		t.Error("expected error after delete")
	}
}

func TestSQLiteStorage_Chunks(t *testing.T) {
	path := filepath.Join(t.TempDir(), "chunks.db")
	store, err := NewSQLiteStorage(path)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	ctx := context.Background()

	doc := &models.Document{ID: "d1", Title: "T", Content: "C", Metadata: nil}
	_ = store.CreateDocument(ctx, doc)

	chunk := &models.DocumentChunk{
		ID: "d1_c1", DocumentID: "d1", Content: "chunk1", ChunkIndex: 0,
	}
	if err := store.CreateChunk(ctx, chunk); err != nil {
		t.Fatal(err)
	}
	chunks := []*models.DocumentChunk{
		{ID: "d1_c2", DocumentID: "d1", Content: "chunk2", ChunkIndex: 1},
		{ID: "d1_c3", DocumentID: "d1", Content: "chunk3", ChunkIndex: 2},
	}
	if err := store.BatchCreateChunks(ctx, chunks); err != nil {
		t.Fatal(err)
	}

	list, err := store.GetChunksByDocumentID(ctx, "d1")
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 3 {
		t.Errorf("expected 3 chunks, got %d", len(list))
	}

	got, err := store.GetChunk(ctx, "d1_c2")
	if err != nil {
		t.Fatal(err)
	}
	if got.Content != "chunk2" {
		t.Errorf("got %s", got.Content)
	}

	if err := store.DeleteChunksByDocumentID(ctx, "d1"); err != nil {
		t.Fatal(err)
	}
	list, _ = store.GetChunksByDocumentID(ctx, "d1")
	if len(list) != 0 {
		t.Errorf("expected 0 chunks after delete, got %d", len(list))
	}
}

func TestSQLiteStorage_Counts(t *testing.T) {
	path := filepath.Join(t.TempDir(), "count.db")
	store, err := NewSQLiteStorage(path)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	ctx := context.Background()

	n, err := store.CountDocuments(ctx)
	if err != nil || n != 0 {
		t.Errorf("CountDocuments: %v, %d", err, n)
	}
	_ = store.CreateDocument(ctx, &models.Document{ID: "x", Content: "c", Metadata: nil})
	n, _ = store.CountDocuments(ctx)
	if n != 1 {
		t.Errorf("expected 1 document, got %d", n)
	}
}
