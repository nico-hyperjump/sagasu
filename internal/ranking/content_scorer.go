package ranking

import (
	"math"
	"regexp"
	"strings"
)

// ContentScorer scores documents based on content matching with TF-IDF, phrase matching, and position boosts.
type ContentScorer struct {
	config *RankingConfig
}

// NewContentScorer creates a new ContentScorer with the given config.
func NewContentScorer(config *RankingConfig) *ContentScorer {
	return &ContentScorer{config: config}
}

// Name returns the scorer name.
func (s *ContentScorer) Name() string {
	return "content"
}

// Score calculates the content match score.
func (s *ContentScorer) Score(ctx *ScoringContext) float64 {
	if ctx.Document == nil || ctx.Query == nil {
		return 0
	}

	content := ctx.Content
	if content == "" {
		content = ctx.Document.Content
	}
	if content == "" {
		return 0
	}

	analyzer := NewQueryAnalyzer()
	tokens := analyzer.TokenizeForMatching(ctx.Query)

	score := 0.0

	// Score phrase matches (highest priority)
	for _, phrase := range ctx.Query.Phrases {
		phraseScore := s.scorePhraseMatch(phrase, content)
		score = max(score, phraseScore)
	}

	// Score header matches
	headerScore := s.scoreHeaderMatch(tokens, content)
	score = max(score, headerScore)

	// Score term matches with TF-IDF
	termScore := s.scoreTermMatches(tokens, content, ctx.CorpusStats)
	score = max(score, termScore)

	// Apply position boost if matches found near the beginning
	if score > 0 && s.config.PositionBoostEnabled {
		posMultiplier := s.calculatePositionMultiplier(tokens, ctx.Query.Phrases, content)
		score *= posMultiplier
	}

	return score
}

// scorePhraseMatch scores exact phrase matches in content.
func (s *ContentScorer) scorePhraseMatch(phrase, content string) float64 {
	pos := FindPhrasePosition(phrase, content)
	if pos == -1 {
		return 0
	}

	// Exact phrase match found
	baseScore := s.config.PhraseMatchScore

	// Multiple phrase occurrences get a small bonus
	count := strings.Count(strings.ToLower(content), strings.ToLower(phrase))
	if count > 1 {
		// Diminishing returns for multiple occurrences
		bonus := math.Min(float64(count-1)*5, 20)
		baseScore += bonus
	}

	return baseScore
}

// scoreHeaderMatch scores matches in headers (markdown, HTML).
func (s *ContentScorer) scoreHeaderMatch(terms []string, content string) float64 {
	if len(terms) == 0 {
		return 0
	}

	headers := DetectHeaders(content)
	if len(headers) == 0 {
		return 0
	}

	maxScore := 0.0
	for _, header := range headers {
		matchCount := CountMatchingTerms(terms, header.Text)
		if matchCount == 0 {
			continue
		}

		// Base score for header match
		score := s.config.HeaderMatchScore

		// Higher level headers (h1, h2) get more weight
		levelBonus := 1.0 + (5.0-float64(header.Level))*0.1
		score *= levelBonus

		// Bonus for matching more terms
		matchRatio := float64(matchCount) / float64(len(terms))
		score *= matchRatio

		maxScore = max(maxScore, score)
	}

	return maxScore
}

// scoreTermMatches scores individual term matches using TF-IDF.
func (s *ContentScorer) scoreTermMatches(terms []string, content string, stats *CorpusStats) float64 {
	if len(terms) == 0 {
		return 0
	}

	contentLower := strings.ToLower(content)
	matchCount := CountMatchingTerms(terms, content)

	if matchCount == 0 {
		return 0
	}

	// Calculate base score based on match ratio
	matchRatio := float64(matchCount) / float64(len(terms))
	var baseScore float64

	if matchCount == len(terms) {
		// All terms match
		if TermsInOrder(terms, content) {
			baseScore = s.config.AllWordsContentScore
		} else {
			baseScore = s.config.ScatteredWordsScore
		}
	} else {
		// Partial match
		baseScore = s.config.ScatteredWordsScore * matchRatio
	}

	// Apply TF-IDF multiplier if enabled and stats available
	if s.config.TFIDFEnabled && stats != nil {
		tfidfMultiplier := s.calculateTFIDFMultiplier(terms, contentLower, stats)
		baseScore *= tfidfMultiplier
	}

	return baseScore
}

// calculateTFIDFMultiplier calculates a TF-IDF based multiplier.
func (s *ContentScorer) calculateTFIDFMultiplier(terms []string, contentLower string, stats *CorpusStats) float64 {
	if stats == nil || stats.TotalDocs == 0 {
		return 1.0
	}

	totalTFIDF := 0.0
	matchingTerms := 0

	// Split content into words for proper TF calculation
	words := strings.Fields(contentLower)
	totalWords := len(words)
	if totalWords == 0 {
		return 1.0
	}

	for _, term := range terms {
		// Term Frequency: count of term / total words
		count := CountOccurrences(term, contentLower)
		if count == 0 {
			continue
		}
		matchingTerms++

		tf := float64(count) / float64(totalWords)

		// Inverse Document Frequency
		idf := stats.IDF(term)

		// TF-IDF
		tfidf := tf * idf
		totalTFIDF += tfidf
	}

	if matchingTerms == 0 {
		return 1.0
	}

	// Average TF-IDF and scale to a multiplier
	avgTFIDF := totalTFIDF / float64(matchingTerms)

	// Scale to a reasonable multiplier range (1.0 to max)
	multiplier := 1.0 + avgTFIDF*10 // Scale factor, tune as needed
	return math.Min(multiplier, s.config.MaxTFIDFMultiplier)
}

// calculatePositionMultiplier calculates a multiplier based on match position.
func (s *ContentScorer) calculatePositionMultiplier(terms []string, phrases []string, content string) float64 {
	contentLen := len(content)
	if contentLen == 0 {
		return 1.0
	}

	threshold := int(float64(contentLen) * s.config.PositionBoostThreshold)
	if threshold < 100 {
		threshold = 100 // Minimum threshold
	}

	// Check if any match is in the first portion of the content
	contentLower := strings.ToLower(content)
	earlyContent := contentLower
	if len(earlyContent) > threshold {
		earlyContent = earlyContent[:threshold]
	}

	// Check phrases first
	for _, phrase := range phrases {
		if strings.Contains(earlyContent, strings.ToLower(phrase)) {
			return s.config.PositionBoostMultiplier
		}
	}

	// Check terms
	for _, term := range terms {
		if strings.Contains(earlyContent, term) {
			return s.config.PositionBoostMultiplier
		}
	}

	return 1.0
}

// DetectHeaders extracts headers from content (markdown and HTML).
func DetectHeaders(content string) []HeaderMatch {
	var headers []HeaderMatch

	// Markdown headers: # Header, ## Header, etc.
	mdHeaderRegex := regexp.MustCompile(`(?m)^(#{1,6})\s+(.+)$`)
	mdMatches := mdHeaderRegex.FindAllStringSubmatchIndex(content, -1)
	for _, match := range mdMatches {
		if len(match) >= 6 {
			level := match[3] - match[2] // Length of # characters
			text := content[match[4]:match[5]]
			headers = append(headers, HeaderMatch{
				Level:    level,
				Text:     strings.TrimSpace(text),
				Position: match[0],
			})
		}
	}

	// HTML headers: <h1>Header</h1>, <h2>Header</h2>, etc.
	htmlHeaderRegex := regexp.MustCompile(`(?i)<h([1-6])[^>]*>([^<]+)</h[1-6]>`)
	htmlMatches := htmlHeaderRegex.FindAllStringSubmatch(content, -1)
	for i, match := range htmlMatches {
		if len(match) >= 3 {
			level := int(match[1][0] - '0')
			headers = append(headers, HeaderMatch{
				Level:    level,
				Text:     strings.TrimSpace(match[2]),
				Position: i * 100, // Approximate position
			})
		}
	}

	// RST headers: Header followed by === or ---
	rstHeaderRegex := regexp.MustCompile(`(?m)^(.+)\n[=\-~\^]+\s*$`)
	rstMatches := rstHeaderRegex.FindAllStringSubmatch(content, -1)
	for i, match := range rstMatches {
		if len(match) >= 2 {
			// RST headers don't have explicit levels, estimate by underline character
			level := 1
			headers = append(headers, HeaderMatch{
				Level:    level,
				Text:     strings.TrimSpace(match[1]),
				Position: i * 100, // Approximate position
			})
		}
	}

	return headers
}

// ScoreContentWithDetails returns detailed scoring information.
func (s *ContentScorer) ScoreWithDetails(ctx *ScoringContext) *ContentMatchResult {
	result := &ContentMatchResult{
		Score:     s.Score(ctx),
		MatchType: MatchTypeNone,
	}

	if ctx.Document == nil || ctx.Query == nil {
		return result
	}

	content := ctx.Content
	if content == "" {
		content = ctx.Document.Content
	}
	if content == "" {
		return result
	}

	analyzer := NewQueryAnalyzer()
	tokens := analyzer.TokenizeForMatching(ctx.Query)

	// Determine match type and collect details
	for _, phrase := range ctx.Query.Phrases {
		if FindPhrasePosition(phrase, content) != -1 {
			result.MatchType = MatchTypePhrase
			result.PhraseMatches = append(result.PhraseMatches, phrase)
		}
	}

	if result.MatchType == MatchTypeNone && len(tokens) > 0 {
		matchCount := CountMatchingTerms(tokens, content)
		if matchCount == len(tokens) {
			if TermsInOrder(tokens, content) {
				result.MatchType = MatchTypePhrase
			} else {
				result.MatchType = MatchTypeAllWords
			}
		} else if matchCount > 0 {
			result.MatchType = MatchTypePartial
		}
	}

	// Collect matched terms
	for _, term := range tokens {
		if CountOccurrences(term, content) > 0 {
			result.MatchedTerms = append(result.MatchedTerms, term)
		}
	}

	// Detect header matches
	headers := DetectHeaders(content)
	for _, header := range headers {
		if CountMatchingTerms(tokens, header.Text) > 0 {
			result.HeaderMatches = append(result.HeaderMatches, header.Text)
		}
	}

	return result
}

// ContentMatchResult contains detailed content matching information.
type ContentMatchResult struct {
	Score         float64
	MatchType     MatchType
	MatchedTerms  []string
	PhraseMatches []string
	HeaderMatches []string
	TFIDFScore    float64
}

// CalculateTermFrequency calculates the term frequency (TF) for a term in content.
func CalculateTermFrequency(term, content string) float64 {
	contentLower := strings.ToLower(content)
	termLower := strings.ToLower(term)

	count := strings.Count(contentLower, termLower)
	words := strings.Fields(contentLower)

	if len(words) == 0 {
		return 0
	}

	return float64(count) / float64(len(words))
}

// CalculateTFIDF calculates the TF-IDF score for a term.
func CalculateTFIDF(term, content string, stats *CorpusStats) float64 {
	tf := CalculateTermFrequency(term, content)
	if tf == 0 {
		return 0
	}

	idf := 1.0
	if stats != nil {
		idf = stats.IDF(term)
	}

	return tf * idf
}
