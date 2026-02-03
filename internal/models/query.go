package models

import "fmt"

// SearchQuery represents a search request with optional weights and filters.
type SearchQuery struct {
	Query          string                 `json:"query"`
	Limit          int                    `json:"limit,omitempty"`
	Offset         int                    `json:"offset,omitempty"`
	KeywordWeight  float64                `json:"keyword_weight,omitempty"`
	SemanticWeight float64                `json:"semantic_weight,omitempty"`
	MinScore       float64                `json:"min_score,omitempty"`
	Filters        map[string]interface{} `json:"filters,omitempty"`
}

// Validate ensures the search query has valid fields and sets defaults.
// Returns an error if the query is empty; otherwise normalizes limit and weights.
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
	if q.KeywordWeight == 0 && q.SemanticWeight == 0 {
		q.KeywordWeight = 0.5
		q.SemanticWeight = 0.5
	}
	return nil
}
