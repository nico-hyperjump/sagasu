package ranking

import (
	"sort"

	"github.com/hyperjump/sagasu/internal/models"
)

// Ranker combines all scorers and multipliers to rank documents.
type Ranker struct {
	config         *RankingConfig
	analyzer       *QueryAnalyzer
	filenameScorer *FilenameScorer
	contentScorer  *ContentScorer
	pathScorer     *PathScorer
	metadataScorer *MetadataScorer
	multipliers    []Multiplier
	corpusStats    *CorpusStats
}

// NewRanker creates a new Ranker with the given configuration.
func NewRanker(config *RankingConfig) *Ranker {
	if config == nil {
		config = DefaultRankingConfig()
	}
	config.ApplyDefaults()

	return &Ranker{
		config:         config,
		analyzer:       NewQueryAnalyzer(),
		filenameScorer: NewFilenameScorer(config),
		contentScorer:  NewContentScorer(config),
		pathScorer:     NewPathScorer(config),
		metadataScorer: NewMetadataScorer(config),
		multipliers:    DefaultMultipliers(config),
		corpusStats:    NewCorpusStats(),
	}
}

// WithCorpusStats sets the corpus statistics for IDF calculation.
func (r *Ranker) WithCorpusStats(stats *CorpusStats) *Ranker {
	r.corpusStats = stats
	return r
}

// WithMultipliers sets custom multipliers.
func (r *Ranker) WithMultipliers(multipliers []Multiplier) *Ranker {
	r.multipliers = multipliers
	return r
}

// AnalyzeQuery parses and analyzes a query string.
func (r *Ranker) AnalyzeQuery(query string) *AnalyzedQuery {
	return r.analyzer.Analyze(query)
}

// Rank calculates the final score for a document given an analyzed query.
func (r *Ranker) Rank(query *AnalyzedQuery, doc *models.Document) float64 {
	ctx := NewScoringContext(query, doc, r.corpusStats)
	return r.RankWithContext(ctx)
}

// RankWithContext calculates the final score using a pre-built context.
func (r *Ranker) RankWithContext(ctx *ScoringContext) float64 {
	// Calculate base scores from each scorer
	filenameScore := r.filenameScorer.Score(ctx)
	contentScore := r.contentScorer.Score(ctx)
	pathScore := r.pathScorer.Score(ctx)
	metadataScore := r.metadataScorer.Score(ctx)

	// Apply weighted sum formula:
	// Score = (Wf * Sf) + (Wc * Sc) + (Wp * Sp) + (Wm * Sm)
	score := (r.config.FilenameWeight * filenameScore) +
		(r.config.ContentWeight * contentScore) +
		(r.config.PathWeight * pathScore) +
		(r.config.MetadataWeight * metadataScore)

	// Apply multipliers
	for _, m := range r.multipliers {
		score = m.Multiply(ctx, score)
	}

	return score
}

// RankWithBreakdown returns detailed scoring information.
func (r *Ranker) RankWithBreakdown(query *AnalyzedQuery, doc *models.Document) *ScoreBreakdown {
	ctx := NewScoringContext(query, doc, r.corpusStats)
	breakdown := NewScoreBreakdown()

	// Calculate individual scores
	breakdown.FilenameScore = r.filenameScorer.Score(ctx)
	breakdown.ContentScore = r.contentScorer.Score(ctx)
	breakdown.PathScore = r.pathScorer.Score(ctx)
	breakdown.MetadataScore = r.metadataScorer.Score(ctx)

	// Calculate weighted sum
	baseScore := (r.config.FilenameWeight * breakdown.FilenameScore) +
		(r.config.ContentWeight * breakdown.ContentScore) +
		(r.config.PathWeight * breakdown.PathScore) +
		(r.config.MetadataWeight * breakdown.MetadataScore)

	// Apply multipliers and track their values
	score := baseScore
	for _, m := range r.multipliers {
		prevScore := score
		score = m.Multiply(ctx, score)
		if prevScore != 0 {
			breakdown.Multipliers[m.Name()] = score / prevScore
		} else {
			breakdown.Multipliers[m.Name()] = 1.0
		}
	}

	breakdown.FinalScore = score

	// Determine best match type
	breakdown.MatchType = r.determineBestMatchType(ctx)

	return breakdown
}

// determineBestMatchType determines the best match type across all scorers.
func (r *Ranker) determineBestMatchType(ctx *ScoringContext) MatchType {
	if ctx.Query == nil || ctx.Document == nil {
		return MatchTypeNone
	}

	tokens := r.analyzer.TokenizeForMatching(ctx.Query)
	bestMatch := MatchTypeNone

	// Check filename
	if ctx.Document.Title != "" {
		normalized := NormalizeFilename(ctx.Document.Title)
		queryNorm := NormalizeFilename(ctx.Query.Original)

		if normalized == queryNorm {
			return MatchTypeExact
		}

		for _, phrase := range ctx.Query.Phrases {
			if FindPhrasePosition(phrase, normalized) != -1 {
				bestMatch = MatchTypePhrase
				break
			}
		}

		if bestMatch == MatchTypeNone && len(tokens) > 0 {
			if AllTermsMatch(tokens, normalized) {
				if TermsInOrder(tokens, normalized) {
					bestMatch = MatchTypePhrase
				} else {
					bestMatch = MatchTypeAllWords
				}
			}
		}
	}

	// Check content if no strong filename match
	if bestMatch < MatchTypePhrase {
		content := ctx.Content
		if content == "" {
			content = ctx.Document.Content
		}
		if content != "" {
			for _, phrase := range ctx.Query.Phrases {
				if FindPhrasePosition(phrase, content) != -1 {
					if bestMatch < MatchTypePhrase {
						bestMatch = MatchTypePhrase
					}
					break
				}
			}

			if bestMatch < MatchTypeAllWords && len(tokens) > 0 {
				if AllTermsMatch(tokens, content) {
					if TermsInOrder(tokens, content) {
						bestMatch = MatchTypePhrase
					} else if bestMatch < MatchTypeAllWords {
						bestMatch = MatchTypeAllWords
					}
				} else if CountMatchingTerms(tokens, content) > 0 {
					if bestMatch < MatchTypePartial {
						bestMatch = MatchTypePartial
					}
				}
			}
		}
	}

	return bestMatch
}

// RankedResult holds a document with its computed score.
type RankedResult struct {
	Document  *models.Document
	Score     float64
	Breakdown *ScoreBreakdown
}

// RankDocuments ranks a list of documents by score.
func (r *Ranker) RankDocuments(query string, docs []*models.Document) []*RankedResult {
	analyzed := r.AnalyzeQuery(query)
	return r.RankDocumentsWithQuery(analyzed, docs)
}

// RankDocumentsWithQuery ranks documents using a pre-analyzed query.
func (r *Ranker) RankDocumentsWithQuery(query *AnalyzedQuery, docs []*models.Document) []*RankedResult {
	results := make([]*RankedResult, 0, len(docs))

	for _, doc := range docs {
		score := r.Rank(query, doc)
		if score > 0 {
			results = append(results, &RankedResult{
				Document: doc,
				Score:    score,
			})
		}
	}

	// Sort by score descending
	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	return results
}

// RankDocumentsWithBreakdown ranks documents and includes score breakdowns.
func (r *Ranker) RankDocumentsWithBreakdown(query string, docs []*models.Document) []*RankedResult {
	analyzed := r.AnalyzeQuery(query)
	results := make([]*RankedResult, 0, len(docs))

	for _, doc := range docs {
		breakdown := r.RankWithBreakdown(analyzed, doc)
		if breakdown.FinalScore > 0 {
			results = append(results, &RankedResult{
				Document:  doc,
				Score:     breakdown.FinalScore,
				Breakdown: breakdown,
			})
		}
	}

	// Sort by score descending
	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	return results
}

// ReRank re-ranks existing search results using the ranking system.
func (r *Ranker) ReRank(query string, results []*models.SearchResult) []*models.SearchResult {
	analyzed := r.AnalyzeQuery(query)

	for _, result := range results {
		if result.Document != nil {
			result.Score = r.Rank(analyzed, result.Document)
		}
	}

	// Sort by new scores
	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	// Update ranks
	for i := range results {
		results[i].Rank = i + 1
	}

	return results
}

// UpdateCorpusStats updates corpus statistics from a list of documents.
func (r *Ranker) UpdateCorpusStats(docs []*models.Document) {
	r.corpusStats.TotalDocs = len(docs)

	// Count document frequencies for each term
	termDocs := make(map[string]map[string]bool) // term -> set of doc IDs

	for _, doc := range docs {
		docID := doc.ID
		content := doc.Content + " " + doc.Title

		// Tokenize content
		tokens := tokenizeForStats(content)
		for _, token := range tokens {
			if termDocs[token] == nil {
				termDocs[token] = make(map[string]bool)
			}
			termDocs[token][docID] = true
		}
	}

	// Convert to document frequencies
	r.corpusStats.DocFrequencies = make(map[string]int)
	for term, docSet := range termDocs {
		r.corpusStats.DocFrequencies[term] = len(docSet)
	}
}

// tokenizeForStats tokenizes content for corpus statistics.
func tokenizeForStats(content string) []string {
	analyzer := NewQueryAnalyzer()
	// Simple tokenization - split by whitespace and normalize
	query := &AnalyzedQuery{
		Original: content,
		Terms:    nil,
	}
	// Use the tokenizer to get normalized terms
	query = analyzer.Analyze(content)
	return query.Terms
}

// GetConfig returns the ranking configuration.
func (r *Ranker) GetConfig() *RankingConfig {
	return r.config
}

// GetCorpusStats returns the current corpus statistics.
func (r *Ranker) GetCorpusStats() *CorpusStats {
	return r.corpusStats
}

// FilterByMinScore filters results below a minimum score.
func FilterByMinScore(results []*RankedResult, minScore float64) []*RankedResult {
	filtered := make([]*RankedResult, 0, len(results))
	for _, r := range results {
		if r.Score >= minScore {
			filtered = append(filtered, r)
		}
	}
	return filtered
}

// TopN returns the top N results.
func TopN(results []*RankedResult, n int) []*RankedResult {
	if n >= len(results) {
		return results
	}
	return results[:n]
}

// Paginate returns a page of results.
func Paginate(results []*RankedResult, offset, limit int) []*RankedResult {
	if offset >= len(results) {
		return nil
	}
	end := offset + limit
	if end > len(results) {
		end = len(results)
	}
	return results[offset:end]
}
