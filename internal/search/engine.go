// Package search provides the main hybrid search engine.
package search

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/hyperjump/sagasu/internal/config"
	"github.com/hyperjump/sagasu/internal/embedding"
	"github.com/hyperjump/sagasu/internal/keyword"
	"github.com/hyperjump/sagasu/internal/models"
	"github.com/hyperjump/sagasu/internal/storage"
	"github.com/hyperjump/sagasu/internal/vector"
)

// Engine runs hybrid (keyword + semantic) search.
type Engine struct {
	storage      storage.Storage
	embedder     embedding.Embedder
	vectorIndex  vector.VectorIndex
	keywordIndex keyword.KeywordIndex
	config       *config.SearchConfig
}

// NewEngine creates a search engine with the given dependencies.
func NewEngine(
	storage storage.Storage,
	embedder embedding.Embedder,
	vectorIndex vector.VectorIndex,
	keywordIndex keyword.KeywordIndex,
	cfg *config.SearchConfig,
) *Engine {
	return &Engine{
		storage:      storage,
		embedder:     embedder,
		vectorIndex:  vectorIndex,
		keywordIndex: keywordIndex,
		config:       cfg,
	}
}

// Search runs hybrid search and returns document-level results.
func (e *Engine) Search(ctx context.Context, query *models.SearchQuery) (*models.SearchResponse, error) {
	startTime := time.Now()
	if err := ProcessQuery(query); err != nil {
		return nil, err
	}

	var (
		keywordResults  []*keyword.KeywordResult
		semanticResults []*vector.VectorResult
		errChan         = make(chan error, 2)
		wg              sync.WaitGroup
	)

	if query.KeywordWeight > 0 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			results, err := e.keywordIndex.Search(ctx, query.Query, e.config.TopKCandidates)
			if err != nil {
				errChan <- fmt.Errorf("keyword search failed: %w", err)
				return
			}
			keywordResults = results
		}()
	}

	if query.SemanticWeight > 0 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			queryEmbedding, err := e.embedder.Embed(ctx, query.Query)
			if err != nil {
				errChan <- fmt.Errorf("embedding failed: %w", err)
				return
			}
			results, err := e.vectorIndex.Search(ctx, queryEmbedding, e.config.TopKCandidates)
			if err != nil {
				errChan <- fmt.Errorf("vector search failed: %w", err)
				return
			}
			semanticResults = results
		}()
	}

	wg.Wait()
	close(errChan)
	for err := range errChan {
		if err != nil {
			return nil, err
		}
	}

	keywordScores := NormalizeKeywordScores(keywordResults)
	semanticByChunk := NormalizeSemanticScores(semanticResults)
	chunkToDoc := make(map[string]string)
	for _, r := range semanticResults {
		chunk, err := e.storage.GetChunk(ctx, r.ID)
		if err != nil {
			continue
		}
		chunkToDoc[r.ID] = chunk.DocumentID
	}
	semanticByDoc := AggregateSemanticByDocument(chunkToDoc, semanticByChunk)
	fusedResults := Fuse(keywordScores, semanticByDoc, query.KeywordWeight, query.SemanticWeight)

	if query.MinScore > 0 {
		filtered := fusedResults[:0]
		for _, r := range fusedResults {
			if r.Score >= query.MinScore {
				filtered = append(filtered, r)
			}
		}
		fusedResults = filtered
	}

	start := query.Offset
	end := query.Offset + query.Limit
	if start > len(fusedResults) {
		start = len(fusedResults)
	}
	if end > len(fusedResults) {
		end = len(fusedResults)
	}
	pagedResults := fusedResults[start:end]

	response := &models.SearchResponse{
		Results:   make([]*models.SearchResult, 0, len(pagedResults)),
		Total:     len(fusedResults),
		QueryTime: time.Since(startTime).Milliseconds(),
		Query:     query.Query,
	}

	for i, fusedResult := range pagedResults {
		doc, err := e.storage.GetDocument(ctx, fusedResult.DocumentID)
		if err != nil {
			continue
		}
		response.Results = append(response.Results, &models.SearchResult{
			Document:      doc,
			Score:         fusedResult.Score,
			KeywordScore:  fusedResult.KeywordScore,
			SemanticScore: fusedResult.SemanticScore,
			Rank:          start + i + 1,
		})
	}
	return response, nil
}
