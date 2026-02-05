// Package search provides the main hybrid search engine.
package search

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/hyperjump/sagasu/internal/config"
	"github.com/hyperjump/sagasu/internal/embedding"
	"github.com/hyperjump/sagasu/internal/keyword"
	"github.com/hyperjump/sagasu/internal/models"
	"github.com/hyperjump/sagasu/internal/ranking"
	"github.com/hyperjump/sagasu/internal/storage"
	"github.com/hyperjump/sagasu/internal/vector"
)

// Engine runs hybrid (keyword + semantic) search.
type Engine struct {
	storage       storage.Storage
	embedder      embedding.Embedder
	vectorIndex   vector.VectorIndex
	keywordIndex  keyword.KeywordIndex
	config        *config.SearchConfig
	ranker        *ranking.Ranker
	rankingConfig *config.RankingConfig
	spellChecker  *keyword.SpellChecker
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

// WithRanking enables content-aware ranking with the given configuration.
func (e *Engine) WithRanking(cfg *config.RankingConfig) *Engine {
	e.rankingConfig = cfg
	if cfg != nil {
		e.ranker = ranking.NewRanker(configToRankingConfig(cfg))
	}
	return e
}

// WithSpellChecker enables spell checking for "Did you mean?" suggestions.
// The keywordIndex must implement the TermDictionary interface.
func (e *Engine) WithSpellChecker() *Engine {
	// Check if keywordIndex implements TermDictionary
	if dict, ok := e.keywordIndex.(keyword.TermDictionary); ok {
		e.spellChecker = keyword.NewSpellChecker(dict,
			keyword.WithMaxDistance(2),
			keyword.WithMinFrequency(1),
			keyword.WithMaxSuggestions(3),
		)
	}
	return e
}

// RefreshSpellChecker refreshes the spell checker's term dictionary cache.
// Call this after indexing new documents.
func (e *Engine) RefreshSpellChecker() error {
	if e.spellChecker != nil {
		return e.spellChecker.RefreshCache()
	}
	return nil
}

// configToRankingConfig converts config.RankingConfig to ranking.RankingConfig.
func configToRankingConfig(cfg *config.RankingConfig) *ranking.RankingConfig {
	if cfg == nil {
		return nil
	}
	return &ranking.RankingConfig{
		FilenameWeight:          cfg.FilenameWeight,
		ContentWeight:           cfg.ContentWeight,
		PathWeight:              cfg.PathWeight,
		MetadataWeight:          cfg.MetadataWeight,
		ExactFilenameScore:      cfg.ExactFilenameScore,
		AllWordsInOrderScore:    cfg.AllWordsInOrderScore,
		AllWordsAnyOrderScore:   cfg.AllWordsAnyOrderScore,
		SubstringMatchScore:     cfg.SubstringMatchScore,
		PrefixMatchScore:        cfg.PrefixMatchScore,
		MultipleOccurrenceBonus: cfg.MultipleOccurrenceBonus,
		ExtensionMatchScore:     cfg.ExtensionMatchScore,
		PhraseMatchScore:        cfg.PhraseMatchScore,
		HeaderMatchScore:        cfg.HeaderMatchScore,
		AllWordsContentScore:    cfg.AllWordsContentScore,
		ScatteredWordsScore:     cfg.ScatteredWordsScore,
		StemmingMatchScore:      cfg.StemmingMatchScore,
		PathExactMatchScore:     cfg.PathExactMatchScore,
		PathPartialMatchScore:   cfg.PathPartialMatchScore,
		PathComponentBonus:      cfg.PathComponentBonus,
		AuthorMatchScore:        cfg.AuthorMatchScore,
		TagMatchScore:           cfg.TagMatchScore,
		OtherMetadataScore:      cfg.OtherMetadataScore,
		MaxTFIDFMultiplier:      cfg.MaxTFIDFMultiplier,
		TFIDFEnabled:            cfg.TFIDFEnabled,
		PositionBoostEnabled:    cfg.PositionBoostEnabled,
		PositionBoostThreshold:  cfg.PositionBoostThreshold,
		PositionBoostMultiplier: cfg.PositionBoostMultiplier,
		RecencyEnabled:          cfg.RecencyEnabled,
		Recency24hMultiplier:    cfg.Recency24hMultiplier,
		RecencyWeekMultiplier:   cfg.RecencyWeekMultiplier,
		RecencyMonthMultiplier:  cfg.RecencyMonthMultiplier,
		QueryQualityEnabled:     cfg.QueryQualityEnabled,
		PhraseMatchMultiplier:   cfg.PhraseMatchMultiplier,
		AllWordsMultiplier:      cfg.AllWordsMultiplier,
		PartialMatchMultiplier:  cfg.PartialMatchMultiplier,
		FileSizeNormEnabled:     cfg.FileSizeNormEnabled,
	}
}

// UpdateCorpusStats updates the ranker's corpus statistics for IDF calculation.
func (e *Engine) UpdateCorpusStats(ctx context.Context) error {
	if e.ranker == nil {
		return nil
	}

	// Get all documents for corpus stats
	docs, err := e.storage.ListDocuments(ctx, 0, 10000) // Get up to 10k docs for stats
	if err != nil {
		return fmt.Errorf("failed to list documents for corpus stats: %w", err)
	}

	e.ranker.UpdateCorpusStats(docs)
	return nil
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
				TitleBoost:   e.config.KeywordTitleBoost,
				PhraseBoost:  e.config.KeywordPhraseBoost,
				FuzzyEnabled: query.FuzzyEnabled,
				Fuzziness:    2, // default fuzziness level
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

	// Collect documents for potential re-ranking
	var nonSemanticDocs []*models.SearchResult
	for _, r := range nonSemanticPaged {
		doc, err := e.storage.GetDocument(ctx, r.DocumentID)
		if err != nil {
			continue
		}
		nonSemanticDocs = append(nonSemanticDocs, &models.SearchResult{
			Document:      doc,
			Score:         r.Score,
			KeywordScore:  r.KeywordScore,
			SemanticScore: r.SemanticScore,
		})
	}

	var semanticDocs []*models.SearchResult
	for _, r := range semanticPaged {
		doc, err := e.storage.GetDocument(ctx, r.DocumentID)
		if err != nil {
			continue
		}
		semanticDocs = append(semanticDocs, &models.SearchResult{
			Document:      doc,
			Score:         r.Score,
			KeywordScore:  r.KeywordScore,
			SemanticScore: r.SemanticScore,
		})
	}

	// Apply content-aware re-ranking if enabled
	if e.ranker != nil && e.config.RankingEnabled {
		nonSemanticDocs = e.reRankResults(query.Query, nonSemanticDocs)
		semanticDocs = e.reRankResults(query.Query, semanticDocs)
	}

	// Assign final ranks
	for i := range nonSemanticDocs {
		nonSemanticDocs[i].Rank = i + 1
	}
	for i := range semanticDocs {
		semanticDocs[i].Rank = i + 1
	}

	response.NonSemanticResults = nonSemanticDocs
	response.SemanticResults = semanticDocs

	// Add spell check suggestions if fuzzy is enabled and spell checker is available
	if query.FuzzyEnabled && e.spellChecker != nil {
		suggestions := e.spellChecker.GetTopSuggestions(query.Query, 3)
		if len(suggestions) > 0 {
			response.Suggestions = suggestions
		}
	}

	return response, nil
}

// reRankResults re-ranks search results using the content-aware ranker.
func (e *Engine) reRankResults(queryStr string, results []*models.SearchResult) []*models.SearchResult {
	if e.ranker == nil || len(results) == 0 {
		return results
	}

	analyzedQuery := e.ranker.AnalyzeQuery(queryStr)

	for _, result := range results {
		if result.Document != nil {
			// Calculate new score using the ranker
			newScore := e.ranker.Rank(analyzedQuery, result.Document)
			result.Score = newScore
		}
	}

	// Sort by new scores (descending)
	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	return results
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

// VectorIndexType returns the type of vector index being used (e.g., "memory", "faiss").
func (e *Engine) VectorIndexType() string {
	return e.vectorIndex.Type()
}
