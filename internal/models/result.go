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
type SearchResponse struct {
	Results   []*SearchResult `json:"results"`
	Total     int             `json:"total"`
	QueryTime int64           `json:"query_time_ms"`
	Query     string          `json:"query"`
}
