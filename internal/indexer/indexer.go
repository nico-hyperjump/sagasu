// Package indexer provides document indexing into storage, keyword, and vector indices.
package indexer

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/hyperjump/sagasu/internal/config"
	"github.com/hyperjump/sagasu/internal/embedding"
	"github.com/hyperjump/sagasu/internal/extract"
	"github.com/hyperjump/sagasu/internal/fileid"
	"github.com/hyperjump/sagasu/internal/keyword"
	"github.com/hyperjump/sagasu/internal/models"
	"github.com/hyperjump/sagasu/internal/storage"
	"github.com/hyperjump/sagasu/internal/vector"
	"go.uber.org/zap"
)

// Indexer indexes documents into storage, keyword index, and vector index.
type Indexer struct {
	storage      storage.Storage
	embedder     embedding.Embedder
	vectorIndex  vector.VectorIndex
	keywordIndex keyword.KeywordIndex
	chunker      *Chunker
	config       *config.SearchConfig
	extractor    *extract.Extractor
	logger       *zap.Logger // optional; when set, logs debug events
}

// IndexerOption configures an Indexer.
type IndexerOption func(*Indexer)

// WithLogger sets a logger for debug output (file indexed, document deleted, etc.).
func WithLogger(l *zap.Logger) IndexerOption {
	return func(idx *Indexer) { idx.logger = l }
}

// NewIndexer creates an indexer with the given dependencies.
// extractor may be nil; when nil, IndexFile treats all files as plain text.
// Options (e.g. WithLogger) can be passed for debug logging.
func NewIndexer(
	storage storage.Storage,
	embedder embedding.Embedder,
	vectorIndex vector.VectorIndex,
	keywordIndex keyword.KeywordIndex,
	cfg *config.SearchConfig,
	extractor *extract.Extractor,
	opts ...IndexerOption,
) *Indexer {
	idx := &Indexer{
		storage:      storage,
		embedder:     embedder,
		vectorIndex:  vectorIndex,
		keywordIndex: keywordIndex,
		chunker:      NewChunker(cfg.ChunkSize, cfg.ChunkOverlap),
		config:       cfg,
		extractor:    extractor,
	}
	for _, opt := range opts {
		opt(idx)
	}
	return idx
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
	// Normalize title for keyword search: underscores as spaces so "hyperjump_company_profile_2021.pptx"
	// is searchable as "hyperjump company profile 2021" (standard analyzer does not split on underscore).
	docForKeyword := *doc
	docForKeyword.Title = normalizeTitleForKeywordSearch(doc.Title)
	if err := idx.keywordIndex.Index(ctx, doc.ID, &docForKeyword); err != nil {
		return fmt.Errorf("failed to index keywords: %w", err)
	}
	return nil
}

// normalizeTitleForKeywordSearch returns the title with underscores replaced by spaces
// so that Bleve's standard analyzer can match multi-word queries (e.g. "hyperjump profile")
// against filenames like "hyperjump_company_profile_2021.pptx".
func normalizeTitleForKeywordSearch(title string) string {
	return strings.ReplaceAll(title, "_", " ")
}

const (
	metaKeySourcePath  = "source_path"
	metaKeySourceMtime = "source_mtime"
	metaKeySourceSize  = "source_size"
)

// IndexFile reads a file from path and indexes it. The document ID is derived from the
// absolute path so re-indexing updates the same document. If allowedExts is non-nil and
// non-empty, the file's extension must be in the list (case-insensitive). Returns an error
// if the path is not a regular file, cannot be read, or indexing fails.
// Skips indexing if the file is already indexed with the same mtime and size (incremental sync).
func (idx *Indexer) IndexFile(ctx context.Context, path string, allowedExts []string) error {
	if idx.logger != nil {
		idx.logger.Debug("indexer indexing file", zap.String("path", path))
	}
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
	docID := fileid.FileDocID(absPath)
	if skip, err := idx.shouldSkipFile(ctx, absPath, docID, info); err != nil {
		return err
	} else if skip {
		// Ensure the doc is in the keyword index (repopulates if Bleve was opened empty).
		if doc, getErr := idx.storage.GetDocument(ctx, docID); getErr == nil {
			docForKeyword := *doc
			docForKeyword.Title = normalizeTitleForKeywordSearch(doc.Title)
			_ = idx.keywordIndex.Index(ctx, doc.ID, &docForKeyword)
		}
		if idx.logger != nil {
			idx.logger.Debug("indexer skipping unchanged file", zap.String("path", absPath))
		}
		return nil
	}
	text, err := idx.extractContent(absPath)
	if err != nil {
		return fmt.Errorf("extract content: %w", err)
	}
	_ = idx.DeleteDocument(ctx, docID)
	input := &models.DocumentInput{
		ID:    docID,
		Title: filepath.Base(absPath),
		Content: text,
		Metadata: map[string]interface{}{
			metaKeySourcePath:  absPath,
			metaKeySourceMtime: strconv.FormatInt(info.ModTime().UnixNano(), 10),
			metaKeySourceSize:  strconv.FormatInt(info.Size(), 10),
		},
	}
	if err := idx.IndexDocument(ctx, input); err != nil {
		return err
	}
	if idx.logger != nil {
		idx.logger.Debug("indexer file indexed", zap.String("path", absPath), zap.String("doc_id", docID))
	}
	return nil
}

// shouldSkipFile returns true if the file is already indexed with the same mtime and size.
func (idx *Indexer) shouldSkipFile(ctx context.Context, absPath, docID string, info os.FileInfo) (bool, error) {
	doc, err := idx.storage.GetDocument(ctx, docID)
	if err != nil {
		return false, nil
	}
	if doc.Metadata == nil {
		return false, nil
	}
	if doc.Metadata[metaKeySourcePath] != absPath {
		return false, nil
	}
	wantMtime := info.ModTime().UnixNano()
	wantSize := info.Size()
	// Values are stored as strings to avoid JSON float64 precision loss (UnixNano exceeds 53 bits).
	if metadataInt64(doc.Metadata, metaKeySourceMtime) != wantMtime || metadataInt64(doc.Metadata, metaKeySourceSize) != wantSize {
		return false, nil
	}
	return true, nil
}

func metadataInt64(m map[string]interface{}, key string) int64 {
	v, ok := m[key]
	if !ok {
		return 0
	}
	switch n := v.(type) {
	case string:
		x, _ := strconv.ParseInt(n, 10, 64)
		return x
	case int64:
		return n
	case int:
		return int64(n)
	case float64:
		return int64(n)
	default:
		return 0
	}
}

// IndexDirectory walks dir recursively and indexes each regular file whose extension
// is in allowedExts (if non-nil and non-empty; otherwise all files). Returns the number
// of files indexed and the first error encountered, if any.
func (idx *Indexer) IndexDirectory(ctx context.Context, dir string, allowedExts []string) (n int, err error) {
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return 0, fmt.Errorf("absolute path: %w", err)
	}
	info, err := os.Stat(absDir)
	if err != nil {
		return 0, fmt.Errorf("stat directory: %w", err)
	}
	if !info.IsDir() {
		return 0, fmt.Errorf("not a directory: %s", absDir)
	}
	err = filepath.WalkDir(absDir, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		ext := strings.ToLower(filepath.Ext(path))
		if len(allowedExts) > 0 && !extensionAllowed(ext, allowedExts) {
			return nil
		}
		// Resolve symlinks so we only index regular files
		finfo, statErr := os.Stat(path)
		if statErr != nil {
			return nil
		}
		if !finfo.Mode().IsRegular() {
			return nil
		}
		if indexErr := idx.IndexFile(ctx, path, allowedExts); indexErr != nil {
			return indexErr
		}
		n++
		return nil
	})
	return n, err
}

func (idx *Indexer) extractContent(path string) (string, error) {
	if idx.extractor != nil {
		return idx.extractor.Extract(path)
	}
	content, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(content), nil
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
	if idx.logger != nil {
		idx.logger.Debug("indexer deleting document", zap.String("id", id))
	}
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
	if idx.logger != nil {
		idx.logger.Debug("indexer document deleted", zap.String("id", id))
	}
	return nil
}
