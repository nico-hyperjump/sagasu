package models

import "fmt"

// SearchQuery represents a search request with optional filters.
type SearchQuery struct {
	Query           string                 `json:"query"`
	Limit           int                    `json:"limit,omitempty"`
	Offset          int                    `json:"offset,omitempty"`
	KeywordEnabled  bool                   `json:"keyword_enabled,omitempty"`
	SemanticEnabled bool                   `json:"semantic_enabled,omitempty"`
	MinScore        float64                `json:"min_score,omitempty"`
	Filters         map[string]interface{} `json:"filters,omitempty"`
}

// Validate ensures the search query has valid fields and sets defaults.
// Returns an error if the query is empty; otherwise normalizes limit and enables at least one search type.
func (q *SearchQuery) Validate() error {
	if q.Query == "" {
		return fmt.Errorf("query cannot be empty")
	}
	if q.Limit <= 0 {
		q.Limit = 10
	}
	if q.Limit > 100 {
		q.Limit = 100
	}
	if !q.KeywordEnabled && !q.SemanticEnabled {
		q.KeywordEnabled = true
		q.SemanticEnabled = true
	}
	return nil
}
