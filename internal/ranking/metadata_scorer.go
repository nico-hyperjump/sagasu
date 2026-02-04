package ranking

import (
	"fmt"
	"strings"
)

// MetadataScorer scores documents based on metadata field matching.
type MetadataScorer struct {
	config *RankingConfig
}

// NewMetadataScorer creates a new MetadataScorer with the given config.
func NewMetadataScorer(config *RankingConfig) *MetadataScorer {
	return &MetadataScorer{config: config}
}

// Name returns the scorer name.
func (s *MetadataScorer) Name() string {
	return "metadata"
}

// Score calculates the metadata match score.
func (s *MetadataScorer) Score(ctx *ScoringContext) float64 {
	if ctx.Query == nil || ctx.Document == nil {
		return 0
	}

	metadata := ctx.Document.Metadata
	if metadata == nil || len(metadata) == 0 {
		return 0
	}

	analyzer := NewQueryAnalyzer()
	tokens := analyzer.TokenizeForMatching(ctx.Query)
	if len(tokens) == 0 {
		return 0
	}

	totalScore := 0.0

	// Check each metadata field
	for key, value := range metadata {
		// Skip internal metadata keys
		if isInternalMetadataKey(key) {
			continue
		}

		valueStr := metadataValueToString(value)
		if valueStr == "" {
			continue
		}

		fieldScore := s.scoreMetadataField(key, valueStr, tokens)
		totalScore += fieldScore
	}

	return totalScore
}

// scoreMetadataField scores a single metadata field.
func (s *MetadataScorer) scoreMetadataField(key, value string, terms []string) float64 {
	valueLower := strings.ToLower(value)
	matchCount := 0

	for _, term := range terms {
		if strings.Contains(valueLower, strings.ToLower(term)) {
			matchCount++
		}
	}

	if matchCount == 0 {
		return 0
	}

	// Determine base score based on field type
	baseScore := s.getFieldBaseScore(key)

	// Scale by match ratio
	matchRatio := float64(matchCount) / float64(len(terms))

	return baseScore * matchRatio
}

// getFieldBaseScore returns the base score for a metadata field type.
func (s *MetadataScorer) getFieldBaseScore(key string) float64 {
	keyLower := strings.ToLower(key)

	// Author-related fields
	if keyLower == "author" || keyLower == "creator" || keyLower == "by" || keyLower == "created_by" {
		return s.config.AuthorMatchScore
	}

	// Tag-related fields
	if keyLower == "tags" || keyLower == "tag" || keyLower == "keywords" || keyLower == "categories" || keyLower == "category" {
		return s.config.TagMatchScore
	}

	// Default for other fields
	return s.config.OtherMetadataScore
}

// isInternalMetadataKey checks if a metadata key is internal (not for user matching).
func isInternalMetadataKey(key string) bool {
	internalKeys := map[string]bool{
		"source_path":  true,
		"source_mtime": true,
		"source_size":  true,
	}
	return internalKeys[key]
}

// metadataValueToString converts a metadata value to a string for matching.
func metadataValueToString(value interface{}) string {
	if value == nil {
		return ""
	}

	switch v := value.(type) {
	case string:
		return v
	case []string:
		return strings.Join(v, " ")
	case []interface{}:
		parts := make([]string, 0, len(v))
		for _, item := range v {
			if s, ok := item.(string); ok {
				parts = append(parts, s)
			} else {
				parts = append(parts, fmt.Sprintf("%v", item))
			}
		}
		return strings.Join(parts, " ")
	default:
		return fmt.Sprintf("%v", v)
	}
}

// ScoreWithDetails returns detailed metadata scoring information.
func (s *MetadataScorer) ScoreWithDetails(ctx *ScoringContext) *MetadataMatchResult {
	result := &MetadataMatchResult{
		Score:         s.Score(ctx),
		MatchedFields: make(map[string][]string),
	}

	if ctx.Query == nil || ctx.Document == nil {
		return result
	}

	metadata := ctx.Document.Metadata
	if metadata == nil || len(metadata) == 0 {
		return result
	}

	analyzer := NewQueryAnalyzer()
	tokens := analyzer.TokenizeForMatching(ctx.Query)
	if len(tokens) == 0 {
		return result
	}

	// Find matched fields and terms
	for key, value := range metadata {
		if isInternalMetadataKey(key) {
			continue
		}

		valueStr := metadataValueToString(value)
		if valueStr == "" {
			continue
		}

		valueLower := strings.ToLower(valueStr)
		var matchedTerms []string

		for _, term := range tokens {
			if strings.Contains(valueLower, strings.ToLower(term)) {
				matchedTerms = append(matchedTerms, term)
			}
		}

		if len(matchedTerms) > 0 {
			result.MatchedFields[key] = matchedTerms
		}
	}

	return result
}

// MetadataMatchResult contains detailed metadata matching information.
type MetadataMatchResult struct {
	Score         float64
	MatchedFields map[string][]string // field name -> matched terms
}

// ExtractMetadataText extracts all searchable text from metadata.
func ExtractMetadataText(metadata map[string]interface{}) string {
	if metadata == nil {
		return ""
	}

	var parts []string
	for key, value := range metadata {
		if isInternalMetadataKey(key) {
			continue
		}

		valueStr := metadataValueToString(value)
		if valueStr != "" {
			parts = append(parts, valueStr)
		}
	}

	return strings.Join(parts, " ")
}

// GetMetadataField retrieves a metadata field value as a string.
func GetMetadataField(metadata map[string]interface{}, key string) string {
	if metadata == nil {
		return ""
	}

	value, ok := metadata[key]
	if !ok {
		return ""
	}

	return metadataValueToString(value)
}

// GetMetadataFieldSlice retrieves a metadata field value as a string slice.
func GetMetadataFieldSlice(metadata map[string]interface{}, key string) []string {
	if metadata == nil {
		return nil
	}

	value, ok := metadata[key]
	if !ok {
		return nil
	}

	switch v := value.(type) {
	case []string:
		return v
	case []interface{}:
		result := make([]string, 0, len(v))
		for _, item := range v {
			if s, ok := item.(string); ok {
				result = append(result, s)
			} else {
				result = append(result, fmt.Sprintf("%v", item))
			}
		}
		return result
	case string:
		// Split by common delimiters
		return strings.FieldsFunc(v, func(r rune) bool {
			return r == ',' || r == ';' || r == '|'
		})
	default:
		return nil
	}
}

// MatchMetadataField checks if a query matches a specific metadata field.
func MatchMetadataField(query *AnalyzedQuery, metadata map[string]interface{}, fieldName string) bool {
	value := GetMetadataField(metadata, fieldName)
	if value == "" {
		return false
	}

	analyzer := NewQueryAnalyzer()
	tokens := analyzer.TokenizeForMatching(query)

	valueLower := strings.ToLower(value)
	for _, term := range tokens {
		if strings.Contains(valueLower, strings.ToLower(term)) {
			return true
		}
	}

	return false
}
