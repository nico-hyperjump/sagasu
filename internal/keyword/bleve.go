// Package keyword provides Bleve implementation of KeywordIndex.
package keyword

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/blevesearch/bleve/v2"
	"github.com/blevesearch/bleve/v2/analysis/analyzer/standard"
	blevequery "github.com/blevesearch/bleve/v2/search/query"
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
// When opts.TitleBoost > 1, we run separate title and content queries and merge with additive scoring,
// term coverage bonus, and phrase proximity boost for smarter multi-term ranking.
// When opts.FuzzyEnabled is true, fuzzy matching is used for typo tolerance.
func (b *BleveIndex) Search(ctx context.Context, query string, limit int, opts *SearchOptions) ([]*KeywordResult, error) {
	titleBoost := 1.0
	phraseBoost := 1.0
	fuzzyEnabled := false
	fuzziness := 2 // default fuzziness level
	if opts != nil {
		if opts.TitleBoost > 0 {
			titleBoost = opts.TitleBoost
		}
		if opts.PhraseBoost > 0 {
			phraseBoost = opts.PhraseBoost
		}
		fuzzyEnabled = opts.FuzzyEnabled
		if opts.Fuzziness > 0 {
			fuzziness = opts.Fuzziness
		}
	}

	if titleBoost <= 1.0 && phraseBoost <= 1.0 {
		return b.searchSingle(ctx, query, limit, fuzzyEnabled, fuzziness)
	}
	return b.searchWithBoosts(ctx, query, limit, titleBoost, phraseBoost, fuzzyEnabled, fuzziness)
}

// searchSingle runs one MatchQuery over all fields (original behavior).
// When fuzzyEnabled is true, uses FuzzyQuery for each term with the specified fuzziness.
func (b *BleveIndex) searchSingle(ctx context.Context, query string, limit int, fuzzyEnabled bool, fuzziness int) ([]*KeywordResult, error) {
	var q blevequery.Query
	if fuzzyEnabled {
		q = b.buildFuzzyQuery(query, fuzziness, "")
	} else {
		q = bleve.NewMatchQuery(query)
	}
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

// searchWithBoosts runs smart multi-term search with:
// 1. Additive scoring: score = (titleScore * titleBoost) + contentScore
// 2. Term coverage bonus: documents matching more query terms get higher scores
// 3. Phrase proximity boost: documents with adjacent query terms get boosted
// When fuzzyEnabled is true, uses FuzzyQuery for typo tolerance.
func (b *BleveIndex) searchWithBoosts(ctx context.Context, query string, limit int, titleBoost, phraseBoost float64, fuzzyEnabled bool, fuzziness int) ([]*KeywordResult, error) {
	// Request enough from each so merged top "limit" is correct (same doc can appear in both).
	reqSize := limit * 2
	if reqSize < 50 {
		reqSize = 50
	}

	// Tokenize query into terms for term coverage calculation
	terms := tokenizeQuery(query)
	numTerms := len(terms)

	// Run title and content queries
	var titleQuery, contentQuery blevequery.Query
	if fuzzyEnabled {
		titleQuery = b.buildFuzzyQuery(query, fuzziness, "title")
		contentQuery = b.buildFuzzyQuery(query, fuzziness, "content")
	} else {
		tq := bleve.NewMatchQuery(query)
		tq.SetField("title")
		titleQuery = tq
		cq := bleve.NewMatchQuery(query)
		cq.SetField("content")
		contentQuery = cq
	}
	titleReq := bleve.NewSearchRequest(titleQuery)
	titleReq.Size = reqSize
	titleReq.Fields = []string{"*"}

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

	// Collect title and content scores separately for additive merge
	titleScores := make(map[string]float64)
	contentScores := make(map[string]float64)

	for _, hit := range titleResults.Hits {
		titleScores[hit.ID] = hit.Score * titleBoost
	}
	for _, hit := range contentResults.Hits {
		contentScores[hit.ID] = hit.Score
	}

	// Calculate term coverage: for multi-term queries, count how many terms each doc matches
	termCoverage := make(map[string]int) // docID -> number of matched terms
	if numTerms > 1 {
		termCoverage = b.calculateTermCoverage(terms, reqSize, fuzzyEnabled, fuzziness)
	}

	// Check for phrase matches if phraseBoost > 1 and query has multiple terms
	phraseMatches := make(map[string]bool)
	if phraseBoost > 1.0 && numTerms > 1 {
		phraseMatches = b.findPhraseMatches(query, reqSize)
	}

	// Merge scores: ADDITIVE (title + content) * termCoverageMultiplier * phraseMultiplier
	scores := make(map[string]float64)
	allDocIDs := make(map[string]struct{})
	for id := range titleScores {
		allDocIDs[id] = struct{}{}
	}
	for id := range contentScores {
		allDocIDs[id] = struct{}{}
	}

	for id := range allDocIDs {
		// Additive: title + content (both can contribute)
		baseScore := titleScores[id] + contentScores[id]

		// Term coverage multiplier: PENALIZE documents that don't match all terms
		// Formula: (matched/total)^2 - this heavily penalizes partial matches
		// - 2/2 terms: (1.0)^2 = 1.0 (no penalty)
		// - 1/2 terms: (0.5)^2 = 0.25 (75% penalty!)
		// - 1/3 terms: (0.33)^2 = 0.11 (89% penalty!)
		// This ensures documents matching ALL query terms rank higher than partial matches
		termCoverageMultiplier := 1.0
		if numTerms > 1 {
			matched := termCoverage[id]
			if matched == 0 {
				matched = 1 // at least matched once to be in results
			}
			coverage := float64(matched) / float64(numTerms)
			termCoverageMultiplier = coverage * coverage // squared penalty
		}

		// Phrase boost multiplier
		phraseMultiplier := 1.0
		if phraseMatches[id] {
			phraseMultiplier = phraseBoost
		}

		scores[id] = baseScore * termCoverageMultiplier * phraseMultiplier
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

// tokenizeQuery splits query into lowercase terms, filtering out empty strings.
func tokenizeQuery(query string) []string {
	words := strings.Fields(strings.ToLower(query))
	terms := make([]string, 0, len(words))
	for _, w := range words {
		w = strings.TrimSpace(w)
		if w != "" {
			terms = append(terms, w)
		}
	}
	return terms
}

// buildFuzzyQuery creates a disjunction of FuzzyQueries for each term in the query.
// If field is empty, searches all fields; otherwise restricts to the specified field.
func (b *BleveIndex) buildFuzzyQuery(queryStr string, fuzziness int, field string) blevequery.Query {
	terms := tokenizeQuery(queryStr)
	if len(terms) == 0 {
		// Fallback to match query for empty terms
		mq := bleve.NewMatchQuery(queryStr)
		if field != "" {
			mq.SetField(field)
		}
		return mq
	}

	if len(terms) == 1 {
		// Single term: use simple FuzzyQuery
		fq := bleve.NewFuzzyQuery(terms[0])
		fq.SetFuzziness(fuzziness)
		if field != "" {
			fq.SetField(field)
		}
		return fq
	}

	// Multiple terms: combine with BooleanQuery (should match)
	// This mimics MatchQuery behavior where any term can match
	queries := make([]blevequery.Query, 0, len(terms))
	for _, term := range terms {
		fq := bleve.NewFuzzyQuery(term)
		fq.SetFuzziness(fuzziness)
		if field != "" {
			fq.SetField(field)
		}
		queries = append(queries, fq)
	}

	// Use DisjunctionQuery - matches if any term matches (OR semantics)
	disjunction := bleve.NewDisjunctionQuery(queries...)
	return disjunction
}

// calculateTermCoverage counts how many unique query terms each document matches.
// When fuzzyEnabled is true, uses FuzzyQuery for each term.
func (b *BleveIndex) calculateTermCoverage(terms []string, reqSize int, fuzzyEnabled bool, fuzziness int) map[string]int {
	coverage := make(map[string]int)
	for _, term := range terms {
		// Run a match/fuzzy query for each individual term
		var q blevequery.Query
		if fuzzyEnabled {
			fq := bleve.NewFuzzyQuery(term)
			fq.SetFuzziness(fuzziness)
			q = fq
		} else {
			q = bleve.NewMatchQuery(term)
		}
		req := bleve.NewSearchRequest(q)
		req.Size = reqSize
		results, err := b.index.Search(req)
		if err != nil {
			continue
		}
		for _, hit := range results.Hits {
			coverage[hit.ID]++
		}
	}
	return coverage
}

// findPhraseMatches finds documents where the query appears as a phrase (adjacent terms).
func (b *BleveIndex) findPhraseMatches(query string, reqSize int) map[string]bool {
	matches := make(map[string]bool)

	// Use MatchPhraseQuery which is more flexible than PhraseQuery
	// It allows some slop (terms don't need to be immediately adjacent)
	phraseQuery := bleve.NewMatchPhraseQuery(query)
	phraseQuery.SetField("content")
	req := bleve.NewSearchRequest(phraseQuery)
	req.Size = reqSize
	results, err := b.index.Search(req)
	if err != nil {
		return matches
	}
	for _, hit := range results.Hits {
		matches[hit.ID] = true
	}

	// Also check title field for phrase matches
	titlePhraseQuery := bleve.NewMatchPhraseQuery(query)
	titlePhraseQuery.SetField("title")
	titleReq := bleve.NewSearchRequest(titlePhraseQuery)
	titleReq.Size = reqSize
	titleResults, err := b.index.Search(titleReq)
	if err != nil {
		return matches
	}
	for _, hit := range titleResults.Hits {
		matches[hit.ID] = true
	}

	return matches
}

// Delete removes a document from the index.
func (b *BleveIndex) Delete(ctx context.Context, id string) error {
	return b.index.Delete(id)
}

// Close closes the Bleve index.
func (b *BleveIndex) Close() error {
	return b.index.Close()
}

// DocCount returns the total number of documents in the index.
func (b *BleveIndex) DocCount() (uint64, error) {
	return b.index.DocCount()
}

// GetTermDocFrequency returns the number of documents containing the given term.
// This is useful for IDF (Inverse Document Frequency) calculation.
func (b *BleveIndex) GetTermDocFrequency(term string) (int, error) {
	// Search for the term and count unique documents
	q := bleve.NewMatchQuery(term)
	req := bleve.NewSearchRequest(q)
	req.Size = 10000 // Get all matching docs for accurate count
	results, err := b.index.Search(req)
	if err != nil {
		return 0, fmt.Errorf("failed to search for term frequency: %w", err)
	}
	return int(results.Total), nil
}

// GetCorpusStats returns corpus-level statistics for a set of terms.
// Returns total document count and document frequencies for each term.
func (b *BleveIndex) GetCorpusStats(terms []string) (totalDocs int, docFreqs map[string]int, err error) {
	// Get total document count
	count, err := b.DocCount()
	if err != nil {
		return 0, nil, fmt.Errorf("failed to get doc count: %w", err)
	}
	totalDocs = int(count)

	// Get document frequency for each term
	docFreqs = make(map[string]int, len(terms))
	for _, term := range terms {
		freq, err := b.GetTermDocFrequency(term)
		if err != nil {
			// Log error but continue with other terms
			docFreqs[term] = 0
			continue
		}
		docFreqs[term] = freq
	}

	return totalDocs, docFreqs, nil
}

// GetAllTerms returns all unique terms from the index dictionary.
// This is used for spell checking to build the term dictionary.
func (b *BleveIndex) GetAllTerms() ([]string, error) {
	terms := make([]string, 0)
	seen := make(map[string]struct{})

	// Get terms from content field
	contentDict, err := b.index.FieldDict("content")
	if err == nil {
		defer contentDict.Close()
		for {
			entry, err := contentDict.Next()
			if err != nil || entry == nil {
				break
			}
			if _, ok := seen[entry.Term]; !ok {
				terms = append(terms, entry.Term)
				seen[entry.Term] = struct{}{}
			}
		}
	}

	// Get terms from title field
	titleDict, err := b.index.FieldDict("title")
	if err == nil {
		defer titleDict.Close()
		for {
			entry, err := titleDict.Next()
			if err != nil || entry == nil {
				break
			}
			if _, ok := seen[entry.Term]; !ok {
				terms = append(terms, entry.Term)
				seen[entry.Term] = struct{}{}
			}
		}
	}

	return terms, nil
}

// ContainsTerm checks if a term exists in the index.
func (b *BleveIndex) ContainsTerm(term string) (bool, error) {
	freq, err := b.GetTermDocFrequency(term)
	if err != nil {
		return false, err
	}
	return freq > 0, nil
}

// GetTermFrequency returns the document frequency for a term.
// This is an alias for GetTermDocFrequency to satisfy the TermDictionary interface.
func (b *BleveIndex) GetTermFrequency(term string) (int, error) {
	return b.GetTermDocFrequency(term)
}
