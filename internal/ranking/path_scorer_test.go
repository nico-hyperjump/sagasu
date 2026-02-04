package ranking

import (
	"testing"

	"github.com/hyperjump/sagasu/internal/models"
)

func TestPathScorer_Score(t *testing.T) {
	config := DefaultRankingConfig()
	scorer := NewPathScorer(config)
	analyzer := NewQueryAnalyzer()

	tests := []struct {
		name    string
		query   string
		path    string
		wantMin float64
		wantMax float64
	}{
		{
			name:    "exact directory match",
			query:   "projects",
			path:    "/home/user/projects/file.txt",
			wantMin: config.PathExactMatchScore * 0.9,
			wantMax: config.PathExactMatchScore * 1.1,
		},
		{
			name:    "partial directory match",
			query:   "proj",
			path:    "/home/user/projects/file.txt",
			wantMin: config.PathPartialMatchScore * 0.8,
			wantMax: config.PathPartialMatchScore * 1.5,
		},
		{
			name:    "multiple components match",
			query:   "home projects",
			path:    "/home/user/projects/file.txt",
			wantMin: config.PathExactMatchScore,
			wantMax: config.PathExactMatchScore*2 + config.PathComponentBonus*2,
		},
		{
			name:    "no match",
			query:   "xyz",
			path:    "/home/user/projects/file.txt",
			wantMin: 0,
			wantMax: 0.1,
		},
		{
			name:    "case insensitive",
			query:   "PROJECTS",
			path:    "/home/user/projects/file.txt",
			wantMin: config.PathExactMatchScore * 0.9,
			wantMax: config.PathExactMatchScore * 1.1,
		},
		{
			name:    "nested path",
			query:   "2024",
			path:    "/home/user/documents/reports/2024/quarterly/file.txt",
			wantMin: config.PathExactMatchScore * 0.9,
			wantMax: config.PathExactMatchScore * 1.1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			analyzed := analyzer.Analyze(tt.query)
			ctx := &ScoringContext{
				Query:    analyzed,
				FilePath: tt.path,
				Document: &models.Document{
					Metadata: map[string]interface{}{
						"source_path": tt.path,
					},
				},
			}

			score := scorer.Score(ctx)

			if score < tt.wantMin || score > tt.wantMax {
				t.Errorf("Score() = %v, want between %v and %v", score, tt.wantMin, tt.wantMax)
			}
		})
	}
}

func TestPathScorer_NilInputs(t *testing.T) {
	config := DefaultRankingConfig()
	scorer := NewPathScorer(config)

	// Nil query
	score := scorer.Score(&ScoringContext{
		Query:    nil,
		FilePath: "/home/user/file.txt",
	})
	if score != 0 {
		t.Errorf("Expected 0 for nil query, got %v", score)
	}

	// Empty path
	score = scorer.Score(&ScoringContext{
		Query:    &AnalyzedQuery{Terms: []string{"test"}},
		FilePath: "",
		Document: &models.Document{},
	})
	if score != 0 {
		t.Errorf("Expected 0 for empty path, got %v", score)
	}

	// Empty terms
	score = scorer.Score(&ScoringContext{
		Query:    &AnalyzedQuery{Terms: []string{}},
		FilePath: "/home/user/file.txt",
	})
	if score != 0 {
		t.Errorf("Expected 0 for empty terms, got %v", score)
	}
}

func TestPathScorer_Name(t *testing.T) {
	config := DefaultRankingConfig()
	scorer := NewPathScorer(config)

	if scorer.Name() != "path" {
		t.Errorf("Name() = %v, want 'path'", scorer.Name())
	}
}

func TestExtractPathComponents(t *testing.T) {
	tests := []struct {
		name string
		path string
		want []string
	}{
		{
			name: "unix path",
			path: "/home/user/projects/file.txt",
			want: []string{"home", "user", "projects"},
		},
		{
			name: "deep path",
			path: "/a/b/c/d/e/file.txt",
			want: []string{"a", "b", "c", "d", "e"},
		},
		{
			name: "root file",
			path: "/file.txt",
			want: []string{},
		},
		{
			name: "relative path",
			path: "projects/file.txt",
			want: []string{"projects"},
		},
		{
			name: "current directory",
			path: "./file.txt",
			want: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractPathComponents(tt.path)
			if len(got) != len(tt.want) {
				t.Errorf("ExtractPathComponents() = %v (len %d), want %v (len %d)", got, len(got), tt.want, len(tt.want))
				return
			}
			for i, component := range got {
				if component != tt.want[i] {
					t.Errorf("ExtractPathComponents()[%d] = %v, want %v", i, component, tt.want[i])
				}
			}
		})
	}
}

func TestPathScorer_ScoreWithDetails(t *testing.T) {
	config := DefaultRankingConfig()
	scorer := NewPathScorer(config)
	analyzer := NewQueryAnalyzer()

	analyzed := analyzer.Analyze("projects reports")
	ctx := &ScoringContext{
		Query:    analyzed,
		FilePath: "/home/user/projects/reports/file.txt",
		Document: &models.Document{
			Metadata: map[string]interface{}{
				"source_path": "/home/user/projects/reports/file.txt",
			},
		},
	}

	result := scorer.ScoreWithDetails(ctx)

	if result.Score <= 0 {
		t.Errorf("Expected positive score, got %v", result.Score)
	}
	if len(result.MatchedComponents) == 0 {
		t.Error("Expected matched components to be populated")
	}
}

func TestNormalizePath(t *testing.T) {
	tests := []struct {
		name string
		path string
		want string
	}{
		{
			name: "lowercase",
			path: "/Home/User/Projects/",
			want: "/home/user/projects",
		},
		{
			name: "remove trailing slash",
			path: "/home/user/",
			want: "/home/user",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NormalizePath(tt.path)
			if got != tt.want {
				t.Errorf("NormalizePath() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPathContainsComponent(t *testing.T) {
	tests := []struct {
		name      string
		path      string
		component string
		want      bool
	}{
		{
			name:      "component present",
			path:      "/home/user/projects/file.txt",
			component: "projects",
			want:      true,
		},
		{
			name:      "component absent",
			path:      "/home/user/projects/file.txt",
			component: "missing",
			want:      false,
		},
		{
			name:      "case insensitive",
			path:      "/home/user/Projects/file.txt",
			component: "projects",
			want:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := PathContainsComponent(tt.path, tt.component)
			if got != tt.want {
				t.Errorf("PathContainsComponent() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPathMatchesPattern(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		pattern string
		want    bool
	}{
		{
			name:    "exact match",
			path:    "/home/user/projects/file.txt",
			pattern: "projects",
			want:    true,
		},
		{
			name:    "wildcard match",
			path:    "/home/user/projects/file.txt",
			pattern: "proj*",
			want:    true,
		},
		{
			name:    "no match",
			path:    "/home/user/projects/file.txt",
			pattern: "xyz",
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := PathMatchesPattern(tt.path, tt.pattern)
			if got != tt.want {
				t.Errorf("PathMatchesPattern() = %v, want %v", got, tt.want)
			}
		})
	}
}
