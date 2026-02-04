package ranking

import (
	"path/filepath"
	"strings"

	"github.com/hyperjump/sagasu/internal/models"
)

// FilenameScorer scores documents based on filename matching.
type FilenameScorer struct {
	config *RankingConfig
}

// NewFilenameScorer creates a new FilenameScorer with the given config.
func NewFilenameScorer(config *RankingConfig) *FilenameScorer {
	return &FilenameScorer{config: config}
}

// Name returns the scorer name.
func (s *FilenameScorer) Name() string {
	return "filename"
}

// Score calculates the filename match score.
func (s *FilenameScorer) Score(ctx *ScoringContext) float64 {
	if ctx.Document == nil || ctx.Query == nil {
		return 0
	}

	filename := ctx.Document.Title
	if filename == "" {
		return 0
	}

	// Get all matching tokens (terms + phrase words)
	analyzer := NewQueryAnalyzer()
	tokens := analyzer.TokenizeForMatching(ctx.Query)
	if len(tokens) == 0 && len(ctx.Query.Phrases) == 0 {
		return 0
	}

	score := 0.0

	// Check for exact filename match (highest priority)
	exactScore := s.scoreExactMatch(ctx.Query.Original, filename)
	if exactScore > 0 {
		score = max(score, exactScore)
	}

	// Check for phrase match in filename
	for _, phrase := range ctx.Query.Phrases {
		phraseScore := s.scorePhraseMatch(phrase, filename)
		score = max(score, phraseScore)
	}

	// Check for word matches
	if len(tokens) > 0 {
		wordScore := s.scoreWordMatch(tokens, filename)
		score = max(score, wordScore)
	}

	// Check for extension match
	extScore := s.scoreExtensionMatch(tokens, filename)
	score = max(score, extScore)

	// Check for prefix/substring matches
	substringScore := s.scoreSubstringMatch(tokens, filename)
	score = max(score, substringScore)

	// Apply multiple occurrence bonus
	score += s.multipleOccurrenceBonus(tokens, filename)

	return score
}

// scoreExactMatch checks for exact filename match (ignoring extension and case).
func (s *FilenameScorer) scoreExactMatch(query, filename string) float64 {
	normalized := NormalizeFilename(filename)
	queryNorm := strings.ToLower(strings.TrimSpace(query))

	// Exact match
	if normalized == queryNorm {
		return s.config.ExactFilenameScore
	}

	// Check if query matches filename without separators
	queryNoSep := strings.ReplaceAll(queryNorm, " ", "")
	filenameNoSep := strings.ReplaceAll(normalized, " ", "")
	if queryNoSep == filenameNoSep {
		return s.config.ExactFilenameScore * 0.95
	}

	return 0
}

// scorePhraseMatch scores phrase matches in filename.
func (s *FilenameScorer) scorePhraseMatch(phrase, filename string) float64 {
	normalized := NormalizeFilename(filename)
	phraseLower := strings.ToLower(phrase)

	if strings.Contains(normalized, phraseLower) {
		// Full phrase found in filename
		return s.config.AllWordsInOrderScore
	}

	return 0
}

// scoreWordMatch scores based on word matches in filename.
func (s *FilenameScorer) scoreWordMatch(terms []string, filename string) float64 {
	if len(terms) == 0 {
		return 0
	}

	normalized := NormalizeFilename(filename)

	// Check how many terms match
	matchCount := CountMatchingTerms(terms, normalized)
	if matchCount == 0 {
		return 0
	}

	totalTerms := len(terms)
	matchRatio := float64(matchCount) / float64(totalTerms)

	// All terms match
	if matchCount == totalTerms {
		// Check if they're in order
		if TermsInOrder(terms, normalized) {
			return s.config.AllWordsInOrderScore
		}
		return s.config.AllWordsAnyOrderScore
	}

	// Partial match - scale score by match ratio
	return s.config.SubstringMatchScore * matchRatio
}

// scoreExtensionMatch scores extension matches.
func (s *FilenameScorer) scoreExtensionMatch(terms []string, filename string) float64 {
	ext := ExtractExtension(filename)
	if ext == "" {
		return 0
	}

	for _, term := range terms {
		termClean := strings.TrimPrefix(strings.ToLower(term), ".")
		if termClean == ext {
			return s.config.ExtensionMatchScore
		}
	}

	return 0
}

// scoreSubstringMatch scores substring and prefix matches.
func (s *FilenameScorer) scoreSubstringMatch(terms []string, filename string) float64 {
	normalized := NormalizeFilename(filename)
	score := 0.0

	for _, term := range terms {
		// Check for prefix match on any word in filename
		if IsPrefixMatch(term, normalized) {
			prefixScore := s.config.PrefixMatchScore
			// Longer prefix matches score higher
			if len(term) > 3 {
				prefixScore *= 1.0 + float64(len(term)-3)*0.05
			}
			score = max(score, prefixScore)
		}

		// Check for substring match
		if strings.Contains(normalized, strings.ToLower(term)) {
			substringScore := s.config.SubstringMatchScore
			// Longer substring matches score higher
			if len(term) > 3 {
				substringScore *= 1.0 + float64(len(term)-3)*0.03
			}
			score = max(score, substringScore)
		}
	}

	return score
}

// multipleOccurrenceBonus adds bonus for multiple occurrences of terms.
func (s *FilenameScorer) multipleOccurrenceBonus(terms []string, filename string) float64 {
	normalized := NormalizeFilename(filename)
	bonus := 0.0

	for _, term := range terms {
		count := CountOccurrences(term, normalized)
		if count > 1 {
			// Diminishing returns for multiple occurrences
			bonus += s.config.MultipleOccurrenceBonus * (1.0 - 1.0/float64(count))
		}
	}

	return bonus
}

// ScoreFilenameOnly is a convenience function to score just a filename.
func ScoreFilenameOnly(config *RankingConfig, query, filename string) float64 {
	scorer := NewFilenameScorer(config)
	analyzer := NewQueryAnalyzer()
	analyzed := analyzer.Analyze(query)

	ctx := &ScoringContext{
		Query: analyzed,
		Document: &models.Document{
			Title: filename,
		},
	}

	return scorer.Score(ctx)
}

// MatchResult contains detailed matching information for debugging.
type MatchResult struct {
	Score       float64
	MatchType   MatchType
	MatchedTerms []string
	Details     string
}

// ScoreWithDetails returns detailed scoring information.
func (s *FilenameScorer) ScoreWithDetails(ctx *ScoringContext) *MatchResult {
	result := &MatchResult{
		Score:     s.Score(ctx),
		MatchType: MatchTypeNone,
	}

	if ctx.Document == nil || ctx.Query == nil {
		return result
	}

	filename := ctx.Document.Title
	if filename == "" {
		return result
	}

	analyzer := NewQueryAnalyzer()
	tokens := analyzer.TokenizeForMatching(ctx.Query)
	normalized := NormalizeFilename(filename)

	// Determine match type
	queryNorm := strings.ToLower(strings.TrimSpace(ctx.Query.Original))
	if normalized == queryNorm {
		result.MatchType = MatchTypeExact
		result.Details = "exact filename match"
	} else if AllTermsMatch(tokens, normalized) {
		if TermsInOrder(tokens, normalized) {
			result.MatchType = MatchTypePhrase
			result.Details = "all terms in order"
		} else {
			result.MatchType = MatchTypeAllWords
			result.Details = "all terms present (any order)"
		}
	} else if CountMatchingTerms(tokens, normalized) > 0 {
		result.MatchType = MatchTypePartial
		result.Details = "partial match"
	}

	// Collect matched terms
	for _, term := range tokens {
		if strings.Contains(normalized, strings.ToLower(term)) {
			result.MatchedTerms = append(result.MatchedTerms, term)
		}
	}

	return result
}

// ScoreFilenameWithBasename extracts basename from path and scores it.
func (s *FilenameScorer) ScoreFilenameWithBasename(ctx *ScoringContext) float64 {
	if ctx.FilePath == "" {
		return s.Score(ctx)
	}

	// Use basename from path if Title is empty
	if ctx.Document.Title == "" {
		basename := filepath.Base(ctx.FilePath)
		originalTitle := ctx.Document.Title
		ctx.Document.Title = basename
		score := s.Score(ctx)
		ctx.Document.Title = originalTitle
		return score
	}

	return s.Score(ctx)
}
