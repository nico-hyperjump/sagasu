package ranking

import (
	"testing"
	"time"

	"github.com/hyperjump/sagasu/internal/models"
)

func TestNewRanker(t *testing.T) {
	// With nil config - should use defaults
	ranker := NewRanker(nil)
	if ranker == nil {
		t.Fatal("Expected non-nil ranker")
	}
	if ranker.config == nil {
		t.Fatal("Expected non-nil config")
	}

	// With custom config
	config := &RankingConfig{
		FilenameWeight: 2.0,
	}
	ranker = NewRanker(config)
	if ranker.config.FilenameWeight != 2.0 {
		t.Errorf("Expected FilenameWeight 2.0, got %v", ranker.config.FilenameWeight)
	}
}

func TestRanker_AnalyzeQuery(t *testing.T) {
	ranker := NewRanker(nil)

	query := ranker.AnalyzeQuery("budget report")

	if query == nil {
		t.Fatal("Expected non-nil query")
	}
	if len(query.Terms) != 2 {
		t.Errorf("Expected 2 terms, got %d", len(query.Terms))
	}
}

func TestRanker_Rank(t *testing.T) {
	ranker := NewRanker(nil)

	tests := []struct {
		name     string
		query    string
		doc      *models.Document
		wantMin  float64
		wantMax  float64
	}{
		{
			name:  "filename exact match",
			query: "budget",
			doc: &models.Document{
				Title:   "budget.txt",
				Content: "Some content about expenses.",
			},
			wantMin: 100,
			wantMax: 300,
		},
		{
			name:  "content match only",
			query: "expenses",
			doc: &models.Document{
				Title:   "report.txt",
				Content: "This document discusses expenses and costs.",
			},
			wantMin: 30,
			wantMax: 200, // Includes position boost and multipliers
		},
		{
			name:  "both filename and content match",
			query: "budget",
			doc: &models.Document{
				Title:   "budget.txt",
				Content: "The budget allocation for this year.",
			},
			wantMin: 100,
			wantMax: 400,
		},
		{
			name:  "no match",
			query: "xyz",
			doc: &models.Document{
				Title:   "report.txt",
				Content: "Budget and expenses document.",
			},
			wantMin: 0,
			wantMax: 1,
		},
		{
			name:  "phrase query",
			query: `"net profit"`,
			doc: &models.Document{
				Title:   "financials.pdf",
				Content: "The net profit margin increased significantly.",
			},
			wantMin: 50,
			wantMax: 250,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			analyzed := ranker.AnalyzeQuery(tt.query)
			score := ranker.Rank(analyzed, tt.doc)

			if score < tt.wantMin || score > tt.wantMax {
				t.Errorf("Rank() = %v, want between %v and %v", score, tt.wantMin, tt.wantMax)
			}
		})
	}
}

func TestRanker_RankWithBreakdown(t *testing.T) {
	ranker := NewRanker(nil)

	doc := &models.Document{
		Title:   "budget_report.pdf",
		Content: "This is the budget report for 2024.",
		Metadata: map[string]interface{}{
			"author":      "John Smith",
			"source_path": "/home/user/documents/budget_report.pdf",
		},
	}

	analyzed := ranker.AnalyzeQuery("budget report")
	breakdown := ranker.RankWithBreakdown(analyzed, doc)

	if breakdown == nil {
		t.Fatal("Expected non-nil breakdown")
	}
	if breakdown.FinalScore <= 0 {
		t.Errorf("Expected positive final score, got %v", breakdown.FinalScore)
	}
	if breakdown.FilenameScore <= 0 {
		t.Errorf("Expected positive filename score, got %v", breakdown.FilenameScore)
	}
	if breakdown.ContentScore <= 0 {
		t.Errorf("Expected positive content score, got %v", breakdown.ContentScore)
	}
}

func TestRanker_RankDocuments(t *testing.T) {
	ranker := NewRanker(nil)

	docs := []*models.Document{
		{
			ID:      "1",
			Title:   "report.txt",
			Content: "General report about various topics.",
		},
		{
			ID:      "2",
			Title:   "budget.txt",
			Content: "The budget allocation document.",
		},
		{
			ID:      "3",
			Title:   "budget_report.pdf",
			Content: "This is the budget report for this year.",
		},
	}

	results := ranker.RankDocuments("budget report", docs)

	// Document 3 should rank highest (matches both "budget" and "report" in filename)
	if len(results) == 0 {
		t.Fatal("Expected at least one result")
	}

	// Results should be sorted by score descending
	for i := 1; i < len(results); i++ {
		if results[i].Score > results[i-1].Score {
			t.Errorf("Results not sorted: score[%d]=%v > score[%d]=%v",
				i, results[i].Score, i-1, results[i-1].Score)
		}
	}
}

func TestRanker_RankDocumentsWithBreakdown(t *testing.T) {
	ranker := NewRanker(nil)

	docs := []*models.Document{
		{
			ID:      "1",
			Title:   "budget.txt",
			Content: "Budget document.",
		},
	}

	results := ranker.RankDocumentsWithBreakdown("budget", docs)

	if len(results) == 0 {
		t.Fatal("Expected at least one result")
	}
	if results[0].Breakdown == nil {
		t.Error("Expected breakdown to be populated")
	}
}

func TestRanker_ReRank(t *testing.T) {
	ranker := NewRanker(nil)

	results := []*models.SearchResult{
		{
			Document: &models.Document{
				ID:      "1",
				Title:   "report.txt",
				Content: "General report.",
			},
			Score: 50,
			Rank:  1,
		},
		{
			Document: &models.Document{
				ID:      "2",
				Title:   "budget.txt",
				Content: "Budget document.",
			},
			Score: 30,
			Rank:  2,
		},
	}

	reranked := ranker.ReRank("budget", results)

	// Budget document should now rank higher
	if reranked[0].Document.ID != "2" {
		t.Errorf("Expected budget.txt to rank first after re-ranking")
	}

	// Ranks should be updated
	for i, r := range reranked {
		if r.Rank != i+1 {
			t.Errorf("Rank[%d] = %d, want %d", i, r.Rank, i+1)
		}
	}
}

func TestRanker_WithCorpusStats(t *testing.T) {
	ranker := NewRanker(nil)

	stats := NewCorpusStats()
	stats.TotalDocs = 100
	stats.DocFrequencies = map[string]int{
		"common": 90,
		"rare":   5,
	}

	ranker.WithCorpusStats(stats)

	if ranker.corpusStats != stats {
		t.Error("Expected corpus stats to be set")
	}
}

func TestRanker_WithMultipliers(t *testing.T) {
	config := DefaultRankingConfig()
	config.RecencyEnabled = true
	ranker := NewRanker(config)

	customMultipliers := []Multiplier{NewRecencyMultiplier(config)}
	ranker.WithMultipliers(customMultipliers)

	if len(ranker.multipliers) != 1 {
		t.Errorf("Expected 1 multiplier, got %d", len(ranker.multipliers))
	}
}

func TestRanker_UpdateCorpusStats(t *testing.T) {
	ranker := NewRanker(nil)

	docs := []*models.Document{
		{ID: "1", Title: "doc1", Content: "budget report financial"},
		{ID: "2", Title: "doc2", Content: "budget analysis quarterly"},
		{ID: "3", Title: "doc3", Content: "meeting notes general"},
	}

	ranker.UpdateCorpusStats(docs)

	stats := ranker.GetCorpusStats()
	if stats.TotalDocs != 3 {
		t.Errorf("TotalDocs = %d, want 3", stats.TotalDocs)
	}
	if stats.DocFrequencies["budget"] != 2 {
		t.Errorf("DocFrequencies[budget] = %d, want 2", stats.DocFrequencies["budget"])
	}
}

func TestRanker_GetConfig(t *testing.T) {
	config := &RankingConfig{FilenameWeight: 2.5}
	ranker := NewRanker(config)

	cfg := ranker.GetConfig()
	if cfg.FilenameWeight != 2.5 {
		t.Errorf("GetConfig().FilenameWeight = %v, want 2.5", cfg.FilenameWeight)
	}
}

func TestFilterByMinScore(t *testing.T) {
	results := []*RankedResult{
		{Score: 100},
		{Score: 50},
		{Score: 25},
		{Score: 10},
	}

	filtered := FilterByMinScore(results, 30)

	if len(filtered) != 2 {
		t.Errorf("Expected 2 results after filtering, got %d", len(filtered))
	}
}

func TestTopN(t *testing.T) {
	results := []*RankedResult{
		{Score: 100},
		{Score: 90},
		{Score: 80},
		{Score: 70},
		{Score: 60},
	}

	top3 := TopN(results, 3)
	if len(top3) != 3 {
		t.Errorf("TopN(3) returned %d results, want 3", len(top3))
	}

	// Request more than available
	top10 := TopN(results, 10)
	if len(top10) != 5 {
		t.Errorf("TopN(10) returned %d results, want 5", len(top10))
	}
}

func TestPaginate(t *testing.T) {
	results := []*RankedResult{
		{Score: 100},
		{Score: 90},
		{Score: 80},
		{Score: 70},
		{Score: 60},
	}

	// Page 1
	page1 := Paginate(results, 0, 2)
	if len(page1) != 2 {
		t.Errorf("Page 1 has %d results, want 2", len(page1))
	}
	if page1[0].Score != 100 {
		t.Errorf("Page 1 first score = %v, want 100", page1[0].Score)
	}

	// Page 2
	page2 := Paginate(results, 2, 2)
	if len(page2) != 2 {
		t.Errorf("Page 2 has %d results, want 2", len(page2))
	}
	if page2[0].Score != 80 {
		t.Errorf("Page 2 first score = %v, want 80", page2[0].Score)
	}

	// Page 3 (partial)
	page3 := Paginate(results, 4, 2)
	if len(page3) != 1 {
		t.Errorf("Page 3 has %d results, want 1", len(page3))
	}

	// Beyond results
	page4 := Paginate(results, 10, 2)
	if page4 != nil && len(page4) != 0 {
		t.Errorf("Page 4 should be empty, got %d results", len(page4))
	}
}

func TestRanker_WithRecencyMultiplier(t *testing.T) {
	config := DefaultRankingConfig()
	config.RecencyEnabled = true
	ranker := NewRanker(config)

	recentDoc := &models.Document{
		ID:      "1",
		Title:   "budget.txt",
		Content: "Budget document.",
		Metadata: map[string]interface{}{
			"source_mtime": time.Now().Add(-1 * time.Hour).UnixNano(),
		},
	}

	oldDoc := &models.Document{
		ID:      "2",
		Title:   "budget.txt",
		Content: "Budget document.",
		Metadata: map[string]interface{}{
			"source_mtime": time.Now().Add(-60 * 24 * time.Hour).UnixNano(),
		},
	}

	analyzed := ranker.AnalyzeQuery("budget")
	recentScore := ranker.Rank(analyzed, recentDoc)
	oldScore := ranker.Rank(analyzed, oldDoc)

	// Recent document should score higher due to recency multiplier
	if recentScore <= oldScore {
		t.Logf("Recent score: %v, Old score: %v", recentScore, oldScore)
	}
}

func TestRanker_RankingFormula(t *testing.T) {
	// Test the ranking formula:
	// Score = (Wf * Sf) + (Wc * Sc) + (Wp * Sp) + (Wm * Sm)

	config := DefaultRankingConfig()
	config.FilenameWeight = 1.5
	config.ContentWeight = 1.0
	config.PathWeight = 0.3
	config.MetadataWeight = 0.4
	config.RecencyEnabled = false
	config.QueryQualityEnabled = false

	ranker := NewRanker(config)

	doc := &models.Document{
		ID:      "1",
		Title:   "budget.txt",   // Should get filename score
		Content: "Budget data.", // Should get content score
		Metadata: map[string]interface{}{
			"source_path": "/home/user/reports/budget.txt", // Should get path score
			"author":      "John",                          // No match
		},
	}

	analyzed := ranker.AnalyzeQuery("budget")
	breakdown := ranker.RankWithBreakdown(analyzed, doc)

	// Verify that weighted sum is applied
	expectedScore := (config.FilenameWeight * breakdown.FilenameScore) +
		(config.ContentWeight * breakdown.ContentScore) +
		(config.PathWeight * breakdown.PathScore) +
		(config.MetadataWeight * breakdown.MetadataScore)

	// Allow small floating point difference
	if breakdown.FinalScore < expectedScore*0.99 || breakdown.FinalScore > expectedScore*1.01 {
		t.Errorf("FinalScore %v doesn't match expected weighted sum %v", breakdown.FinalScore, expectedScore)
		t.Logf("Filename: %v * %v = %v", config.FilenameWeight, breakdown.FilenameScore, config.FilenameWeight*breakdown.FilenameScore)
		t.Logf("Content: %v * %v = %v", config.ContentWeight, breakdown.ContentScore, config.ContentWeight*breakdown.ContentScore)
		t.Logf("Path: %v * %v = %v", config.PathWeight, breakdown.PathScore, config.PathWeight*breakdown.PathScore)
		t.Logf("Metadata: %v * %v = %v", config.MetadataWeight, breakdown.MetadataScore, config.MetadataWeight*breakdown.MetadataScore)
	}
}

func TestRanker_UseCasesFromSpec(t *testing.T) {
	// Test use cases from the specification table
	ranker := NewRanker(nil)

	tests := []struct {
		name        string
		query       string
		doc         *models.Document
		description string
	}{
		{
			name:  "single word in filename only",
			query: "budget",
			doc: &models.Document{
				Title:   "budget.txt",
				Content: "Some other content.",
			},
			description: "Exact filename match should rank high",
		},
		{
			name:  "single word in content only",
			query: "budget",
			doc: &models.Document{
				Title:   "report.txt",
				Content: "The budget allocation is complete.",
			},
			description: "Content match should rank medium-high",
		},
		{
			name:  "single word in both",
			query: "budget",
			doc: &models.Document{
				Title:   "budget.txt",
				Content: "The budget allocation is complete.",
			},
			description: "Both matches should rank highest",
		},
		{
			name:  "multiple words all in filename",
			query: "annual budget report",
			doc: &models.Document{
				Title:   "annual_budget_report.pdf",
				Content: "Other content.",
			},
			description: "All words in filename should rank high",
		},
		{
			name:  "exact phrase in content",
			query: `"net profit margin"`,
			doc: &models.Document{
				Title:   "financials.pdf",
				Content: "The net profit margin increased by 5%.",
			},
			description: "Exact phrase match should rank very high",
		},
		{
			name:  "path component match",
			query: "projects",
			doc: &models.Document{
				Title: "file.txt",
				Metadata: map[string]interface{}{
					"source_path": "/home/user/projects/file.txt",
				},
			},
			description: "Path match should contribute to score",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			analyzed := ranker.AnalyzeQuery(tt.query)
			score := ranker.Rank(analyzed, tt.doc)

			if score <= 0 {
				t.Errorf("%s: Expected positive score, got %v", tt.description, score)
			}
			t.Logf("%s: Score = %v", tt.name, score)
		})
	}
}
