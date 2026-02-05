package models

// SearchResult represents a single search hit with document and scores.
type SearchResult struct {
	Document      *Document         `json:"document"`
	Score         float64           `json:"score"`
	KeywordScore  float64           `json:"keyword_score"`
	SemanticScore float64           `json:"semantic_score"`
	Highlights    map[string]string `json:"highlights,omitempty"`
	Rank          int               `json:"rank"`
}

// SearchResponse is the response for a search request.
// NonSemanticResults and SemanticResults are disjoint (no document appears in both).
type SearchResponse struct {
	// NonSemanticResults are keyword-only hits (not in semantic set).
	NonSemanticResults []*SearchResult `json:"non_semantic_results"`
	// SemanticResults are semantic-only hits (not in keyword set).
	SemanticResults []*SearchResult `json:"semantic_results"`
	TotalNonSemantic int             `json:"total_non_semantic"`
	TotalSemantic    int             `json:"total_semantic"`
	QueryTime        int64           `json:"query_time_ms"`
	Query            string          `json:"query"`
	// Suggestions contains "Did you mean?" spelling suggestions when typos are detected.
	// Only populated when FuzzyEnabled is true and misspelled terms are found.
	Suggestions []string `json:"suggestions,omitempty"`
	// AutoFuzzy indicates that fuzzy search was automatically enabled because the
	// initial exact search returned no results. This helps the user understand
	// why results may include fuzzy matches.
	AutoFuzzy bool `json:"auto_fuzzy,omitempty"`
}
