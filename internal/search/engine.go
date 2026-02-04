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

	if query.KeywordEnabled {
		wg.Add(1)
		go func() {
			defer wg.Done()
			kwOpts := &keyword.SearchOptions{
				TitleBoost:  e.config.KeywordTitleBoost,
				PhraseBoost: e.config.KeywordPhraseBoost,
			}
			results, err := e.keywordIndex.Search(ctx, query.Query, e.config.TopKCandidates, kwOpts)
			if err != nil {
				errChan <- fmt.Errorf("keyword search failed: %w", err)
				return
			}
			keywordResults = results
		}()
	}

	if query.SemanticEnabled {
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
	nonSemanticFused, semanticFused := SplitBySource(keywordScores, semanticByDoc)

	minKeywordScore := resolveMinKeywordScore(query, e.config)
	minSemanticScore := resolveMinSemanticScore(query, e.config)
	if minKeywordScore > 0 {
		nonSemanticFused = filterByMinScore(nonSemanticFused, minKeywordScore)
	}
	if minSemanticScore > 0 {
		semanticFused = filterByMinScore(semanticFused, minSemanticScore)
	}

	totalNonSemantic := len(nonSemanticFused)
	totalSemantic := len(semanticFused)
	nonSemanticPaged := pageResults(nonSemanticFused, query.Offset, query.Limit)
	semanticPaged := pageResults(semanticFused, query.Offset, query.Limit)

	response := &models.SearchResponse{
		NonSemanticResults: make([]*models.SearchResult, 0, len(nonSemanticPaged)),
		SemanticResults:    make([]*models.SearchResult, 0, len(semanticPaged)),
		TotalNonSemantic:   totalNonSemantic,
		TotalSemantic:      totalSemantic,
		QueryTime:          time.Since(startTime).Milliseconds(),
		Query:              query.Query,
	}

	for i, r := range nonSemanticPaged {
		doc, err := e.storage.GetDocument(ctx, r.DocumentID)
		if err != nil {
			continue
		}
		response.NonSemanticResults = append(response.NonSemanticResults, &models.SearchResult{
			Document:      doc,
			Score:         r.Score,
			KeywordScore:  r.KeywordScore,
			SemanticScore: r.SemanticScore,
			Rank:          i + 1,
		})
	}
	for i, r := range semanticPaged {
		doc, err := e.storage.GetDocument(ctx, r.DocumentID)
		if err != nil {
			continue
		}
		response.SemanticResults = append(response.SemanticResults, &models.SearchResult{
			Document:      doc,
			Score:         r.Score,
			KeywordScore:  r.KeywordScore,
			SemanticScore: r.SemanticScore,
			Rank:          i + 1,
		})
	}
	return response, nil
}

// resolveMinKeywordScore returns the effective minimum score for keyword results:
// MinKeywordScore if set, else legacy MinScore, else config default.
func resolveMinKeywordScore(query *models.SearchQuery, cfg *config.SearchConfig) float64 {
	if query.MinKeywordScore > 0 {
		return query.MinKeywordScore
	}
	if query.MinScore > 0 {
		return query.MinScore
	}
	return cfg.DefaultMinKeywordScore
}

// resolveMinSemanticScore returns the effective minimum score for semantic results:
// MinSemanticScore if set, else legacy MinScore, else config default.
func resolveMinSemanticScore(query *models.SearchQuery, cfg *config.SearchConfig) float64 {
	if query.MinSemanticScore > 0 {
		return query.MinSemanticScore
	}
	if query.MinScore > 0 {
		return query.MinScore
	}
	return cfg.DefaultMinSemanticScore
}

func filterByMinScore(results []*FusedResult, minScore float64) []*FusedResult {
	filtered := results[:0]
	for _, r := range results {
		if r.Score >= minScore {
			filtered = append(filtered, r)
		}
	}
	return filtered
}

func pageResults(results []*FusedResult, offset, limit int) []*FusedResult {
	start := offset
	end := offset + limit
	if start > len(results) {
		start = len(results)
	}
	if end > len(results) {
		end = len(results)
	}
	return results[start:end]
}

// VectorIndexSize returns the number of vectors in the semantic index.
func (e *Engine) VectorIndexSize() int {
	return e.vectorIndex.Size()
}
