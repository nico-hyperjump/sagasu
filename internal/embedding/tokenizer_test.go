package embedding

import (
	"testing"
)

func TestSimpleTokenizer_Tokenize(t *testing.T) {
	tok := &SimpleTokenizer{}
	ids, attn, _ := tok.Tokenize("hello world", 10)
	if len(ids) != 10 {
		t.Errorf("len(ids)=%d", len(ids))
	}
	if ids[0] != 101 {
		t.Errorf("expected CLS 101, got %d", ids[0])
	}
	if attn[0] != 1 {
		t.Error("attention[0] should be 1")
	}
}

func TestSplitWords(t *testing.T) {
	words := SplitWords("  a  b  c  ")
	if len(words) != 3 {
		t.Errorf("expected 3 words, got %v", words)
	}
	if SplitWords("") != nil {
		t.Error("empty string should return nil")
	}
}

func TestHashString(t *testing.T) {
	h := HashString("abc")
	if h == 0 {
		t.Error("hash should be non-zero")
	}
	if HashString("abc") != HashString("abc") {
		t.Error("hash should be deterministic")
	}
}
