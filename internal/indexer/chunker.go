// Package indexer provides document chunking and indexing.
package indexer

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/hyperjump/sagasu/internal/models"
)

// Chunker splits text into overlapping word-based chunks.
type Chunker struct {
	chunkSize    int
	chunkOverlap int
}

// NewChunker creates a chunker with the given size and overlap (in words).
func NewChunker(chunkSize, chunkOverlap int) *Chunker {
	return &Chunker{
		chunkSize:    chunkSize,
		chunkOverlap: chunkOverlap,
	}
}

// Chunk splits text into DocumentChunks with overlapping windows.
func (c *Chunker) Chunk(docID, text string) []*models.DocumentChunk {
	words := strings.Fields(text)
	if len(words) == 0 {
		return nil
	}
	chunks := make([]*models.DocumentChunk, 0)
	chunkIndex := 0
	step := c.chunkSize - c.chunkOverlap
	if step <= 0 {
		step = 1
	}
	for i := 0; i < len(words); i += step {
		end := i + c.chunkSize
		if end > len(words) {
			end = len(words)
		}
		chunkWords := words[i:end]
		chunkText := strings.Join(chunkWords, " ")
		chunk := &models.DocumentChunk{
			ID:         fmt.Sprintf("%s_%s", docID, uuid.New().String()[:8]),
			DocumentID: docID,
			Content:    chunkText,
			ChunkIndex: chunkIndex,
		}
		chunks = append(chunks, chunk)
		chunkIndex++
		if end >= len(words) {
			break
		}
	}
	return chunks
}
