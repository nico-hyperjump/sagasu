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
		{"enables both when both false", &SearchQuery{Query: "x", KeywordEnabled: false, SemanticEnabled: false}, false},
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
				if tt.name == "enables both when both false" && (!tt.query.KeywordEnabled || !tt.query.SemanticEnabled) {
					t.Error("expected both keyword and semantic enabled when both were false")
				}
			}
		})
	}
}

func TestSearchQuery_FuzzyEnabled(t *testing.T) {
	tests := []struct {
		name         string
		query        *SearchQuery
		wantFuzzy    bool
	}{
		{
			name:      "fuzzy disabled by default",
			query:     &SearchQuery{Query: "test"},
			wantFuzzy: false,
		},
		{
			name:      "fuzzy enabled explicitly",
			query:     &SearchQuery{Query: "test", FuzzyEnabled: true},
			wantFuzzy: true,
		},
		{
			name:      "fuzzy with keyword search",
			query:     &SearchQuery{Query: "test", KeywordEnabled: true, FuzzyEnabled: true},
			wantFuzzy: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.query.Validate()
			if err != nil {
				t.Fatalf("Validate() unexpected error: %v", err)
			}
			if tt.query.FuzzyEnabled != tt.wantFuzzy {
				t.Errorf("FuzzyEnabled = %v, want %v", tt.query.FuzzyEnabled, tt.wantFuzzy)
			}
		})
	}
}
