package ranking

import (
	"testing"

	"github.com/hyperjump/sagasu/internal/models"
)

func TestFilenameScorer_Score(t *testing.T) {
	config := DefaultRankingConfig()
	scorer := NewFilenameScorer(config)
	analyzer := NewQueryAnalyzer()

	tests := []struct {
		name      string
		query     string
		filename  string
		wantMin   float64
		wantMax   float64
	}{
		{
			name:     "exact filename match",
			query:    "budget",
			filename: "budget.txt",
			wantMin:  config.ExactFilenameScore * 0.9,
			wantMax:  config.ExactFilenameScore * 1.1,
		},
		{
			name:     "exact multi-word filename match",
			query:    "annual budget report",
			filename: "annual_budget_report.pdf",
			wantMin:  config.AllWordsInOrderScore * 0.9,
			wantMax:  config.AllWordsInOrderScore * 1.2,
		},
		{
			name:     "all words present any order",
			query:    "report budget annual",
			filename: "annual_budget_report.pdf",
			wantMin:  config.AllWordsAnyOrderScore * 0.9,
			wantMax:  config.AllWordsAnyOrderScore * 1.2,
		},
		{
			name:     "partial match - some words",
			query:    "budget analysis",
			filename: "budget_report.pdf",
			wantMin:  config.SubstringMatchScore * 0.3,
			wantMax:  config.SubstringMatchScore * 1.5,
		},
		{
			name:     "substring match",
			query:    "budget",
			filename: "2024_budget_final.xlsx",
			wantMin:  config.SubstringMatchScore * 0.8,
			wantMax:  config.SubstringMatchScore * 1.5,
		},
		{
			name:     "prefix match",
			query:    "budg",
			filename: "budget.txt",
			wantMin:  config.PrefixMatchScore * 0.8,
			wantMax:  config.AllWordsInOrderScore * 1.2, // Can match as all words too
		},
		{
			name:     "extension match",
			query:    "pdf",
			filename: "report.pdf",
			wantMin:  config.ExtensionMatchScore * 0.9,
			wantMax:  config.ExtensionMatchScore * 1.1,
		},
		{
			name:     "no match",
			query:    "xyz",
			filename: "budget.txt",
			wantMin:  0,
			wantMax:  0.1,
		},
		{
			name:     "case insensitive",
			query:    "BUDGET",
			filename: "budget.txt",
			wantMin:  config.ExactFilenameScore * 0.9,
			wantMax:  config.ExactFilenameScore * 1.1,
		},
		{
			name:     "phrase in filename",
			query:    `"budget report"`,
			filename: "budget_report_2024.pdf",
			wantMin:  config.AllWordsInOrderScore * 0.9,
			wantMax:  config.AllWordsInOrderScore * 1.2,
		},
		{
			name:     "multiple occurrences",
			query:    "report",
			filename: "report_annual_report.txt",
			wantMin:  config.SubstringMatchScore,
			wantMax:  config.ExactFilenameScore + config.MultipleOccurrenceBonus*2, // Can match as exact start
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			analyzed := analyzer.Analyze(tt.query)
			ctx := &ScoringContext{
				Query: analyzed,
				Document: &models.Document{
					Title: tt.filename,
				},
			}

			score := scorer.Score(ctx)

			if score < tt.wantMin || score > tt.wantMax {
				t.Errorf("Score() = %v, want between %v and %v", score, tt.wantMin, tt.wantMax)
			}
		})
	}
}

func TestFilenameScorer_NilInputs(t *testing.T) {
	config := DefaultRankingConfig()
	scorer := NewFilenameScorer(config)

	// Nil document
	score := scorer.Score(&ScoringContext{
		Query:    &AnalyzedQuery{Terms: []string{"test"}},
		Document: nil,
	})
	if score != 0 {
		t.Errorf("Expected 0 for nil document, got %v", score)
	}

	// Nil query
	score = scorer.Score(&ScoringContext{
		Query:    nil,
		Document: &models.Document{Title: "test.txt"},
	})
	if score != 0 {
		t.Errorf("Expected 0 for nil query, got %v", score)
	}

	// Empty filename
	score = scorer.Score(&ScoringContext{
		Query:    &AnalyzedQuery{Terms: []string{"test"}},
		Document: &models.Document{Title: ""},
	})
	if score != 0 {
		t.Errorf("Expected 0 for empty filename, got %v", score)
	}

	// Empty terms
	score = scorer.Score(&ScoringContext{
		Query:    &AnalyzedQuery{Terms: []string{}},
		Document: &models.Document{Title: "test.txt"},
	})
	if score != 0 {
		t.Errorf("Expected 0 for empty terms, got %v", score)
	}
}

func TestFilenameScorer_Name(t *testing.T) {
	config := DefaultRankingConfig()
	scorer := NewFilenameScorer(config)

	if scorer.Name() != "filename" {
		t.Errorf("Name() = %v, want 'filename'", scorer.Name())
	}
}

func TestFilenameScorer_ScoreWithDetails(t *testing.T) {
	config := DefaultRankingConfig()
	scorer := NewFilenameScorer(config)
	analyzer := NewQueryAnalyzer()

	tests := []struct {
		name          string
		query         string
		filename      string
		wantMatchType MatchType
	}{
		{
			name:          "exact match",
			query:         "budget",
			filename:      "budget.txt",
			wantMatchType: MatchTypeExact,
		},
		{
			name:          "phrase match",
			query:         "annual budget report",
			filename:      "annual_budget_report.pdf",
			wantMatchType: MatchTypeExact, // Normalized query matches normalized filename exactly
		},
		{
			name:          "all words any order",
			query:         "report annual",
			filename:      "annual_budget_report.pdf",
			wantMatchType: MatchTypeAllWords,
		},
		{
			name:          "partial match",
			query:         "budget analysis",
			filename:      "budget_report.pdf",
			wantMatchType: MatchTypePartial,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			analyzed := analyzer.Analyze(tt.query)
			ctx := &ScoringContext{
				Query: analyzed,
				Document: &models.Document{
					Title: tt.filename,
				},
			}

			result := scorer.ScoreWithDetails(ctx)

			if result.MatchType != tt.wantMatchType {
				t.Errorf("MatchType = %v, want %v", result.MatchType, tt.wantMatchType)
			}
		})
	}
}

func TestScoreFilenameOnly(t *testing.T) {
	config := DefaultRankingConfig()

	score := ScoreFilenameOnly(config, "budget", "budget.txt")
	if score <= 0 {
		t.Errorf("Expected positive score for matching filename, got %v", score)
	}

	score = ScoreFilenameOnly(config, "xyz", "budget.txt")
	if score != 0 {
		t.Errorf("Expected 0 for non-matching filename, got %v", score)
	}
}
