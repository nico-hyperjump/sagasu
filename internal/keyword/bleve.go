// Package keyword provides Bleve implementation of KeywordIndex.
package keyword

import (
	"context"
	"fmt"

	"github.com/blevesearch/bleve/v2"
	"github.com/blevesearch/bleve/v2/analysis/lang/en"
	"github.com/hyperjump/sagasu/internal/models"
)

// BleveIndex implements KeywordIndex using Bleve.
type BleveIndex struct {
	index bleve.Index
}

// NewBleveIndex creates or opens a Bleve index at path.
func NewBleveIndex(path string) (*BleveIndex, error) {
	mapping := bleve.NewIndexMapping()

	docMapping := bleve.NewDocumentMapping()
	textFieldMapping := bleve.NewTextFieldMapping()
	textFieldMapping.Analyzer = en.AnalyzerName
	docMapping.AddFieldMappingsAt("content", textFieldMapping)
	docMapping.AddFieldMappingsAt("title", textFieldMapping)
	keywordFieldMapping := bleve.NewKeywordFieldMapping()
	docMapping.AddFieldMappingsAt("id", keywordFieldMapping)
	mapping.AddDocumentMapping("document", docMapping)

	index, err := bleve.New(path, mapping)
	if err == bleve.ErrorIndexPathExists {
		index, err = bleve.Open(path)
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
