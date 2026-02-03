package indexer

import (
	"testing"
)

func TestChunker_Chunk(t *testing.T) {
	c := NewChunker(3, 1)
	chunks := c.Chunk("doc1", "one two three four five six seven")
	if len(chunks) < 2 {
		t.Errorf("expected at least 2 chunks, got %d", len(chunks))
	}
	for i, ch := range chunks {
		if ch.DocumentID != "doc1" {
			t.Errorf("chunk %d DocumentID=%s", i, ch.DocumentID)
		}
		if ch.ChunkIndex != i {
			t.Errorf("chunk %d ChunkIndex=%d", i, ch.ChunkIndex)
		}
		if ch.ID == "" {
			t.Error("chunk ID should be set")
		}
	}
}

func TestChunker_ChunkEmpty(t *testing.T) {
	c := NewChunker(5, 1)
	chunks := c.Chunk("d", "   \n\t  ")
	if chunks != nil {
		t.Errorf("empty text should return nil, got %v", chunks)
	}
}

func TestPreprocess(t *testing.T) {
	if Preprocess("  a  b  ") != "a b" {
		t.Error("expected trimmed and collapsed spaces")
	}
}
