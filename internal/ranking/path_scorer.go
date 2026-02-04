package ranking

import (
	"path/filepath"
	"strings"
)

// PathScorer scores documents based on path component matching.
type PathScorer struct {
	config *RankingConfig
}

// NewPathScorer creates a new PathScorer with the given config.
func NewPathScorer(config *RankingConfig) *PathScorer {
	return &PathScorer{config: config}
}

// Name returns the scorer name.
func (s *PathScorer) Name() string {
	return "path"
}

// Score calculates the path match score.
func (s *PathScorer) Score(ctx *ScoringContext) float64 {
	if ctx.Query == nil {
		return 0
	}

	// Get path from context or metadata
	path := ctx.FilePath
	if path == "" && ctx.Document != nil && ctx.Document.Metadata != nil {
		if p, ok := ctx.Document.Metadata["source_path"].(string); ok {
			path = p
		}
	}
	if path == "" {
		return 0
	}

	// Extract path components (directory names only, not filename)
	components := ExtractPathComponents(path)
	if len(components) == 0 {
		return 0
	}

	analyzer := NewQueryAnalyzer()
	tokens := analyzer.TokenizeForMatching(ctx.Query)
	if len(tokens) == 0 {
		return 0
	}

	return s.scorePathComponents(tokens, components)
}

// scorePathComponents scores based on term matches in path components.
func (s *PathScorer) scorePathComponents(terms []string, components []string) float64 {
	if len(terms) == 0 || len(components) == 0 {
		return 0
	}

	totalScore := 0.0
	matchedComponents := 0

	for _, component := range components {
		componentLower := strings.ToLower(component)
		componentScore := 0.0

		for _, term := range terms {
			termLower := strings.ToLower(term)

			// Exact match on component
			if componentLower == termLower {
				componentScore = max(componentScore, s.config.PathExactMatchScore)
				continue
			}

			// Component contains term as substring
			if strings.Contains(componentLower, termLower) {
				// Score based on how much of the component is matched
				ratio := float64(len(termLower)) / float64(len(componentLower))
				partialScore := s.config.PathPartialMatchScore + (s.config.PathExactMatchScore-s.config.PathPartialMatchScore)*ratio
				componentScore = max(componentScore, partialScore)
			}

			// Term is prefix of component
			if strings.HasPrefix(componentLower, termLower) {
				prefixScore := s.config.PathPartialMatchScore * 1.2
				componentScore = max(componentScore, prefixScore)
			}
		}

		if componentScore > 0 {
			totalScore += componentScore
			matchedComponents++
		}
	}

	// Add bonus for matching multiple components
	if matchedComponents > 1 {
		totalScore += s.config.PathComponentBonus * float64(matchedComponents-1)
	}

	return totalScore
}

// ExtractPathComponents extracts directory names from a file path.
// Returns directory names only (not the filename).
func ExtractPathComponents(path string) []string {
	// Clean the path
	path = filepath.Clean(path)

	// Get directory path (without filename)
	dir := filepath.Dir(path)

	// Split into components
	var components []string
	for dir != "" && dir != "/" && dir != "." {
		base := filepath.Base(dir)
		if base != "" && base != "/" && base != "." {
			components = append([]string{base}, components...)
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	return components
}

// ScoreWithDetails returns detailed path scoring information.
func (s *PathScorer) ScoreWithDetails(ctx *ScoringContext) *PathMatchResult {
	result := &PathMatchResult{
		Score: s.Score(ctx),
	}

	if ctx.Query == nil {
		return result
	}

	path := ctx.FilePath
	if path == "" && ctx.Document != nil && ctx.Document.Metadata != nil {
		if p, ok := ctx.Document.Metadata["source_path"].(string); ok {
			path = p
		}
	}
	if path == "" {
		return result
	}

	components := ExtractPathComponents(path)
	analyzer := NewQueryAnalyzer()
	tokens := analyzer.TokenizeForMatching(ctx.Query)

	// Find matching components
	for _, component := range components {
		componentLower := strings.ToLower(component)
		for _, term := range tokens {
			if strings.Contains(componentLower, strings.ToLower(term)) {
				result.MatchedComponents = append(result.MatchedComponents, component)
				result.MatchedTerms = append(result.MatchedTerms, term)
				break
			}
		}
	}

	return result
}

// PathMatchResult contains detailed path matching information.
type PathMatchResult struct {
	Score             float64
	MatchedComponents []string
	MatchedTerms      []string
}

// NormalizePath normalizes a path for matching (lowercase, consistent separators).
func NormalizePath(path string) string {
	// Convert to forward slashes
	path = filepath.ToSlash(path)
	// Lowercase
	path = strings.ToLower(path)
	// Remove trailing slash
	path = strings.TrimSuffix(path, "/")
	return path
}

// PathContainsComponent checks if a path contains a specific directory component.
func PathContainsComponent(path, component string) bool {
	components := ExtractPathComponents(path)
	componentLower := strings.ToLower(component)

	for _, c := range components {
		if strings.ToLower(c) == componentLower {
			return true
		}
	}
	return false
}

// PathMatchesPattern checks if a path matches a glob-like pattern.
// Supports * for any characters within a component.
func PathMatchesPattern(path, pattern string) bool {
	// Simple pattern matching - just check if path contains pattern
	// For proper glob matching, use filepath.Match
	patternLower := strings.ToLower(pattern)
	pathLower := strings.ToLower(path)

	// Handle wildcards by converting to simpler contains check
	if strings.Contains(patternLower, "*") {
		parts := strings.Split(patternLower, "*")
		lastIdx := 0
		for _, part := range parts {
			if part == "" {
				continue
			}
			idx := strings.Index(pathLower[lastIdx:], part)
			if idx == -1 {
				return false
			}
			lastIdx = lastIdx + idx + len(part)
		}
		return true
	}

	return strings.Contains(pathLower, patternLower)
}
