// Package keyword provides keyword (BM25) search indexing and search.
package keyword

import (
	"context"

	"github.com/hyperjump/sagasu/internal/models"
)

// SearchOptions optional parameters for keyword search. Nil means use defaults.
type SearchOptions struct {
	// TitleBoost multiplies the score contribution from matches in the title (filename) field.
	// Values > 1 make filename matches rank higher (e.g. 3.0). Use 1.0 for no boost.
	TitleBoost float64
	// PhraseBoost multiplies the score when query terms appear close together (phrase match).
	// Values > 1 boost documents with adjacent query terms (e.g. 1.5). Use 1.0 for no boost.
	PhraseBoost float64
	// FuzzyEnabled enables fuzzy matching for typo tolerance.
	// When true, searches will match terms within the specified edit distance.
	FuzzyEnabled bool
	// Fuzziness is the maximum Levenshtein edit distance for fuzzy matching (1 or 2).
	// Default is 2 when FuzzyEnabled is true. Higher values are more lenient.
	Fuzziness int
}

// KeywordIndex defines keyword search operations.
type KeywordIndex interface {
	Index(ctx context.Context, id string, doc *models.Document) error
	Search(ctx context.Context, query string, limit int, opts *SearchOptions) ([]*KeywordResult, error)
	Delete(ctx context.Context, id string) error
	Close() error
	// DocCount returns the total number of documents in the index.
	DocCount() (uint64, error)
	// GetTermDocFrequency returns the number of documents containing the term.
	GetTermDocFrequency(term string) (int, error)
	// GetCorpusStats returns total doc count and doc frequencies for terms.
	GetCorpusStats(terms []string) (totalDocs int, docFreqs map[string]int, err error)
}

// KeywordResult is a single keyword search hit.
type KeywordResult struct {
	ID    string
	Score float64
}

// TermDictionary provides access to the term dictionary for spell checking.
// This interface allows dependency injection for testing.
type TermDictionary interface {
	// GetAllTerms returns all unique terms in the index.
	GetAllTerms() ([]string, error)
	// GetTermFrequency returns the document frequency for a term.
	GetTermFrequency(term string) (int, error)
	// ContainsTerm checks if a term exists in the index.
	ContainsTerm(term string) (bool, error)
}
