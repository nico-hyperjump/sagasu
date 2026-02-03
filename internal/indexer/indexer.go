// Package indexer provides document indexing into storage, keyword, and vector indices.
package indexer

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"github.com/hyperjump/sagasu/internal/config"
	"github.com/hyperjump/sagasu/internal/embedding"
	"github.com/hyperjump/sagasu/internal/fileid"
	"github.com/hyperjump/sagasu/internal/keyword"
	"github.com/hyperjump/sagasu/internal/models"
	"github.com/hyperjump/sagasu/internal/storage"
	"github.com/hyperjump/sagasu/internal/vector"
)

// Indexer indexes documents into storage, keyword index, and vector index.
type Indexer struct {
	storage      storage.Storage
	embedder     embedding.Embedder
	vectorIndex  vector.VectorIndex
	keywordIndex keyword.KeywordIndex
	chunker      *Chunker
	config       *config.SearchConfig
}

// NewIndexer creates an indexer with the given dependencies.
func NewIndexer(
	storage storage.Storage,
	embedder embedding.Embedder,
	vectorIndex vector.VectorIndex,
	keywordIndex keyword.KeywordIndex,
	cfg *config.SearchConfig,
) *Indexer {
	return &Indexer{
		storage:      storage,
		embedder:     embedder,
		vectorIndex:  vectorIndex,
		keywordIndex: keywordIndex,
		chunker:      NewChunker(cfg.ChunkSize, cfg.ChunkOverlap),
		config:       cfg,
	}
}

// IndexDocument indexes a document: store, chunk, embed, index in vector and keyword.
func (idx *Indexer) IndexDocument(ctx context.Context, input *models.DocumentInput) error {
	if input.ID == "" {
		input.ID = uuid.New().String()
	}
	doc := &models.Document{
		ID:       input.ID,
		Title:    input.Title,
		Content:  Preprocess(input.Content),
		Metadata: input.Metadata,
	}
	if err := idx.storage.CreateDocument(ctx, doc); err != nil {
		return fmt.Errorf("failed to store document: %w", err)
	}
	chunks := idx.chunker.Chunk(doc.ID, doc.Content)
	if len(chunks) == 0 {
		chunks = []*models.DocumentChunk{{
			ID:         doc.ID + "_0",
			DocumentID: doc.ID,
			Content:    doc.Content,
			ChunkIndex: 0,
		}}
	}
	texts := make([]string, len(chunks))
	for i, ch := range chunks {
		texts[i] = ch.Content
	}
	embeddings, err := idx.embedder.EmbedBatch(ctx, texts)
	if err != nil {
		return fmt.Errorf("failed to generate embeddings: %w", err)
	}
	for i := range chunks {
		chunks[i].Embedding = embeddings[i]
	}
	if err := idx.storage.BatchCreateChunks(ctx, chunks); err != nil {
		return fmt.Errorf("failed to store chunks: %w", err)
	}
	chunkIDs := make([]string, len(chunks))
	for i, ch := range chunks {
		chunkIDs[i] = ch.ID
	}
	if err := idx.vectorIndex.Add(ctx, chunkIDs, embeddings); err != nil {
		return fmt.Errorf("failed to index vectors: %w", err)
	}
	if err := idx.keywordIndex.Index(ctx, doc.ID, doc); err != nil {
		return fmt.Errorf("failed to index keywords: %w", err)
	}
	return nil
}

// IndexFile reads a file from path and indexes it. The document ID is derived from the
// absolute path so re-indexing updates the same document. If allowedExts is non-nil and
// non-empty, the file's extension must be in the list (case-insensitive). Returns an error
// if the path is not a regular file, cannot be read, or indexing fails.
func (idx *Indexer) IndexFile(ctx context.Context, path string, allowedExts []string) error {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("absolute path: %w", err)
	}
	ext := strings.ToLower(filepath.Ext(absPath))
	if len(allowedExts) > 0 && !extensionAllowed(ext, allowedExts) {
		return fmt.Errorf("extension %q not in allowed list", ext)
	}
	info, err := os.Stat(absPath)
	if err != nil {
		return fmt.Errorf("stat file: %w", err)
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("not a regular file: %s", absPath)
	}
	content, err := os.ReadFile(absPath)
	if err != nil {
		return fmt.Errorf("read file: %w", err)
	}
	docID := fileid.FileDocID(absPath)
	_ = idx.DeleteDocument(ctx, docID)
	input := &models.DocumentInput{
		ID:       docID,
		Title:    filepath.Base(absPath),
		Content:  string(content),
		Metadata: map[string]interface{}{"source_path": absPath},
	}
	return idx.IndexDocument(ctx, input)
}

func extensionAllowed(ext string, allowed []string) bool {
	extNorm := strings.ToLower(strings.TrimPrefix(ext, "."))
	for _, a := range allowed {
		if strings.ToLower(strings.TrimPrefix(a, ".")) == extNorm {
			return true
		}
	}
	return false
}

// DeleteDocument removes a document from all indices and storage.
func (idx *Indexer) DeleteDocument(ctx context.Context, id string) error {
	if err := idx.keywordIndex.Delete(ctx, id); err != nil {
		return fmt.Errorf("failed to delete from keyword index: %w", err)
	}
	chunks, err := idx.storage.GetChunksByDocumentID(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to get chunks: %w", err)
	}
	chunkIDs := make([]string, len(chunks))
	for i, ch := range chunks {
		chunkIDs[i] = ch.ID
	}
	if err := idx.vectorIndex.Remove(ctx, chunkIDs); err != nil {
		return fmt.Errorf("failed to delete from vector index: %w", err)
	}
	if err := idx.storage.DeleteChunksByDocumentID(ctx, id); err != nil {
		return fmt.Errorf("failed to delete chunks: %w", err)
	}
	if err := idx.storage.DeleteDocument(ctx, id); err != nil {
		return fmt.Errorf("failed to delete document: %w", err)
	}
	return nil
}
