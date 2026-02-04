package ranking

import (
	"testing"
	"time"

	"github.com/hyperjump/sagasu/internal/models"
)

func TestRecencyMultiplier_Multiply(t *testing.T) {
	config := DefaultRankingConfig()
	config.RecencyEnabled = true
	mult := NewRecencyMultiplier(config)

	baseScore := 100.0
	now := time.Now()

	tests := []struct {
		name       string
		modTime    time.Time
		wantMin    float64
		wantMax    float64
	}{
		{
			name:    "last hour",
			modTime: now.Add(-30 * time.Minute),
			wantMin: baseScore * config.Recency24hMultiplier * 0.99,
			wantMax: baseScore * config.Recency24hMultiplier * 1.01,
		},
		{
			name:    "yesterday",
			modTime: now.Add(-20 * time.Hour),
			wantMin: baseScore * config.Recency24hMultiplier * 0.99,
			wantMax: baseScore * config.Recency24hMultiplier * 1.01,
		},
		{
			name:    "3 days ago",
			modTime: now.Add(-3 * 24 * time.Hour),
			wantMin: baseScore * config.RecencyWeekMultiplier * 0.99,
			wantMax: baseScore * config.RecencyWeekMultiplier * 1.01,
		},
		{
			name:    "2 weeks ago",
			modTime: now.Add(-14 * 24 * time.Hour),
			wantMin: baseScore * config.RecencyMonthMultiplier * 0.99,
			wantMax: baseScore * config.RecencyMonthMultiplier * 1.01,
		},
		{
			name:    "2 months ago",
			modTime: now.Add(-60 * 24 * time.Hour),
			wantMin: baseScore * 0.99,
			wantMax: baseScore * 1.01,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := &ScoringContext{
				ModTime: tt.modTime,
			}

			result := mult.Multiply(ctx, baseScore)

			if result < tt.wantMin || result > tt.wantMax {
				t.Errorf("Multiply() = %v, want between %v and %v", result, tt.wantMin, tt.wantMax)
			}
		})
	}
}

func TestRecencyMultiplier_Disabled(t *testing.T) {
	config := DefaultRankingConfig()
	config.RecencyEnabled = false
	mult := NewRecencyMultiplier(config)

	ctx := &ScoringContext{
		ModTime: time.Now().Add(-30 * time.Minute),
	}

	baseScore := 100.0
	result := mult.Multiply(ctx, baseScore)

	if result != baseScore {
		t.Errorf("Expected no change when disabled, got %v", result)
	}
}

func TestRecencyMultiplier_ZeroTime(t *testing.T) {
	config := DefaultRankingConfig()
	config.RecencyEnabled = true
	mult := NewRecencyMultiplier(config)

	ctx := &ScoringContext{
		ModTime: time.Time{}, // Zero time
	}

	baseScore := 100.0
	result := mult.Multiply(ctx, baseScore)

	if result != baseScore {
		t.Errorf("Expected no change for zero time, got %v", result)
	}
}

func TestRecencyMultiplier_Name(t *testing.T) {
	config := DefaultRankingConfig()
	mult := NewRecencyMultiplier(config)

	if mult.Name() != "recency" {
		t.Errorf("Name() = %v, want 'recency'", mult.Name())
	}
}

func TestQueryQualityMultiplier_Multiply(t *testing.T) {
	config := DefaultRankingConfig()
	config.QueryQualityEnabled = true
	mult := NewQueryQualityMultiplier(config)
	analyzer := NewQueryAnalyzer()

	baseScore := 100.0

	tests := []struct {
		name     string
		query    string
		filename string
		content  string
		wantMin  float64
		wantMax  float64
	}{
		{
			name:     "phrase match in filename",
			query:    "budget report",
			filename: "budget_report.pdf",
			content:  "",
			wantMin:  baseScore * config.PhraseMatchMultiplier * 0.99,
			wantMax:  baseScore * config.PhraseMatchMultiplier * 1.01,
		},
		{
			name:     "all words match",
			query:    "budget analysis",
			filename: "budget_analysis_2024.pdf",
			content:  "",
			wantMin:  baseScore * config.AllWordsMultiplier * 0.99,
			wantMax:  baseScore * config.PhraseMatchMultiplier * 1.01,
		},
		{
			name:     "partial match",
			query:    "budget forecast analysis",
			filename: "budget.pdf",
			content:  "This is about budgeting.",
			wantMin:  baseScore * config.PartialMatchMultiplier * 0.99,
			wantMax:  baseScore * config.AllWordsMultiplier * 1.01,
		},
		{
			name:     "phrase in content",
			query:    `"important data"`,
			filename: "report.pdf",
			content:  "This contains important data for review.",
			wantMin:  baseScore * config.PhraseMatchMultiplier * 0.99,
			wantMax:  baseScore * config.PhraseMatchMultiplier * 1.01,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			analyzed := analyzer.Analyze(tt.query)
			ctx := &ScoringContext{
				Query: analyzed,
				Document: &models.Document{
					Title:   tt.filename,
					Content: tt.content,
				},
				Content: tt.content,
			}

			result := mult.Multiply(ctx, baseScore)

			if result < tt.wantMin || result > tt.wantMax {
				t.Errorf("Multiply() = %v, want between %v and %v", result, tt.wantMin, tt.wantMax)
			}
		})
	}
}

func TestQueryQualityMultiplier_Disabled(t *testing.T) {
	config := DefaultRankingConfig()
	config.QueryQualityEnabled = false
	mult := NewQueryQualityMultiplier(config)

	ctx := &ScoringContext{
		Query: &AnalyzedQuery{Terms: []string{"test"}},
		Document: &models.Document{
			Title:   "test.txt",
			Content: "test content",
		},
	}

	baseScore := 100.0
	result := mult.Multiply(ctx, baseScore)

	if result != baseScore {
		t.Errorf("Expected no change when disabled, got %v", result)
	}
}

func TestQueryQualityMultiplier_Name(t *testing.T) {
	config := DefaultRankingConfig()
	mult := NewQueryQualityMultiplier(config)

	if mult.Name() != "query_quality" {
		t.Errorf("Name() = %v, want 'query_quality'", mult.Name())
	}
}

func TestFileSizeMultiplier_Multiply(t *testing.T) {
	config := DefaultRankingConfig()
	config.FileSizeNormEnabled = true
	mult := NewFileSizeMultiplier(config)

	baseScore := 100.0

	tests := []struct {
		name     string
		fileSize int64
		wantMin  float64
		wantMax  float64
	}{
		{
			name:     "small file (100 bytes)",
			fileSize: 100,
			wantMin:  baseScore * 1.0,
			wantMax:  baseScore * 1.15,
		},
		{
			name:     "medium file (100KB)",
			fileSize: 100 * 1024,
			wantMin:  baseScore * 0.95,
			wantMax:  baseScore * 1.05,
		},
		{
			name:     "large file (10MB)",
			fileSize: 10 * 1024 * 1024,
			wantMin:  baseScore * 0.85,
			wantMax:  baseScore * 1.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := &ScoringContext{
				FileSize: tt.fileSize,
			}

			result := mult.Multiply(ctx, baseScore)

			if result < tt.wantMin || result > tt.wantMax {
				t.Errorf("Multiply() = %v, want between %v and %v", result, tt.wantMin, tt.wantMax)
			}
		})
	}
}

func TestFileSizeMultiplier_Disabled(t *testing.T) {
	config := DefaultRankingConfig()
	config.FileSizeNormEnabled = false
	mult := NewFileSizeMultiplier(config)

	ctx := &ScoringContext{
		FileSize: 100,
	}

	baseScore := 100.0
	result := mult.Multiply(ctx, baseScore)

	if result != baseScore {
		t.Errorf("Expected no change when disabled, got %v", result)
	}
}

func TestFileSizeMultiplier_ZeroSize(t *testing.T) {
	config := DefaultRankingConfig()
	config.FileSizeNormEnabled = true
	mult := NewFileSizeMultiplier(config)

	ctx := &ScoringContext{
		FileSize: 0,
	}

	baseScore := 100.0
	result := mult.Multiply(ctx, baseScore)

	if result != baseScore {
		t.Errorf("Expected no change for zero size, got %v", result)
	}
}

func TestFileSizeMultiplier_Name(t *testing.T) {
	config := DefaultRankingConfig()
	mult := NewFileSizeMultiplier(config)

	if mult.Name() != "file_size" {
		t.Errorf("Name() = %v, want 'file_size'", mult.Name())
	}
}

func TestCombinedMultiplier(t *testing.T) {
	config := DefaultRankingConfig()
	config.RecencyEnabled = true
	config.QueryQualityEnabled = true

	recency := NewRecencyMultiplier(config)
	quality := NewQueryQualityMultiplier(config)
	combined := NewCombinedMultiplier(recency, quality)

	analyzer := NewQueryAnalyzer()
	analyzed := analyzer.Analyze("budget report")

	ctx := &ScoringContext{
		Query:   analyzed,
		ModTime: time.Now().Add(-30 * time.Minute),
		Document: &models.Document{
			Title:   "budget_report.pdf",
			Content: "",
		},
	}

	baseScore := 100.0
	result := combined.Multiply(ctx, baseScore)

	// Should be greater than base due to both multipliers
	expectedMin := baseScore * config.Recency24hMultiplier * 0.9
	if result < expectedMin {
		t.Errorf("Combined multiplier result %v < expected minimum %v", result, expectedMin)
	}
}

func TestCombinedMultiplier_Name(t *testing.T) {
	combined := NewCombinedMultiplier()

	if combined.Name() != "combined" {
		t.Errorf("Name() = %v, want 'combined'", combined.Name())
	}
}

func TestCombinedMultiplier_GetMultiplierValues(t *testing.T) {
	config := DefaultRankingConfig()
	config.RecencyEnabled = true
	config.QueryQualityEnabled = false

	recency := NewRecencyMultiplier(config)
	combined := NewCombinedMultiplier(recency)

	ctx := &ScoringContext{
		ModTime: time.Now().Add(-30 * time.Minute),
	}

	values := combined.GetMultiplierValues(ctx)

	if _, ok := values["recency"]; !ok {
		t.Error("Expected recency multiplier value")
	}
}

func TestDefaultMultipliers(t *testing.T) {
	config := DefaultRankingConfig()
	config.RecencyEnabled = true
	config.QueryQualityEnabled = true
	config.FileSizeNormEnabled = true

	multipliers := DefaultMultipliers(config)

	if len(multipliers) != 3 {
		t.Errorf("Expected 3 multipliers, got %d", len(multipliers))
	}
}

func TestApplyMultipliers(t *testing.T) {
	config := DefaultRankingConfig()
	config.RecencyEnabled = true

	multipliers := []Multiplier{NewRecencyMultiplier(config)}

	ctx := &ScoringContext{
		ModTime: time.Now().Add(-30 * time.Minute),
	}

	baseScore := 100.0
	result := ApplyMultipliers(ctx, baseScore, multipliers)

	expected := baseScore * config.Recency24hMultiplier
	if result < expected*0.99 || result > expected*1.01 {
		t.Errorf("ApplyMultipliers() = %v, want ~%v", result, expected)
	}
}

func TestApplyMultipliersWithDetails(t *testing.T) {
	config := DefaultRankingConfig()
	config.RecencyEnabled = true

	multipliers := []Multiplier{NewRecencyMultiplier(config)}

	ctx := &ScoringContext{
		ModTime: time.Now().Add(-30 * time.Minute),
	}

	baseScore := 100.0
	result := ApplyMultipliersWithDetails(ctx, baseScore, multipliers)

	if result.BaseScore != baseScore {
		t.Errorf("BaseScore = %v, want %v", result.BaseScore, baseScore)
	}
	if result.FinalScore < baseScore {
		t.Errorf("FinalScore should be >= BaseScore")
	}
	if _, ok := result.MultiplierVals["recency"]; !ok {
		t.Error("Expected recency multiplier in details")
	}
}

func TestCalculateRecencyMultiplier(t *testing.T) {
	config := DefaultRankingConfig()

	// Recent file
	mult := CalculateRecencyMultiplier(time.Now().Add(-1*time.Hour), config)
	if mult != config.Recency24hMultiplier {
		t.Errorf("Expected %v for recent file, got %v", config.Recency24hMultiplier, mult)
	}

	// Old file
	mult = CalculateRecencyMultiplier(time.Now().Add(-60*24*time.Hour), config)
	if mult != 1.0 {
		t.Errorf("Expected 1.0 for old file, got %v", mult)
	}
}

func TestCalculateFileSizeMultiplier(t *testing.T) {
	config := DefaultRankingConfig()

	// Small file should get boost
	mult := CalculateFileSizeMultiplier(100, config)
	if mult <= 1.0 {
		t.Errorf("Expected >1.0 for small file, got %v", mult)
	}

	// Large file should get penalty
	mult = CalculateFileSizeMultiplier(10*1024*1024, config)
	if mult >= 1.0 {
		t.Errorf("Expected <1.0 for large file, got %v", mult)
	}
}
