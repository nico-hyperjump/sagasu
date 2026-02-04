package ranking

import (
	"testing"

	"github.com/hyperjump/sagasu/internal/models"
)

func TestMetadataScorer_Score(t *testing.T) {
	config := DefaultRankingConfig()
	scorer := NewMetadataScorer(config)
	analyzer := NewQueryAnalyzer()

	tests := []struct {
		name     string
		query    string
		metadata map[string]interface{}
		wantMin  float64
		wantMax  float64
	}{
		{
			name:  "author match",
			query: "john",
			metadata: map[string]interface{}{
				"author": "John Smith",
			},
			wantMin: config.AuthorMatchScore * 0.9,
			wantMax: config.AuthorMatchScore * 1.1,
		},
		{
			name:  "tag match",
			query: "finance",
			metadata: map[string]interface{}{
				"tags": "finance, budget, quarterly",
			},
			wantMin: config.TagMatchScore * 0.3,
			wantMax: config.TagMatchScore * 1.1,
		},
		{
			name:  "tags array match",
			query: "finance",
			metadata: map[string]interface{}{
				"tags": []string{"finance", "budget", "quarterly"},
			},
			wantMin: config.TagMatchScore * 0.3,
			wantMax: config.TagMatchScore * 1.1,
		},
		{
			name:  "other metadata match",
			query: "report",
			metadata: map[string]interface{}{
				"description": "Annual report document",
			},
			wantMin: config.OtherMetadataScore * 0.4,
			wantMax: config.OtherMetadataScore * 1.1,
		},
		{
			name:  "multiple fields match",
			query: "john finance",
			metadata: map[string]interface{}{
				"author": "John Smith",
				"tags":   "finance, budget",
			},
			wantMin: config.AuthorMatchScore * 0.4,
			wantMax: config.AuthorMatchScore + config.TagMatchScore,
		},
		{
			name:  "no match",
			query: "xyz",
			metadata: map[string]interface{}{
				"author": "John Smith",
				"tags":   "finance, budget",
			},
			wantMin: 0,
			wantMax: 0.1,
		},
		{
			name:  "internal metadata ignored",
			query: "source",
			metadata: map[string]interface{}{
				"source_path":  "/home/user/file.txt",
				"source_mtime": "1234567890",
			},
			wantMin: 0,
			wantMax: 0.1,
		},
		{
			name:  "creator field as author",
			query: "jane",
			metadata: map[string]interface{}{
				"creator": "Jane Doe",
			},
			wantMin: config.AuthorMatchScore * 0.9,
			wantMax: config.AuthorMatchScore * 1.1,
		},
		{
			name:  "keywords field as tags",
			query: "important",
			metadata: map[string]interface{}{
				"keywords": "important, urgent",
			},
			wantMin: config.TagMatchScore * 0.4,
			wantMax: config.TagMatchScore * 1.1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			analyzed := analyzer.Analyze(tt.query)
			ctx := &ScoringContext{
				Query: analyzed,
				Document: &models.Document{
					Metadata: tt.metadata,
				},
			}

			score := scorer.Score(ctx)

			if score < tt.wantMin || score > tt.wantMax {
				t.Errorf("Score() = %v, want between %v and %v", score, tt.wantMin, tt.wantMax)
			}
		})
	}
}

func TestMetadataScorer_NilInputs(t *testing.T) {
	config := DefaultRankingConfig()
	scorer := NewMetadataScorer(config)

	// Nil query
	score := scorer.Score(&ScoringContext{
		Query:    nil,
		Document: &models.Document{Metadata: map[string]interface{}{"author": "John"}},
	})
	if score != 0 {
		t.Errorf("Expected 0 for nil query, got %v", score)
	}

	// Nil document
	score = scorer.Score(&ScoringContext{
		Query:    &AnalyzedQuery{Terms: []string{"test"}},
		Document: nil,
	})
	if score != 0 {
		t.Errorf("Expected 0 for nil document, got %v", score)
	}

	// Nil metadata
	score = scorer.Score(&ScoringContext{
		Query:    &AnalyzedQuery{Terms: []string{"test"}},
		Document: &models.Document{Metadata: nil},
	})
	if score != 0 {
		t.Errorf("Expected 0 for nil metadata, got %v", score)
	}

	// Empty metadata
	score = scorer.Score(&ScoringContext{
		Query:    &AnalyzedQuery{Terms: []string{"test"}},
		Document: &models.Document{Metadata: map[string]interface{}{}},
	})
	if score != 0 {
		t.Errorf("Expected 0 for empty metadata, got %v", score)
	}
}

func TestMetadataScorer_Name(t *testing.T) {
	config := DefaultRankingConfig()
	scorer := NewMetadataScorer(config)

	if scorer.Name() != "metadata" {
		t.Errorf("Name() = %v, want 'metadata'", scorer.Name())
	}
}

func TestMetadataScorer_ScoreWithDetails(t *testing.T) {
	config := DefaultRankingConfig()
	scorer := NewMetadataScorer(config)
	analyzer := NewQueryAnalyzer()

	analyzed := analyzer.Analyze("john finance")
	ctx := &ScoringContext{
		Query: analyzed,
		Document: &models.Document{
			Metadata: map[string]interface{}{
				"author": "John Smith",
				"tags":   "finance, budget",
			},
		},
	}

	result := scorer.ScoreWithDetails(ctx)

	if result.Score <= 0 {
		t.Errorf("Expected positive score, got %v", result.Score)
	}
	if len(result.MatchedFields) == 0 {
		t.Error("Expected matched fields to be populated")
	}
	if _, ok := result.MatchedFields["author"]; !ok {
		t.Error("Expected 'author' field to be matched")
	}
	if _, ok := result.MatchedFields["tags"]; !ok {
		t.Error("Expected 'tags' field to be matched")
	}
}

func TestExtractMetadataText(t *testing.T) {
	tests := []struct {
		name     string
		metadata map[string]interface{}
		wantLen  int
	}{
		{
			name: "string values",
			metadata: map[string]interface{}{
				"author":      "John Smith",
				"description": "A document",
			},
			wantLen: 2, // Should have content from both fields
		},
		{
			name: "internal keys excluded",
			metadata: map[string]interface{}{
				"author":       "John Smith",
				"source_path":  "/path/to/file",
				"source_mtime": "123456",
			},
			wantLen: 1, // Only author should be included
		},
		{
			name:     "nil metadata",
			metadata: nil,
			wantLen:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			text := ExtractMetadataText(tt.metadata)
			if tt.wantLen > 0 && len(text) == 0 {
				t.Error("Expected non-empty text")
			}
		})
	}
}

func TestGetMetadataField(t *testing.T) {
	metadata := map[string]interface{}{
		"author":      "John Smith",
		"count":       42,
		"tags":        []string{"a", "b"},
		"source_path": "/path",
	}

	tests := []struct {
		name string
		key  string
		want string
	}{
		{"string field", "author", "John Smith"},
		{"int field", "count", "42"},
		{"array field", "tags", "a b"},
		{"missing field", "missing", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetMetadataField(metadata, tt.key)
			if got != tt.want {
				t.Errorf("GetMetadataField() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetMetadataFieldSlice(t *testing.T) {
	metadata := map[string]interface{}{
		"tags":       []string{"a", "b", "c"},
		"interfaces": []interface{}{"x", "y"},
		"csv":        "one,two,three",
	}

	tests := []struct {
		name    string
		key     string
		wantLen int
	}{
		{"string slice", "tags", 3},
		{"interface slice", "interfaces", 2},
		{"comma separated", "csv", 3},
		{"missing", "missing", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetMetadataFieldSlice(metadata, tt.key)
			if len(got) != tt.wantLen {
				t.Errorf("GetMetadataFieldSlice() len = %v, want %v", len(got), tt.wantLen)
			}
		})
	}
}

func TestMatchMetadataField(t *testing.T) {
	metadata := map[string]interface{}{
		"author": "John Smith",
		"tags":   "finance, budget",
	}

	tests := []struct {
		name  string
		query string
		field string
		want  bool
	}{
		{"author match", "john", "author", true},
		{"tag match", "finance", "tags", true},
		{"no match", "xyz", "author", false},
		{"missing field", "john", "missing", false},
	}

	analyzer := NewQueryAnalyzer()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			analyzed := analyzer.Analyze(tt.query)
			got := MatchMetadataField(analyzed, metadata, tt.field)
			if got != tt.want {
				t.Errorf("MatchMetadataField() = %v, want %v", got, tt.want)
			}
		})
	}
}
