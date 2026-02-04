package ranking

import (
	"math"
	"time"
)

// RecencyMultiplier applies a boost based on how recently the file was modified.
type RecencyMultiplier struct {
	config *RankingConfig
}

// NewRecencyMultiplier creates a new RecencyMultiplier.
func NewRecencyMultiplier(config *RankingConfig) *RecencyMultiplier {
	return &RecencyMultiplier{config: config}
}

// Name returns the multiplier name.
func (m *RecencyMultiplier) Name() string {
	return "recency"
}

// Multiply applies the recency multiplier to the base score.
func (m *RecencyMultiplier) Multiply(ctx *ScoringContext, baseScore float64) float64 {
	if !m.config.RecencyEnabled || baseScore == 0 {
		return baseScore
	}

	modTime := ctx.ModTime
	if modTime.IsZero() {
		return baseScore
	}

	multiplier := m.calculateMultiplier(modTime)
	return baseScore * multiplier
}

// calculateMultiplier calculates the recency multiplier based on modification time.
func (m *RecencyMultiplier) calculateMultiplier(modTime time.Time) float64 {
	now := time.Now()
	age := now.Sub(modTime)

	// Last 24 hours
	if age < 24*time.Hour {
		return m.config.Recency24hMultiplier
	}

	// Last week
	if age < 7*24*time.Hour {
		return m.config.RecencyWeekMultiplier
	}

	// Last month
	if age < 30*24*time.Hour {
		return m.config.RecencyMonthMultiplier
	}

	// Older files get no boost
	return 1.0
}

// CalculateRecencyMultiplier is a standalone function to calculate recency multiplier.
func CalculateRecencyMultiplier(modTime time.Time, config *RankingConfig) float64 {
	m := NewRecencyMultiplier(config)
	return m.calculateMultiplier(modTime)
}

// QueryQualityMultiplier applies a multiplier based on match quality.
type QueryQualityMultiplier struct {
	config *RankingConfig
}

// NewQueryQualityMultiplier creates a new QueryQualityMultiplier.
func NewQueryQualityMultiplier(config *RankingConfig) *QueryQualityMultiplier {
	return &QueryQualityMultiplier{config: config}
}

// Name returns the multiplier name.
func (m *QueryQualityMultiplier) Name() string {
	return "query_quality"
}

// Multiply applies the query quality multiplier to the base score.
func (m *QueryQualityMultiplier) Multiply(ctx *ScoringContext, baseScore float64) float64 {
	if !m.config.QueryQualityEnabled || baseScore == 0 {
		return baseScore
	}

	matchType := m.determineMatchType(ctx)
	multiplier := m.getMultiplierForMatchType(matchType)

	return baseScore * multiplier
}

// determineMatchType determines the best match type for the document.
func (m *QueryQualityMultiplier) determineMatchType(ctx *ScoringContext) MatchType {
	if ctx.Query == nil || ctx.Document == nil {
		return MatchTypeNone
	}

	analyzer := NewQueryAnalyzer()
	tokens := analyzer.TokenizeForMatching(ctx.Query)

	// Check filename
	if ctx.Document.Title != "" {
		normalized := NormalizeFilename(ctx.Document.Title)

		// Check for phrase match in filename
		for _, phrase := range ctx.Query.Phrases {
			if FindPhrasePosition(phrase, normalized) != -1 {
				return MatchTypePhrase
			}
		}

		// Check for all words match
		if len(tokens) > 0 && AllTermsMatch(tokens, normalized) {
			if TermsInOrder(tokens, normalized) {
				return MatchTypePhrase
			}
			return MatchTypeAllWords
		}
	}

	// Check content
	content := ctx.Content
	if content == "" {
		content = ctx.Document.Content
	}
	if content != "" {
		// Check for phrase match in content
		for _, phrase := range ctx.Query.Phrases {
			if FindPhrasePosition(phrase, content) != -1 {
				return MatchTypePhrase
			}
		}

		// Check for all words match in content
		if len(tokens) > 0 {
			if AllTermsMatch(tokens, content) {
				if TermsInOrder(tokens, content) {
					return MatchTypePhrase
				}
				return MatchTypeAllWords
			}

			// Check for partial match
			if CountMatchingTerms(tokens, content) > 0 {
				return MatchTypePartial
			}
		}
	}

	return MatchTypeNone
}

// getMultiplierForMatchType returns the multiplier for a given match type.
func (m *QueryQualityMultiplier) getMultiplierForMatchType(matchType MatchType) float64 {
	switch matchType {
	case MatchTypeExact, MatchTypePhrase:
		return m.config.PhraseMatchMultiplier
	case MatchTypeAllWords:
		return m.config.AllWordsMultiplier
	case MatchTypePartial:
		return m.config.PartialMatchMultiplier
	default:
		return 1.0
	}
}

// FileSizeMultiplier normalizes scores based on file size.
type FileSizeMultiplier struct {
	config *RankingConfig
}

// NewFileSizeMultiplier creates a new FileSizeMultiplier.
func NewFileSizeMultiplier(config *RankingConfig) *FileSizeMultiplier {
	return &FileSizeMultiplier{config: config}
}

// Name returns the multiplier name.
func (m *FileSizeMultiplier) Name() string {
	return "file_size"
}

// Multiply applies the file size normalization to the base score.
func (m *FileSizeMultiplier) Multiply(ctx *ScoringContext, baseScore float64) float64 {
	if !m.config.FileSizeNormEnabled || baseScore == 0 {
		return baseScore
	}

	fileSize := ctx.FileSize
	if fileSize <= 0 {
		return baseScore
	}

	multiplier := m.calculateMultiplier(fileSize)
	return baseScore * multiplier
}

// calculateMultiplier calculates the file size multiplier.
// Smaller files with matches are considered more relevant.
func (m *FileSizeMultiplier) calculateMultiplier(fileSize int64) float64 {
	// Use logarithmic scaling to avoid extreme values
	// Small files (< 1KB) get slight boost
	// Large files (> 1MB) get slight penalty

	const (
		smallThreshold = 1024        // 1KB
		largeThreshold = 1024 * 1024 // 1MB
	)

	if fileSize < smallThreshold {
		// Small file boost: up to 1.1x
		return 1.0 + 0.1*(1.0-float64(fileSize)/float64(smallThreshold))
	}

	if fileSize > largeThreshold {
		// Large file penalty: down to 0.9x
		// Use log scale to avoid extreme penalties
		logSize := math.Log10(float64(fileSize) / float64(largeThreshold))
		penalty := math.Min(logSize*0.05, 0.1)
		return 1.0 - penalty
	}

	// Medium files: no adjustment
	return 1.0
}

// CalculateFileSizeMultiplier is a standalone function to calculate file size multiplier.
func CalculateFileSizeMultiplier(fileSize int64, config *RankingConfig) float64 {
	m := NewFileSizeMultiplier(config)
	return m.calculateMultiplier(fileSize)
}

// CombinedMultiplier applies multiple multipliers in sequence.
type CombinedMultiplier struct {
	multipliers []Multiplier
}

// NewCombinedMultiplier creates a combined multiplier from multiple multipliers.
func NewCombinedMultiplier(multipliers ...Multiplier) *CombinedMultiplier {
	return &CombinedMultiplier{multipliers: multipliers}
}

// Name returns the multiplier name.
func (m *CombinedMultiplier) Name() string {
	return "combined"
}

// Multiply applies all multipliers in sequence.
func (m *CombinedMultiplier) Multiply(ctx *ScoringContext, baseScore float64) float64 {
	score := baseScore
	for _, mult := range m.multipliers {
		score = mult.Multiply(ctx, score)
	}
	return score
}

// GetMultiplierValues returns the individual multiplier values for debugging.
func (m *CombinedMultiplier) GetMultiplierValues(ctx *ScoringContext) map[string]float64 {
	values := make(map[string]float64)
	testScore := 1.0 // Use 1.0 to get pure multiplier values

	for _, mult := range m.multipliers {
		result := mult.Multiply(ctx, testScore)
		values[mult.Name()] = result / testScore
	}

	return values
}

// DefaultMultipliers returns the default set of multipliers based on config.
func DefaultMultipliers(config *RankingConfig) []Multiplier {
	var multipliers []Multiplier

	if config.RecencyEnabled {
		multipliers = append(multipliers, NewRecencyMultiplier(config))
	}

	if config.QueryQualityEnabled {
		multipliers = append(multipliers, NewQueryQualityMultiplier(config))
	}

	if config.FileSizeNormEnabled {
		multipliers = append(multipliers, NewFileSizeMultiplier(config))
	}

	return multipliers
}

// ApplyMultipliers applies a list of multipliers to a base score.
func ApplyMultipliers(ctx *ScoringContext, baseScore float64, multipliers []Multiplier) float64 {
	score := baseScore
	for _, m := range multipliers {
		score = m.Multiply(ctx, score)
	}
	return score
}

// MultiplierResult contains detailed multiplier application results.
type MultiplierResult struct {
	FinalScore     float64
	BaseScore      float64
	MultiplierVals map[string]float64
}

// ApplyMultipliersWithDetails applies multipliers and returns detailed results.
func ApplyMultipliersWithDetails(ctx *ScoringContext, baseScore float64, multipliers []Multiplier) *MultiplierResult {
	result := &MultiplierResult{
		BaseScore:      baseScore,
		MultiplierVals: make(map[string]float64),
	}

	score := baseScore
	for _, m := range multipliers {
		prevScore := score
		score = m.Multiply(ctx, score)
		if prevScore != 0 {
			result.MultiplierVals[m.Name()] = score / prevScore
		} else {
			result.MultiplierVals[m.Name()] = 1.0
		}
	}

	result.FinalScore = score
	return result
}
