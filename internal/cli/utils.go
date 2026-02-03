// Package cli provides CLI utilities for Sagasu.
package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/hyperjump/sagasu/internal/models"
)

// SearchOutputFormat is the format for search result output.
type SearchOutputFormat string

const (
	// OutputText is human-readable text (default).
	OutputText SearchOutputFormat = "text"
	// OutputJSON is structured JSON for machine consumption.
	OutputJSON SearchOutputFormat = "json"
)

// WriteSearchResults writes search results to w in the given format.
// Use OutputJSON for parseable output consumable by other apps.
func WriteSearchResults(w io.Writer, response *models.SearchResponse, format SearchOutputFormat) error {
	switch format {
	case OutputJSON:
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		return enc.Encode(response)
	default:
		writeSearchResultsText(w, response)
		return nil
	}
}

func writeSearchResultsText(w io.Writer, response *models.SearchResponse) {
	total := response.TotalNonSemantic + response.TotalSemantic
	fmt.Fprintf(w, "\nFound %d results in %dms (%d keyword-only, %d semantic-only)\n\n",
		total, response.QueryTime, response.TotalNonSemantic, response.TotalSemantic)
	if len(response.NonSemanticResults) > 0 {
		fmt.Fprintln(w, "--- Non-semantic (keyword) results ---")
		for _, result := range response.NonSemanticResults {
			writeOneResult(w, result, "keyword")
		}
	}
	if len(response.SemanticResults) > 0 {
		fmt.Fprintln(w, "--- Semantic results ---")
		for _, result := range response.SemanticResults {
			writeOneResult(w, result, "semantic")
		}
	}
}

func writeOneResult(w io.Writer, result *models.SearchResult, source string) {
	fmt.Fprintf(w, "─────────────────────────────────────────────────────────\n")
	fmt.Fprintf(w, "[%s] Rank: %d | Score: %.4f (Keyword: %.4f, Semantic: %.4f)\n",
		source, result.Rank, result.Score, result.KeywordScore, result.SemanticScore)
	fmt.Fprintf(w, "ID: %s\n", result.Document.ID)
	if result.Document.Title != "" {
		fmt.Fprintf(w, "Title: %s\n", result.Document.Title)
	}
	fmt.Fprintf(w, "\n%s\n", Truncate(result.Document.Content, 200))
	fmt.Fprintln(w)
}

// PrintSearchResults prints search results to stdout in text format (backward compatible).
func PrintSearchResults(response *models.SearchResponse) {
	_ = WriteSearchResults(os.Stdout, response, OutputText)
}

// Truncate truncates s to maxLen and appends "..." if truncated.
func Truncate(s string, maxLen int) string {
	if maxLen <= 0 || len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// TruncateWords returns up to maxWords from the space-separated string.
func TruncateWords(s string, maxWords int) string {
	words := strings.Fields(s)
	if len(words) <= maxWords {
		return s
	}
	return strings.Join(words[:maxWords], " ") + "..."
}
