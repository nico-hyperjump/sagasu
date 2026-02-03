// Package storage provides SQLite implementation of the Storage interface.
package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "github.com/mattn/go-sqlite3"

	"github.com/hyperjump/sagasu/internal/models"
)

// SQLiteStorage implements Storage using SQLite.
type SQLiteStorage struct {
	db *sql.DB
}

// NewSQLiteStorage opens or creates a SQLite database at dbPath and initializes the schema.
// Parent directories are created if they do not exist.
func NewSQLiteStorage(dbPath string) (*SQLiteStorage, error) {
	if dir := filepath.Dir(dbPath); dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create database directory: %w", err)
		}
	}
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("failed to enable WAL: %w", err)
	}

	if err := initSchema(db); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	return &SQLiteStorage{db: db}, nil
}

func initSchema(db *sql.DB) error {
	schema := `
	CREATE TABLE IF NOT EXISTS documents (
		id TEXT PRIMARY KEY,
		title TEXT,
		content TEXT NOT NULL,
		metadata TEXT,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);

	CREATE INDEX IF NOT EXISTS idx_documents_created_at ON documents(created_at);

	CREATE TABLE IF NOT EXISTS document_chunks (
		id TEXT PRIMARY KEY,
		document_id TEXT NOT NULL,
		content TEXT NOT NULL,
		chunk_index INTEGER NOT NULL,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (document_id) REFERENCES documents(id) ON DELETE CASCADE
	);

	CREATE INDEX IF NOT EXISTS idx_chunks_document_id ON document_chunks(document_id);
	CREATE INDEX IF NOT EXISTS idx_chunks_document_chunk ON document_chunks(document_id, chunk_index);
	`
	_, err := db.Exec(schema)
	return err
}

// CreateDocument inserts a document.
func (s *SQLiteStorage) CreateDocument(ctx context.Context, doc *models.Document) error {
	metadataJSON, err := json.Marshal(doc.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	now := time.Now()
	doc.CreatedAt = now
	doc.UpdatedAt = now

	_, err = s.db.ExecContext(ctx,
		`INSERT INTO documents (id, title, content, metadata, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		doc.ID, doc.Title, doc.Content, string(metadataJSON), doc.CreatedAt, doc.UpdatedAt,
	)
	return err
}

// GetDocument returns a document by ID.
func (s *SQLiteStorage) GetDocument(ctx context.Context, id string) (*models.Document, error) {
	var doc models.Document
	var metadataJSON string

	err := s.db.QueryRowContext(ctx,
		`SELECT id, title, content, metadata, created_at, updated_at
		 FROM documents WHERE id = ?`, id,
	).Scan(&doc.ID, &doc.Title, &doc.Content, &metadataJSON, &doc.CreatedAt, &doc.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("document not found: %s", id)
	}
	if err != nil {
		return nil, err
	}

	if metadataJSON != "" {
		if err := json.Unmarshal([]byte(metadataJSON), &doc.Metadata); err != nil {
			return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
		}
	}

	return &doc, nil
}

// UpdateDocument updates an existing document.
func (s *SQLiteStorage) UpdateDocument(ctx context.Context, doc *models.Document) error {
	metadataJSON, err := json.Marshal(doc.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	doc.UpdatedAt = time.Now()

	result, err := s.db.ExecContext(ctx,
		`UPDATE documents SET title = ?, content = ?, metadata = ?, updated_at = ?
		 WHERE id = ?`,
		doc.Title, doc.Content, string(metadataJSON), doc.UpdatedAt, doc.ID,
	)
	if err != nil {
		return err
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return fmt.Errorf("document not found: %s", doc.ID)
	}
	return nil
}

// DeleteDocument removes a document by ID.
func (s *SQLiteStorage) DeleteDocument(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM documents WHERE id = ?`, id)
	return err
}

// ListDocuments returns documents with offset and limit.
func (s *SQLiteStorage) ListDocuments(ctx context.Context, offset, limit int) ([]*models.Document, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, title, content, metadata, created_at, updated_at
		 FROM documents ORDER BY created_at DESC LIMIT ? OFFSET ?`,
		limit, offset,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var docs []*models.Document
	for rows.Next() {
		var doc models.Document
		var metadataJSON string
		if err := rows.Scan(&doc.ID, &doc.Title, &doc.Content, &metadataJSON, &doc.CreatedAt, &doc.UpdatedAt); err != nil {
			return nil, err
		}
		if metadataJSON != "" {
			_ = json.Unmarshal([]byte(metadataJSON), &doc.Metadata)
		}
		docs = append(docs, &doc)
	}
	return docs, rows.Err()
}

// CreateChunk inserts a single chunk.
func (s *SQLiteStorage) CreateChunk(ctx context.Context, chunk *models.DocumentChunk) error {
	chunk.CreatedAt = time.Now()
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO document_chunks (id, document_id, content, chunk_index, created_at)
		 VALUES (?, ?, ?, ?, ?)`,
		chunk.ID, chunk.DocumentID, chunk.Content, chunk.ChunkIndex, chunk.CreatedAt,
	)
	return err
}

// GetChunk returns a chunk by ID.
func (s *SQLiteStorage) GetChunk(ctx context.Context, id string) (*models.DocumentChunk, error) {
	var chunk models.DocumentChunk
	err := s.db.QueryRowContext(ctx,
		`SELECT id, document_id, content, chunk_index, created_at
		 FROM document_chunks WHERE id = ?`, id,
	).Scan(&chunk.ID, &chunk.DocumentID, &chunk.Content, &chunk.ChunkIndex, &chunk.CreatedAt)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("chunk not found: %s", id)
	}
	if err != nil {
		return nil, err
	}
	return &chunk, nil
}

// GetChunksByDocumentID returns all chunks for a document ordered by chunk_index.
func (s *SQLiteStorage) GetChunksByDocumentID(ctx context.Context, docID string) ([]*models.DocumentChunk, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, document_id, content, chunk_index, created_at
		 FROM document_chunks WHERE document_id = ? ORDER BY chunk_index`,
		docID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var chunks []*models.DocumentChunk
	for rows.Next() {
		var chunk models.DocumentChunk
		if err := rows.Scan(&chunk.ID, &chunk.DocumentID, &chunk.Content, &chunk.ChunkIndex, &chunk.CreatedAt); err != nil {
			return nil, err
		}
		chunks = append(chunks, &chunk)
	}
	return chunks, rows.Err()
}

// DeleteChunksByDocumentID removes all chunks for a document.
func (s *SQLiteStorage) DeleteChunksByDocumentID(ctx context.Context, docID string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM document_chunks WHERE document_id = ?`, docID)
	return err
}

// BatchCreateChunks inserts multiple chunks in a transaction.
func (s *SQLiteStorage) BatchCreateChunks(ctx context.Context, chunks []*models.DocumentChunk) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx,
		`INSERT INTO document_chunks (id, document_id, content, chunk_index, created_at)
		 VALUES (?, ?, ?, ?, ?)`,
	)
	if err != nil {
		return err
	}
	defer stmt.Close()

	now := time.Now()
	for _, chunk := range chunks {
		chunk.CreatedAt = now
		if _, err := stmt.ExecContext(ctx, chunk.ID, chunk.DocumentID, chunk.Content, chunk.ChunkIndex, chunk.CreatedAt); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// CountDocuments returns the total number of documents.
func (s *SQLiteStorage) CountDocuments(ctx context.Context) (int64, error) {
	var count int64
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM documents`).Scan(&count)
	return count, err
}

// CountChunks returns the total number of chunks.
func (s *SQLiteStorage) CountChunks(ctx context.Context) (int64, error) {
	var count int64
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM document_chunks`).Scan(&count)
	return count, err
}

// Close closes the database connection.
func (s *SQLiteStorage) Close() error {
	return s.db.Close()
}
