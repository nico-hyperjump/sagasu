package ranking

import (
	"regexp"
	"strings"
	"unicode"
)

// QueryAnalyzer analyzes search queries to extract terms, phrases, and metadata.
type QueryAnalyzer struct{}

// NewQueryAnalyzer creates a new QueryAnalyzer.
func NewQueryAnalyzer() *QueryAnalyzer {
	return &QueryAnalyzer{}
}

// Analyze parses a query string and returns an AnalyzedQuery.
func (qa *QueryAnalyzer) Analyze(query string) *AnalyzedQuery {
	result := &AnalyzedQuery{
		Original:     query,
		Terms:        []string{},
		Phrases:      []string{},
		NegatedTerms: []string{},
	}

	// Check for wildcards
	result.HasWildcard = strings.ContainsAny(query, "*?")

	// Extract quoted phrases
	remaining := qa.extractPhrases(query, result)

	// Extract negated terms and regular terms
	qa.extractTerms(remaining, result)

	// Classify query type
	result.QueryType = qa.classifyQuery(result)

	return result
}

// extractPhrases extracts quoted phrases from the query.
// Returns the query with phrases removed.
func (qa *QueryAnalyzer) extractPhrases(query string, result *AnalyzedQuery) string {
	// Match both single and double quoted phrases
	phraseRegex := regexp.MustCompile(`["']([^"']+)["']`)
	matches := phraseRegex.FindAllStringSubmatch(query, -1)

	for _, match := range matches {
		if len(match) > 1 {
			phrase := strings.TrimSpace(match[1])
			if phrase != "" {
				result.Phrases = append(result.Phrases, strings.ToLower(phrase))
			}
		}
	}

	// Remove phrases from query
	return phraseRegex.ReplaceAllString(query, " ")
}

// extractTerms extracts individual terms from the remaining query.
func (qa *QueryAnalyzer) extractTerms(query string, result *AnalyzedQuery) {
	// Split by whitespace and filter
	words := strings.Fields(query)

	for _, word := range words {
		word = strings.TrimSpace(word)
		if word == "" {
			continue
		}

		// Check for negation (NOT or -)
		if strings.HasPrefix(word, "-") || strings.EqualFold(word, "NOT") {
			if strings.HasPrefix(word, "-") {
				negated := strings.TrimPrefix(word, "-")
				negated = qa.normalizeToken(negated)
				if negated != "" {
					result.NegatedTerms = append(result.NegatedTerms, negated)
				}
			}
			continue
		}

		// Skip boolean operators
		if strings.EqualFold(word, "AND") || strings.EqualFold(word, "OR") {
			continue
		}

		// Normalize and add term
		normalized := qa.normalizeToken(word)
		if normalized != "" {
			result.Terms = append(result.Terms, normalized)
		}
	}
}

// normalizeToken normalizes a single token (lowercase, remove punctuation from edges).
func (qa *QueryAnalyzer) normalizeToken(token string) string {
	// Convert to lowercase
	token = strings.ToLower(token)

	// Remove leading/trailing punctuation (but keep internal punctuation like hyphens)
	token = strings.TrimFunc(token, func(r rune) bool {
		return unicode.IsPunct(r) && r != '-' && r != '_'
	})

	return token
}

// classifyQuery determines the query type based on its components.
func (qa *QueryAnalyzer) classifyQuery(result *AnalyzedQuery) QueryType {
	// Check for wildcards first
	if result.HasWildcard {
		return QueryTypeWildcard
	}

	// Check for boolean operators (negation)
	if len(result.NegatedTerms) > 0 {
		return QueryTypeBoolean
	}

	// Check for quoted phrases
	if len(result.Phrases) > 0 {
		return QueryTypePhrase
	}

	// Check term count
	if len(result.Terms) == 0 {
		return QueryTypeSingleWord
	}
	if len(result.Terms) == 1 {
		return QueryTypeSingleWord
	}

	return QueryTypeMultiWord
}

// TokenizeForMatching returns all tokens suitable for matching.
// This combines terms and phrase words for comprehensive matching.
func (qa *QueryAnalyzer) TokenizeForMatching(analyzed *AnalyzedQuery) []string {
	seen := make(map[string]bool)
	tokens := make([]string, 0, len(analyzed.Terms)+len(analyzed.Phrases)*3)

	// Add all terms
	for _, term := range analyzed.Terms {
		if !seen[term] {
			tokens = append(tokens, term)
			seen[term] = true
		}
	}

	// Add words from phrases
	for _, phrase := range analyzed.Phrases {
		words := strings.Fields(phrase)
		for _, word := range words {
			normalized := qa.normalizeToken(word)
			if normalized != "" && !seen[normalized] {
				tokens = append(tokens, normalized)
				seen[normalized] = true
			}
		}
	}

	return tokens
}

// ContainsTerm checks if the analyzed query contains a specific term.
func (qa *QueryAnalyzer) ContainsTerm(analyzed *AnalyzedQuery, term string) bool {
	normalized := strings.ToLower(term)
	for _, t := range analyzed.Terms {
		if t == normalized {
			return true
		}
	}
	return false
}

// AllTermsMatch checks if all query terms are found in the given text.
func AllTermsMatch(terms []string, text string) bool {
	if len(terms) == 0 {
		return false
	}
	textLower := strings.ToLower(text)
	for _, term := range terms {
		if !strings.Contains(textLower, term) {
			return false
		}
	}
	return true
}

// CountMatchingTerms counts how many query terms are found in the text.
func CountMatchingTerms(terms []string, text string) int {
	if len(terms) == 0 {
		return 0
	}
	count := 0
	textLower := strings.ToLower(text)
	for _, term := range terms {
		if strings.Contains(textLower, term) {
			count++
		}
	}
	return count
}

// TermsInOrder checks if terms appear in order in the text.
func TermsInOrder(terms []string, text string) bool {
	if len(terms) == 0 {
		return false
	}
	textLower := strings.ToLower(text)
	lastPos := -1
	for _, term := range terms {
		pos := strings.Index(textLower[lastPos+1:], term)
		if pos == -1 {
			return false
		}
		lastPos = lastPos + 1 + pos
	}
	return true
}

// FindPhrasePosition finds the position of a phrase in text.
// Returns -1 if not found.
func FindPhrasePosition(phrase, text string) int {
	return strings.Index(strings.ToLower(text), strings.ToLower(phrase))
}

// CountOccurrences counts how many times a term appears in text.
func CountOccurrences(term, text string) int {
	return strings.Count(strings.ToLower(text), strings.ToLower(term))
}

// NormalizeFilename normalizes a filename for matching.
// Removes extension and replaces separators with spaces.
func NormalizeFilename(filename string) string {
	// Remove extension
	if idx := strings.LastIndex(filename, "."); idx > 0 {
		filename = filename[:idx]
	}

	// Replace common separators with spaces
	filename = strings.ReplaceAll(filename, "_", " ")
	filename = strings.ReplaceAll(filename, "-", " ")
	filename = strings.ReplaceAll(filename, ".", " ")

	// Lowercase and trim
	return strings.ToLower(strings.TrimSpace(filename))
}

// ExtractExtension extracts the file extension from a filename.
func ExtractExtension(filename string) string {
	if idx := strings.LastIndex(filename, "."); idx > 0 {
		return strings.ToLower(filename[idx+1:])
	}
	return ""
}

// IsPrefixMatch checks if the term is a prefix of any word in text.
func IsPrefixMatch(term, text string) bool {
	termLower := strings.ToLower(term)
	textLower := strings.ToLower(text)

	words := strings.Fields(textLower)
	for _, word := range words {
		if strings.HasPrefix(word, termLower) {
			return true
		}
	}
	return false
}

// FindAllPrefixMatches finds all words in text that start with the term.
func FindAllPrefixMatches(term, text string) []string {
	termLower := strings.ToLower(term)
	textLower := strings.ToLower(text)

	var matches []string
	words := strings.Fields(textLower)
	for _, word := range words {
		if strings.HasPrefix(word, termLower) {
			matches = append(matches, word)
		}
	}
	return matches
}
