package models

import (
	"testing"
)

func TestSearchQuery_Validate(t *testing.T) {
	tests := []struct {
		name    string
		query   *SearchQuery
		wantErr bool
	}{
		{"empty query", &SearchQuery{Query: ""}, true},
		{"valid query", &SearchQuery{Query: "hello"}, false},
		{"sets default limit", &SearchQuery{Query: "x", Limit: 0}, false},
		{"caps limit at 100", &SearchQuery{Query: "x", Limit: 200}, false},
		{"sets default weights", &SearchQuery{Query: "x", KeywordWeight: 0, SemanticWeight: 0}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.query.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && tt.query.Query != "" {
				if tt.query.Limit == 0 {
					t.Error("expected default limit to be set")
				}
				if tt.query.Limit > 100 {
					t.Errorf("expected limit capped at 100, got %d", tt.query.Limit)
				}
			}
		})
	}
}
