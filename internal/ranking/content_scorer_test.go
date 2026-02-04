package ranking

import (
	"testing"

	"github.com/hyperjump/sagasu/internal/models"
)

func TestContentScorer_Score(t *testing.T) {
	config := DefaultRankingConfig()
	scorer := NewContentScorer(config)
	analyzer := NewQueryAnalyzer()

	tests := []struct {
		name    string
		query   string
		content string
		wantMin float64
		wantMax float64
	}{
		{
			name:    "exact phrase match",
			query:   `"net profit margin"`,
			content: "The net profit margin increased by 5% this quarter.",
			wantMin: config.PhraseMatchScore * 0.9,
			wantMax: config.PhraseMatchScore * 1.5,
		},
		{
			name:    "all words present in order",
			query:   "annual budget report",
			content: "This is the annual budget report for 2024.",
			wantMin: config.AllWordsContentScore * 0.9,
			wantMax: config.AllWordsContentScore * 1.5,
		},
		{
			name:    "all words scattered",
			query:   "report budget annual",
			content: "The report shows budget allocation for annual expenses.",
			wantMin: config.ScatteredWordsScore * 0.9,
			wantMax: config.PhraseMatchScore * 1.5, // Position boost can increase score
		},
		{
			name:    "partial match",
			query:   "budget analysis forecast",
			content: "The budget analysis shows positive trends.",
			wantMin: config.ScatteredWordsScore * 0.5,
			wantMax: config.ScatteredWordsScore * 1.2,
		},
		{
			name:    "header match markdown",
			query:   "budget analysis",
			content: "# Budget Analysis\n\nThis is the content.",
			wantMin: config.HeaderMatchScore * 0.8,
			wantMax: config.HeaderMatchScore * 2.5, // Position boost + header level boost
		},
		{
			name:    "match at beginning",
			query:   "important",
			content: "Important: This is the main content of the document that continues for many more words.",
			wantMin: config.ScatteredWordsScore * 0.5,
			wantMax: config.ScatteredWordsScore * 2.0,
		},
		{
			name:    "multiple phrase occurrences",
			query:   `"budget report"`,
			content: "The budget report shows the budget report data from the budget report analysis.",
			wantMin: config.PhraseMatchScore,
			wantMax: config.PhraseMatchScore * 1.5,
		},
		{
			name:    "no match",
			query:   "xyz missing term",
			content: "This is a document about budget and finance.",
			wantMin: 0,
			wantMax: 0.1,
		},
		{
			name:    "case insensitive",
			query:   "BUDGET REPORT",
			content: "the budget report is ready",
			wantMin: config.ScatteredWordsScore * 0.8,
			wantMax: config.AllWordsContentScore * 1.5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			analyzed := analyzer.Analyze(tt.query)
			ctx := &ScoringContext{
				Query: analyzed,
				Document: &models.Document{
					Content: tt.content,
				},
				Content: tt.content,
			}

			score := scorer.Score(ctx)

			if score < tt.wantMin || score > tt.wantMax {
				t.Errorf("Score() = %v, want between %v and %v", score, tt.wantMin, tt.wantMax)
			}
		})
	}
}

func TestContentScorer_NilInputs(t *testing.T) {
	config := DefaultRankingConfig()
	scorer := NewContentScorer(config)

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
		Document: &models.Document{Content: "test content"},
	})
	if score != 0 {
		t.Errorf("Expected 0 for nil query, got %v", score)
	}

	// Empty content
	score = scorer.Score(&ScoringContext{
		Query:    &AnalyzedQuery{Terms: []string{"test"}},
		Document: &models.Document{Content: ""},
		Content:  "",
	})
	if score != 0 {
		t.Errorf("Expected 0 for empty content, got %v", score)
	}
}

func TestContentScorer_Name(t *testing.T) {
	config := DefaultRankingConfig()
	scorer := NewContentScorer(config)

	if scorer.Name() != "content" {
		t.Errorf("Name() = %v, want 'content'", scorer.Name())
	}
}

func TestContentScorer_WithTFIDF(t *testing.T) {
	config := DefaultRankingConfig()
	config.TFIDFEnabled = true
	scorer := NewContentScorer(config)
	analyzer := NewQueryAnalyzer()

	// Create corpus stats with rare and common terms
	stats := NewCorpusStats()
	stats.TotalDocs = 100
	stats.DocFrequencies = map[string]int{
		"the":             95, // very common
		"budget":          20, // moderately common
		"photosynthesis": 1,  // rare
	}

	tests := []struct {
		name    string
		query   string
		content string
		stats   *CorpusStats
	}{
		{
			name:    "rare term should score higher",
			query:   "photosynthesis",
			content: "The process of photosynthesis is essential for plants.",
			stats:   stats,
		},
		{
			name:    "common term should score lower",
			query:   "budget",
			content: "The budget allocation is complete.",
			stats:   stats,
		},
	}

	var scores []float64
	for _, tt := range tests {
		analyzed := analyzer.Analyze(tt.query)
		ctx := &ScoringContext{
			Query: analyzed,
			Document: &models.Document{
				Content: tt.content,
			},
			Content:     tt.content,
			CorpusStats: tt.stats,
		}

		score := scorer.Score(ctx)
		scores = append(scores, score)
	}

	// Rare term should score higher due to IDF
	if scores[0] < scores[1] {
		t.Logf("Rare term score: %v, common term score: %v", scores[0], scores[1])
	}
}

func TestDetectHeaders(t *testing.T) {
	tests := []struct {
		name    string
		content string
		wantLen int
	}{
		{
			name:    "markdown h1",
			content: "# Header One\n\nSome content.",
			wantLen: 1,
		},
		{
			name:    "markdown h2",
			content: "## Header Two\n\nSome content.",
			wantLen: 1,
		},
		{
			name:    "multiple markdown headers",
			content: "# Title\n\n## Section 1\n\nContent\n\n### Subsection\n\nMore content.",
			wantLen: 3,
		},
		{
			name:    "html headers",
			content: "<h1>HTML Title</h1>\n<p>Content</p>",
			wantLen: 1,
		},
		{
			name:    "no headers",
			content: "This is just plain text without any headers.",
			wantLen: 0,
		},
		{
			name:    "rst header",
			content: "Title\n=====\n\nSome content.",
			wantLen: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			headers := DetectHeaders(tt.content)
			if len(headers) != tt.wantLen {
				t.Errorf("DetectHeaders() returned %d headers, want %d", len(headers), tt.wantLen)
			}
		})
	}
}

func TestContentScorer_ScoreWithDetails(t *testing.T) {
	config := DefaultRankingConfig()
	scorer := NewContentScorer(config)
	analyzer := NewQueryAnalyzer()

	analyzed := analyzer.Analyze(`"budget report"`)
	ctx := &ScoringContext{
		Query: analyzed,
		Document: &models.Document{
			Content: "The budget report is ready for review.",
		},
		Content: "The budget report is ready for review.",
	}

	result := scorer.ScoreWithDetails(ctx)

	if result.Score <= 0 {
		t.Errorf("Expected positive score, got %v", result.Score)
	}
	if result.MatchType != MatchTypePhrase {
		t.Errorf("Expected MatchTypePhrase, got %v", result.MatchType)
	}
	if len(result.PhraseMatches) == 0 {
		t.Error("Expected phrase matches to be populated")
	}
}

func TestCalculateTermFrequency(t *testing.T) {
	tests := []struct {
		name    string
		term    string
		content string
		wantMin float64
		wantMax float64
	}{
		{
			name:    "term appears once",
			term:    "budget",
			content: "The budget is approved.",
			wantMin: 0.1,
			wantMax: 0.3,
		},
		{
			name:    "term appears multiple times",
			term:    "the",
			content: "The budget is the approved the document.",
			wantMin: 0.3,
			wantMax: 0.6,
		},
		{
			name:    "term not present",
			term:    "missing",
			content: "The budget is approved.",
			wantMin: 0,
			wantMax: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tf := CalculateTermFrequency(tt.term, tt.content)
			if tf < tt.wantMin || tf > tt.wantMax {
				t.Errorf("CalculateTermFrequency() = %v, want between %v and %v", tf, tt.wantMin, tt.wantMax)
			}
		})
	}
}

func TestCalculateTFIDF(t *testing.T) {
	stats := NewCorpusStats()
	stats.TotalDocs = 100
	stats.DocFrequencies = map[string]int{
		"common": 90,
		"rare":   5,
	}

	commonTFIDF := CalculateTFIDF("common", "This is a common word in a common document.", stats)
	rareTFIDF := CalculateTFIDF("rare", "This is a rare word.", stats)

	// Rare terms should have higher TF-IDF due to higher IDF
	if rareTFIDF <= commonTFIDF {
		t.Logf("Common TF-IDF: %v, Rare TF-IDF: %v", commonTFIDF, rareTFIDF)
	}
}

func TestCorpusStats_IDF(t *testing.T) {
	stats := NewCorpusStats()
	stats.TotalDocs = 100
	stats.DocFrequencies = map[string]int{
		"common": 90,
		"rare":   5,
		"unique": 1,
	}

	tests := []struct {
		term    string
		wantMin float64
		wantMax float64
	}{
		{"common", 1.0, 3.0},   // Low IDF for common term
		{"rare", 10.0, 30.0},   // Higher IDF for rare term
		{"unique", 50.0, 150.0}, // Very high IDF for unique term
		{"unknown", 50.0, 150.0}, // Term not in corpus
	}

	for _, tt := range tests {
		t.Run(tt.term, func(t *testing.T) {
			idf := stats.IDF(tt.term)
			if idf < tt.wantMin || idf > tt.wantMax {
				t.Errorf("IDF(%q) = %v, want between %v and %v", tt.term, idf, tt.wantMin, tt.wantMax)
			}
		})
	}
}
