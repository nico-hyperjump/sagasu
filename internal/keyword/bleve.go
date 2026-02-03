// Package keyword provides Bleve implementation of KeywordIndex.
package keyword

import (
	"context"
	"fmt"
	"os"

	"github.com/blevesearch/bleve/v2"
	"github.com/blevesearch/bleve/v2/analysis/analyzer/standard"
	"github.com/hyperjump/sagasu/internal/models"
)

// BleveIndex implements KeywordIndex using Bleve.
type BleveIndex struct {
	index bleve.Index
}

// NewBleveIndex creates or opens a Bleve index at path.
// If the index already exists, we always remove and recreate it so the mapping
// (content/title searchable) is guaranteed. The caller must re-index documents
// (e.g. server runs SyncExistingFiles after this).
func NewBleveIndex(path string) (*BleveIndex, error) {
	im := bleve.NewIndexMapping()

	docMapping := bleve.NewDocumentMapping()
	textFieldMapping := bleve.NewTextFieldMapping()
	// Use standard analyzer (lowercase + tokenize, no stemming) so queries like "bayes" match
	// the exact word; English analyzer stems e.g. "Bayesian" -> "bayesi" and "bayes" -> "bay", so they don't match.
	textFieldMapping.Analyzer = standard.Name
	docMapping.AddFieldMappingsAt("content", textFieldMapping)
	docMapping.AddFieldMappingsAt("title", textFieldMapping)
	keywordFieldMapping := bleve.NewKeywordFieldMapping()
	docMapping.AddFieldMappingsAt("id", keywordFieldMapping)
	im.AddDocumentMapping("document", docMapping)
	im.DefaultType = "document"
	im.DefaultMapping = docMapping // so _default type also indexes content/title

	index, err := bleve.New(path, im)
	if err == bleve.ErrorIndexPathExists {
		index, err = bleve.Open(path)
		if err != nil {
			return nil, fmt.Errorf("failed to open Bleve index: %w", err)
		}
		// Persisted mapping cannot be changed. Recreate the index with our mapping
		// so content/title are searchable. Caller re-indexes via SyncExistingFiles.
		_ = index.Close()
		if err := os.RemoveAll(path); err != nil {
			return nil, fmt.Errorf("keyword index path %s: remove for recreate: %w", path, err)
		}
		index, err = bleve.New(path, im)
		if err != nil {
			return nil, fmt.Errorf("failed to recreate Bleve index: %w", err)
		}
	}
	if err != nil {
		return nil, fmt.Errorf("failed to create/open Bleve index: %w", err)
	}

	return &BleveIndex{index: index}, nil
}

// Index indexes a document by id.
func (b *BleveIndex) Index(ctx context.Context, id string, doc *models.Document) error {
	return b.index.Index(id, doc)
}

// Search runs a match query and returns up to limit results.
func (b *BleveIndex) Search(ctx context.Context, query string, limit int) ([]*KeywordResult, error) {
	q := bleve.NewMatchQuery(query)
	search := bleve.NewSearchRequest(q)
	search.Size = limit
	search.Fields = []string{"*"}

	results, err := b.index.Search(search)
	if err != nil {
		return nil, fmt.Errorf("Bleve search failed: %w", err)
	}

	keywordResults := make([]*KeywordResult, len(results.Hits))
	for i, hit := range results.Hits {
		keywordResults[i] = &KeywordResult{
			ID:    hit.ID,
			Score: hit.Score,
		}
	}
	return keywordResults, nil
}

// Delete removes a document from the index.
func (b *BleveIndex) Delete(ctx context.Context, id string) error {
	return b.index.Delete(id)
}

// Close closes the Bleve index.
func (b *BleveIndex) Close() error {
	return b.index.Close()
}
