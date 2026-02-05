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
	// OutputCompact is one result per line (compact text).
	OutputCompact SearchOutputFormat = "compact"
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
	case OutputCompact:
		writeSearchResultsCompact(w, response)
		return nil
	default:
		writeSearchResultsText(w, response)
		return nil
	}
}

func writeSearchResultsText(w io.Writer, response *models.SearchResponse) {
	total := response.TotalNonSemantic + response.TotalSemantic
	fmt.Fprintf(w, "\nFound %d results in %dms (%d keyword-only, %d semantic-only)\n\n",
		total, response.QueryTime, response.TotalNonSemantic, response.TotalSemantic)
	// Show auto-fuzzy notice and suggestions
	if response.AutoFuzzy && len(response.Suggestions) > 0 {
		fmt.Fprintf(w, "No exact matches found. Showing results for %q instead.\n\n", response.Suggestions[0])
	} else if response.AutoFuzzy {
		fmt.Fprintln(w, "No exact matches found. Showing fuzzy results instead.")
	} else if len(response.Suggestions) > 0 {
		fmt.Fprintf(w, "Did you mean: %s?\n\n", strings.Join(response.Suggestions, ", "))
	}
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

// writeSearchResultsCompact writes one result per line (source, rank, score, file path).
func writeSearchResultsCompact(w io.Writer, response *models.SearchResponse) {
	total := response.TotalNonSemantic + response.TotalSemantic
	fmt.Fprintf(w, "Found %d results in %dms\n", total, response.QueryTime)
	// Show auto-fuzzy notice and suggestions
	if response.AutoFuzzy && len(response.Suggestions) > 0 {
		fmt.Fprintf(w, "No exact matches found. Showing results for %q instead.\n", response.Suggestions[0])
	} else if response.AutoFuzzy {
		fmt.Fprintln(w, "No exact matches found. Showing fuzzy results instead.")
	} else if len(response.Suggestions) > 0 {
		fmt.Fprintf(w, "Did you mean: %s?\n", strings.Join(response.Suggestions, ", "))
	}
	for _, result := range response.NonSemanticResults {
		writeOneResultCompact(w, result, "keyword")
	}
	for _, result := range response.SemanticResults {
		writeOneResultCompact(w, result, "semantic")
	}
}

func writeOneResultCompact(w io.Writer, result *models.SearchResult, source string) {
	path := DocumentFilePath(result.Document)
	if path == "" {
		path = SanitizeForLine(result.Document.Title)
	}
	if path == "" {
		path = Truncate(SanitizeForLine(result.Document.Content), 80)
	}
	fmt.Fprintf(w, "[%s] #%d %.4f | %s\n", source, result.Rank, result.Score, path)
}

// DocumentFilePath returns the stored file path from document metadata (source_path), or empty if not set.
func DocumentFilePath(doc *models.Document) string {
	if doc == nil || doc.Metadata == nil {
		return ""
	}
	v, ok := doc.Metadata["source_path"]
	if !ok {
		return ""
	}
	s, _ := v.(string)
	return s
}

// SanitizeForLine replaces newlines and tabs with spaces for single-line output.
func SanitizeForLine(s string) string {
	return strings.TrimSpace(strings.ReplaceAll(strings.ReplaceAll(s, "\n", " "), "\t", " "))
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
