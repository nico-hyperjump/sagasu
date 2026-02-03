// Package models defines core data structures for documents, queries, and search results.
package models

import "time"

// Document represents a stored document with metadata.
type Document struct {
	ID        string                 `json:"id" db:"id"`
	Title     string                 `json:"title" db:"title"`
	Content   string                 `json:"content" db:"content"`
	Metadata  map[string]interface{} `json:"metadata" db:"metadata"`
	CreatedAt time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt time.Time              `json:"updated_at" db:"updated_at"`
}

// DocumentChunk represents a chunk of a document, used for semantic indexing.
type DocumentChunk struct {
	ID         string    `json:"id" db:"id"`
	DocumentID string    `json:"document_id" db:"document_id"`
	Content    string    `json:"content" db:"content"`
	ChunkIndex int       `json:"chunk_index" db:"chunk_index"`
	Embedding  []float32 `json:"-" db:"-"`
	CreatedAt  time.Time `json:"created_at" db:"created_at"`
}

// DocumentInput is the input for creating or updating a document.
type DocumentInput struct {
	ID       string                 `json:"id,omitempty"`
	Title    string                 `json:"title,omitempty"`
	Content  string                 `json:"content"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}
