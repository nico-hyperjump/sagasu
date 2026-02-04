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
}

// KeywordIndex defines keyword search operations.
type KeywordIndex interface {
	Index(ctx context.Context, id string, doc *models.Document) error
	Search(ctx context.Context, query string, limit int, opts *SearchOptions) ([]*KeywordResult, error)
	Delete(ctx context.Context, id string) error
	Close() error
}

// KeywordResult is a single keyword search hit.
type KeywordResult struct {
	ID    string
	Score float64
}
