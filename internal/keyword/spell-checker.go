// Package keyword provides keyword (BM25) search indexing and search.
package keyword

import (
	"sort"
	"strings"
	"sync"
)

// Suggestion represents a spelling suggestion with its score.
type Suggestion struct {
	Term      string  // The suggested term
	Distance  int     // Edit distance from the original term
	Frequency int     // Document frequency (popularity)
	Score     float64 // Combined score for ranking
}

// SpellCheckResult contains the result of spell checking a query.
type SpellCheckResult struct {
	OriginalQuery   string       // The original query
	CorrectedQuery  string       // The suggested corrected query
	Suggestions     []Suggestion // Suggestions for each misspelled term
	HasCorrections  bool         // True if any corrections were made
	MisspelledTerms []string     // Terms that were detected as misspelled
}

// SpellChecker provides spell checking and suggestion functionality.
type SpellChecker struct {
	dictionary  TermDictionary
	maxDistance int
	minFreq     int
	maxSuggestions int

	// Cached terms for faster lookup
	termsCache []string
	termSet    map[string]struct{}
	cacheMu    sync.RWMutex
	cacheValid bool
}

// SpellCheckerOption is a functional option for configuring SpellChecker.
type SpellCheckerOption func(*SpellChecker)

// WithMaxDistance sets the maximum edit distance for suggestions.
func WithMaxDistance(d int) SpellCheckerOption {
	return func(s *SpellChecker) {
		if d > 0 {
			s.maxDistance = d
		}
	}
}

// WithMinFrequency sets the minimum document frequency for suggestions.
// Terms with lower frequency are ignored (likely rare or noise).
func WithMinFrequency(f int) SpellCheckerOption {
	return func(s *SpellChecker) {
		if f >= 0 {
			s.minFreq = f
		}
	}
}

// WithMaxSuggestions sets the maximum number of suggestions to return per term.
func WithMaxSuggestions(n int) SpellCheckerOption {
	return func(s *SpellChecker) {
		if n > 0 {
			s.maxSuggestions = n
		}
	}
}

// NewSpellChecker creates a new SpellChecker with the given dictionary.
func NewSpellChecker(dict TermDictionary, opts ...SpellCheckerOption) *SpellChecker {
	s := &SpellChecker{
		dictionary:     dict,
		maxDistance:    2,
		minFreq:        1,
		maxSuggestions: 5,
		termSet:        make(map[string]struct{}),
	}

	for _, opt := range opts {
		opt(s)
	}

	return s
}

// RefreshCache updates the internal term cache from the dictionary.
// This should be called periodically if the index changes.
func (s *SpellChecker) RefreshCache() error {
	terms, err := s.dictionary.GetAllTerms()
	if err != nil {
		return err
	}

	s.cacheMu.Lock()
	defer s.cacheMu.Unlock()

	s.termsCache = terms
	s.termSet = make(map[string]struct{}, len(terms))
	for _, t := range terms {
		s.termSet[strings.ToLower(t)] = struct{}{}
	}
	s.cacheValid = true

	return nil
}

// Check checks a query for spelling errors and returns suggestions.
func (s *SpellChecker) Check(query string) (*SpellCheckResult, error) {
	// Ensure cache is valid
	if !s.cacheValid {
		if err := s.RefreshCache(); err != nil {
			return nil, err
		}
	}

	terms := tokenizeQuery(query)
	result := &SpellCheckResult{
		OriginalQuery:   query,
		Suggestions:     make([]Suggestion, 0),
		MisspelledTerms: make([]string, 0),
	}

	correctedTerms := make([]string, 0, len(terms))

	for _, term := range terms {
		termLower := strings.ToLower(term)

		// Check if term exists in dictionary
		s.cacheMu.RLock()
		_, exists := s.termSet[termLower]
		s.cacheMu.RUnlock()

		if exists {
			// Term is valid, keep it
			correctedTerms = append(correctedTerms, term)
			continue
		}

		// Term not found, get suggestions
		suggestions := s.Suggest(termLower)
		if len(suggestions) > 0 {
			result.HasCorrections = true
			result.MisspelledTerms = append(result.MisspelledTerms, term)
			result.Suggestions = append(result.Suggestions, suggestions...)
			// Use the best suggestion for the corrected query
			correctedTerms = append(correctedTerms, suggestions[0].Term)
		} else {
			// No suggestions found, keep original term
			correctedTerms = append(correctedTerms, term)
		}
	}

	result.CorrectedQuery = strings.Join(correctedTerms, " ")
	return result, nil
}

// Suggest returns spelling suggestions for a single term.
func (s *SpellChecker) Suggest(term string) []Suggestion {
	// Ensure cache is valid
	if !s.cacheValid {
		if err := s.RefreshCache(); err != nil {
			return nil
		}
	}

	termLower := strings.ToLower(term)
	suggestions := make([]Suggestion, 0)

	s.cacheMu.RLock()
	terms := s.termsCache
	s.cacheMu.RUnlock()

	// Find terms within edit distance
	for _, dictTerm := range terms {
		dictTermLower := strings.ToLower(dictTerm)

		// Skip if same term
		if dictTermLower == termLower {
			continue
		}

		// Quick length check - if length difference > maxDistance, can't be within distance
		lenDiff := len(dictTermLower) - len(termLower)
		if lenDiff < 0 {
			lenDiff = -lenDiff
		}
		if lenDiff > s.maxDistance {
			continue
		}

		distance := LevenshteinDistance(termLower, dictTermLower)
		if distance <= s.maxDistance {
			// Get frequency for ranking
			freq, err := s.dictionary.GetTermFrequency(dictTerm)
			if err != nil || freq < s.minFreq {
				continue
			}

			// Calculate score: lower distance is better, higher frequency is better
			// Score = 1 / (distance + 1) * log(frequency + 1)
			score := (1.0 / float64(distance+1)) * float64(freq)

			suggestions = append(suggestions, Suggestion{
				Term:      dictTerm,
				Distance:  distance,
				Frequency: freq,
				Score:     score,
			})
		}
	}

	// Sort by score (descending)
	sort.Slice(suggestions, func(i, j int) bool {
		return suggestions[i].Score > suggestions[j].Score
	})

	// Limit results
	if len(suggestions) > s.maxSuggestions {
		suggestions = suggestions[:s.maxSuggestions]
	}

	return suggestions
}

// IsMisspelled checks if a term is likely misspelled (not in dictionary).
func (s *SpellChecker) IsMisspelled(term string) bool {
	// Ensure cache is valid
	if !s.cacheValid {
		if err := s.RefreshCache(); err != nil {
			return false
		}
	}

	s.cacheMu.RLock()
	defer s.cacheMu.RUnlock()

	_, exists := s.termSet[strings.ToLower(term)]
	return !exists
}

// GetSuggestedQuery returns the best suggested query string for a misspelled query.
// Returns the original query if no corrections are found.
func (s *SpellChecker) GetSuggestedQuery(query string) string {
	result, err := s.Check(query)
	if err != nil || !result.HasCorrections {
		return query
	}
	return result.CorrectedQuery
}

// GetTopSuggestions returns the top N suggestions for a query.
// This flattens all suggestions from all misspelled terms.
func (s *SpellChecker) GetTopSuggestions(query string, n int) []string {
	result, err := s.Check(query)
	if err != nil {
		return nil
	}

	// Collect unique corrected queries
	suggestions := make([]string, 0, n)
	if result.HasCorrections && result.CorrectedQuery != result.OriginalQuery {
		suggestions = append(suggestions, result.CorrectedQuery)
	}

	// If we need more suggestions, try alternative combinations
	// For now, just return the single best suggestion
	if len(suggestions) > n {
		suggestions = suggestions[:n]
	}

	return suggestions
}
