package embedding

import "strings"

// Tokenizer produces token IDs for BERT-style models (input_ids, attention_mask, token_type_ids).
type Tokenizer interface {
	Tokenize(text string, maxTokens int) (inputIDs, attentionMask, tokenTypeIDs []int64)
}

// SimpleTokenizer is a word-split tokenizer with hash-based token IDs (for testing or fallback).
type SimpleTokenizer struct{}

// Tokenize splits text into words and produces padded token IDs up to maxTokens.
func (t *SimpleTokenizer) Tokenize(text string, maxTokens int) (inputIDs, attentionMask, tokenTypeIDs []int64) {
	words := SplitWords(text)
	if maxTokens <= 0 {
		maxTokens = 256
	}
	inputIDs = make([]int64, maxTokens)
	attentionMask = make([]int64, maxTokens)
	tokenTypeIDs = make([]int64, maxTokens)

	inputIDs[0] = 101 // [CLS]
	attentionMask[0] = 1

	pos := 1
	for _, word := range words {
		if pos >= maxTokens-1 {
			break
		}
		inputIDs[pos] = int64(HashString(word) % 30000)
		attentionMask[pos] = 1
		pos++
	}
	if pos < maxTokens {
		inputIDs[pos] = 102 // [SEP]
		attentionMask[pos] = 1
	}
	return inputIDs, attentionMask, tokenTypeIDs
}

// SplitWords splits text on whitespace and returns non-empty words.
func SplitWords(text string) []string {
	var words []string
	word := ""
	for _, r := range text {
		if r == ' ' || r == '\n' || r == '\t' {
			if word != "" {
				words = append(words, word)
				word = ""
			}
		} else {
			word += string(r)
		}
	}
	if word != "" {
		words = append(words, word)
	}
	return words
}

// HashString returns a deterministic hash for use as a simple token ID.
func HashString(s string) int {
	h := 0
	for _, c := range s {
		h = 31*h + int(c)
	}
	if h < 0 {
		h = -h
	}
	return h
}

// TruncateWords returns up to maxWords words from the slice.
func TruncateWords(words []string, maxWords int) []string {
	if len(words) <= maxWords {
		return words
	}
	return words[:maxWords]
}

// JoinWords joins words with a space.
func JoinWords(words []string) string {
	return strings.Join(words, " ")
}
