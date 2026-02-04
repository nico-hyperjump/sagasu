// Package keyword provides Bleve implementation of KeywordIndex.
package keyword

import (
	"context"
	"fmt"
	"os"
	"sort"

	"github.com/blevesearch/bleve/v2"
	"github.com/blevesearch/bleve/v2/analysis/analyzer/standard"
	"github.com/hyperjump/sagasu/internal/models"
)

// BleveIndex implements KeywordIndex using Bleve.
type BleveIndex struct {
	index bleve.Index
}

// NewBleveIndex creates or opens a Bleve index at path.
// If the path already exists, the existing index is opened and reused so that
// keyword search works with incremental sync (unchanged files are not re-indexed).
// If you change the index mapping in code, remove the index directory to force a full re-index.
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

	if _, err := os.Stat(path); err == nil {
		index, openErr := bleve.Open(path)
		if openErr != nil {
			return nil, fmt.Errorf("failed to open Bleve index: %w", openErr)
		}
		return &BleveIndex{index: index}, nil
	}

	index, err := bleve.New(path, im)
	if err != nil {
		return nil, fmt.Errorf("failed to create Bleve index: %w", err)
	}
	return &BleveIndex{index: index}, nil
}

// Index indexes a document by id.
func (b *BleveIndex) Index(ctx context.Context, id string, doc *models.Document) error {
	return b.index.Index(id, doc)
}

// Search runs a match query and returns up to limit results.
// When opts is nil or TitleBoost <= 1, a single match over title+content is used (original behavior).
// When opts.TitleBoost > 1, we run separate title and content queries and merge with
// score = max(titleScore*boost, contentScore) so that title-only matches can compete with content-heavy docs.
func (b *BleveIndex) Search(ctx context.Context, query string, limit int, opts *SearchOptions) ([]*KeywordResult, error) {
	titleBoost := 1.0
	if opts != nil && opts.TitleBoost > 0 {
		titleBoost = opts.TitleBoost
	}

	if titleBoost <= 1.0 {
		return b.searchSingle(ctx, query, limit)
	}
	return b.searchWithTitleBoost(ctx, query, limit, titleBoost)
}

// searchSingle runs one MatchQuery over all fields (original behavior).
func (b *BleveIndex) searchSingle(ctx context.Context, query string, limit int) ([]*KeywordResult, error) {
	q := bleve.NewMatchQuery(query)
	search := bleve.NewSearchRequest(q)
	search.Size = limit
	search.Fields = []string{"*"}
	results, err := b.index.Search(search)
	if err != nil {
		return nil, fmt.Errorf("Bleve search failed: %w", err)
	}
	out := make([]*KeywordResult, len(results.Hits))
	for i, hit := range results.Hits {
		out[i] = &KeywordResult{ID: hit.ID, Score: hit.Score}
	}
	return out, nil
}

// searchWithTitleBoost runs separate title and content queries, then merges by score = max(boost*titleScore, contentScore).
func (b *BleveIndex) searchWithTitleBoost(ctx context.Context, query string, limit int, titleBoost float64) ([]*KeywordResult, error) {
	// Request enough from each so merged top "limit" is correct (same doc can appear in both).
	reqSize := limit * 2
	if reqSize < 50 {
		reqSize = 50
	}

	titleQuery := bleve.NewMatchQuery(query)
	titleQuery.SetField("title")
	titleReq := bleve.NewSearchRequest(titleQuery)
	titleReq.Size = reqSize
	titleReq.Fields = []string{"*"}

	contentQuery := bleve.NewMatchQuery(query)
	contentQuery.SetField("content")
	contentReq := bleve.NewSearchRequest(contentQuery)
	contentReq.Size = reqSize
	contentReq.Fields = []string{"*"}

	titleResults, err := b.index.Search(titleReq)
	if err != nil {
		return nil, fmt.Errorf("Bleve title search failed: %w", err)
	}
	contentResults, err := b.index.Search(contentReq)
	if err != nil {
		return nil, fmt.Errorf("Bleve content search failed: %w", err)
	}

	// Merge: for each doc ID, score = max(titleScore*boost, contentScore)
	scores := make(map[string]float64)
	for _, hit := range titleResults.Hits {
		s := hit.Score * titleBoost
		if s > scores[hit.ID] {
			scores[hit.ID] = s
		}
	}
	for _, hit := range contentResults.Hits {
		if hit.Score > scores[hit.ID] {
			scores[hit.ID] = hit.Score
		}
	}

	// Sort by score desc and take top limit
	type scored struct {
		id    string
		score float64
	}
	merged := make([]scored, 0, len(scores))
	for id, score := range scores {
		merged = append(merged, scored{id: id, score: score})
	}
	sort.Slice(merged, func(i, j int) bool { return merged[i].score > merged[j].score })
	if len(merged) > limit {
		merged = merged[:limit]
	}

	out := make([]*KeywordResult, len(merged))
	for i, s := range merged {
		out[i] = &KeywordResult{ID: s.id, Score: s.score}
	}
	return out, nil
}

// Delete removes a document from the index.
func (b *BleveIndex) Delete(ctx context.Context, id string) error {
	return b.index.Delete(id)
}

// Close closes the Bleve index.
func (b *BleveIndex) Close() error {
	return b.index.Close()
}
