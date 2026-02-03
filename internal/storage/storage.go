// Package storage defines the persistence interface for documents and chunks.
package storage

import (
	"context"

	"github.com/hyperjump/sagasu/internal/models"
)

// Storage defines document and chunk persistence operations.
type Storage interface {
	// Document operations
	CreateDocument(ctx context.Context, doc *models.Document) error
	GetDocument(ctx context.Context, id string) (*models.Document, error)
	UpdateDocument(ctx context.Context, doc *models.Document) error
	DeleteDocument(ctx context.Context, id string) error
	ListDocuments(ctx context.Context, offset, limit int) ([]*models.Document, error)

	// Chunk operations
	CreateChunk(ctx context.Context, chunk *models.DocumentChunk) error
	GetChunksByDocumentID(ctx context.Context, docID string) ([]*models.DocumentChunk, error)
	GetChunk(ctx context.Context, id string) (*models.DocumentChunk, error)
	DeleteChunksByDocumentID(ctx context.Context, docID string) error

	// Batch operations
	BatchCreateChunks(ctx context.Context, chunks []*models.DocumentChunk) error

	// Stats
	CountDocuments(ctx context.Context) (int64, error)
	CountChunks(ctx context.Context) (int64, error)

	Close() error
}
