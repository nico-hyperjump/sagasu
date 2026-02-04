// Package ranking provides content-aware ranking for search results.
package ranking

import (
	"time"

	"github.com/hyperjump/sagasu/internal/models"
)

// MatchType represents the type of query match found.
type MatchType int

const (
	// MatchTypeNone indicates no match was found.
	MatchTypeNone MatchType = iota
	// MatchTypePartial indicates a partial match (some query terms matched).
	MatchTypePartial
	// MatchTypeAllWords indicates all query words matched but not as a phrase.
	MatchTypeAllWords
	// MatchTypePhrase indicates an exact phrase match.
	MatchTypePhrase
	// MatchTypeExact indicates an exact match (e.g., filename without extension).
	MatchTypeExact
)

// String returns a string representation of the match type.
func (m MatchType) String() string {
	switch m {
	case MatchTypeNone:
		return "none"
	case MatchTypePartial:
		return "partial"
	case MatchTypeAllWords:
		return "all_words"
	case MatchTypePhrase:
		return "phrase"
	case MatchTypeExact:
		return "exact"
	default:
		return "unknown"
	}
}

// QueryType represents the type of search query.
type QueryType int

const (
	// QueryTypeSingleWord is a single word query.
	QueryTypeSingleWord QueryType = iota
	// QueryTypeMultiWord is a multi-word query without quotes.
	QueryTypeMultiWord
	// QueryTypePhrase is a quoted exact phrase query.
	QueryTypePhrase
	// QueryTypeWildcard is a query containing wildcards (* or ?).
	QueryTypeWildcard
	// QueryTypeBoolean is a query with boolean operators (AND/OR/NOT).
	QueryTypeBoolean
)

// String returns a string representation of the query type.
func (q QueryType) String() string {
	switch q {
	case QueryTypeSingleWord:
		return "single_word"
	case QueryTypeMultiWord:
		return "multi_word"
	case QueryTypePhrase:
		return "phrase"
	case QueryTypeWildcard:
		return "wildcard"
	case QueryTypeBoolean:
		return "boolean"
	default:
		return "unknown"
	}
}

// AnalyzedQuery holds the parsed and analyzed form of a search query.
type AnalyzedQuery struct {
	// Original is the original query string.
	Original string
	// Terms are the individual normalized tokens from the query.
	Terms []string
	// Phrases are exact phrase matches extracted from quoted strings.
	Phrases []string
	// QueryType is the classified type of the query.
	QueryType QueryType
	// HasWildcard indicates if the query contains wildcard characters.
	HasWildcard bool
	// NegatedTerms are terms that should be excluded (NOT operator).
	NegatedTerms []string
}

// CorpusStats holds corpus-level statistics for IDF calculation.
type CorpusStats struct {
	// TotalDocs is the total number of documents in the corpus.
	TotalDocs int
	// DocFrequencies maps terms to the number of documents containing them.
	DocFrequencies map[string]int
}

// NewCorpusStats creates a new CorpusStats instance.
func NewCorpusStats() *CorpusStats {
	return &CorpusStats{
		TotalDocs:      0,
		DocFrequencies: make(map[string]int),
	}
}

// IDF calculates the Inverse Document Frequency for a term.
// Returns a higher value for rare terms and lower value for common terms.
func (c *CorpusStats) IDF(term string) float64 {
	if c.TotalDocs == 0 {
		return 1.0
	}
	docFreq := c.DocFrequencies[term]
	if docFreq == 0 {
		// Term not in corpus, treat as very rare
		return 1.0 + float64(c.TotalDocs)
	}
	// Standard IDF formula: log(N/df) + 1
	// Adding 1 to avoid zero for terms appearing in all docs
	return 1.0 + float64(c.TotalDocs)/float64(docFreq)
}

// HeaderMatch represents a match found in a document header.
type HeaderMatch struct {
	// Level is the header level (1 for h1, 2 for h2, etc.).
	Level int
	// Text is the header text content.
	Text string
	// Position is the byte offset in the content.
	Position int
}

// ScoringContext provides all the context needed for scoring a document.
type ScoringContext struct {
	// Query is the analyzed query.
	Query *AnalyzedQuery
	// Document is the document being scored.
	Document *models.Document
	// Content is the document content (may be empty if not needed).
	Content string
	// CorpusStats provides corpus-level statistics for IDF.
	CorpusStats *CorpusStats
	// FilePath is the source file path (from metadata).
	FilePath string
	// FileSize is the file size in bytes.
	FileSize int64
	// ModTime is the file modification time.
	ModTime time.Time
}

// NewScoringContext creates a ScoringContext from a query and document.
func NewScoringContext(query *AnalyzedQuery, doc *models.Document, stats *CorpusStats) *ScoringContext {
	ctx := &ScoringContext{
		Query:       query,
		Document:    doc,
		Content:     doc.Content,
		CorpusStats: stats,
	}

	// Extract metadata fields
	if doc.Metadata != nil {
		if path, ok := doc.Metadata["source_path"].(string); ok {
			ctx.FilePath = path
		}
		if sizeStr, ok := doc.Metadata["source_size"].(string); ok {
			var size int64
			if _, err := parseMetadataInt64(sizeStr, &size); err == nil {
				ctx.FileSize = size
			}
		}
		if mtimeStr, ok := doc.Metadata["source_mtime"].(string); ok {
			var mtime int64
			if _, err := parseMetadataInt64(mtimeStr, &mtime); err == nil {
				ctx.ModTime = time.Unix(0, mtime)
			}
		}
	}

	return ctx
}

// parseMetadataInt64 parses a string to int64, handling various formats.
func parseMetadataInt64(s string, result *int64) (bool, error) {
	var n int64
	for _, c := range s {
		if c < '0' || c > '9' {
			return false, nil
		}
		n = n*10 + int64(c-'0')
	}
	*result = n
	return true, nil
}

// Scorer is the interface for all scoring components.
type Scorer interface {
	// Score calculates the score for a document given the scoring context.
	Score(ctx *ScoringContext) float64
	// Name returns the name of the scorer for debugging/logging.
	Name() string
}

// Multiplier is the interface for score multipliers.
type Multiplier interface {
	// Multiply applies a multiplier to the base score.
	Multiply(ctx *ScoringContext, baseScore float64) float64
	// Name returns the name of the multiplier for debugging/logging.
	Name() string
}

// ScoreBreakdown provides detailed scoring information for debugging.
type ScoreBreakdown struct {
	// FinalScore is the computed final score.
	FinalScore float64
	// FilenameScore is the score from filename matching.
	FilenameScore float64
	// ContentScore is the score from content matching.
	ContentScore float64
	// PathScore is the score from path matching.
	PathScore float64
	// MetadataScore is the score from metadata matching.
	MetadataScore float64
	// Multipliers holds the applied multiplier values.
	Multipliers map[string]float64
	// MatchType is the best match type found.
	MatchType MatchType
}

// NewScoreBreakdown creates a new ScoreBreakdown instance.
func NewScoreBreakdown() *ScoreBreakdown {
	return &ScoreBreakdown{
		Multipliers: make(map[string]float64),
	}
}
