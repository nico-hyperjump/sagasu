// Package cli provides CLI utilities for Sagasu.
package cli

import (
	"fmt"
	"strings"

	"github.com/hyperjump/sagasu/internal/models"
)

// PrintSearchResults prints search results to stdout.
func PrintSearchResults(response *models.SearchResponse) {
	fmt.Printf("\nFound %d results in %dms\n\n", response.Total, response.QueryTime)
	for _, result := range response.Results {
		fmt.Printf("─────────────────────────────────────────────────────────\n")
		fmt.Printf("Rank: %d | Score: %.4f (Keyword: %.4f, Semantic: %.4f)\n",
			result.Rank, result.Score, result.KeywordScore, result.SemanticScore)
		fmt.Printf("ID: %s\n", result.Document.ID)
		if result.Document.Title != "" {
			fmt.Printf("Title: %s\n", result.Document.Title)
		}
		fmt.Printf("\n%s\n", Truncate(result.Document.Content, 200))
		fmt.Println()
	}
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
