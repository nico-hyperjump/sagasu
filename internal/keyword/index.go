// Package keyword provides keyword (BM25) search indexing and search.
package keyword

import (
	"context"

	"github.com/hyperjump/sagasu/internal/models"
)

// KeywordIndex defines keyword search operations.
type KeywordIndex interface {
	Index(ctx context.Context, id string, doc *models.Document) error
	Search(ctx context.Context, query string, limit int) ([]*KeywordResult, error)
	Delete(ctx context.Context, id string) error
	Close() error
}

// KeywordResult is a single keyword search hit.
type KeywordResult struct {
	ID    string
	Score float64
}
