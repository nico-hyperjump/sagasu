// Package cli provides CLI utilities for Sagasu.
package cli

import (
	"fmt"
	"strings"

	"github.com/hyperjump/sagasu/internal/models"
)

// PrintSearchResults prints search results to stdout (non-semantic and semantic sections).
func PrintSearchResults(response *models.SearchResponse) {
	total := response.TotalNonSemantic + response.TotalSemantic
	fmt.Printf("\nFound %d results in %dms (%d keyword-only, %d semantic-only)\n\n",
		total, response.QueryTime, response.TotalNonSemantic, response.TotalSemantic)
	if len(response.NonSemanticResults) > 0 {
		fmt.Println("--- Non-semantic (keyword) results ---")
		for _, result := range response.NonSemanticResults {
			printOneResult(result, "keyword")
		}
	}
	if len(response.SemanticResults) > 0 {
		fmt.Println("--- Semantic results ---")
		for _, result := range response.SemanticResults {
			printOneResult(result, "semantic")
		}
	}
}

func printOneResult(result *models.SearchResult, source string) {
	fmt.Printf("─────────────────────────────────────────────────────────\n")
	fmt.Printf("[%s] Rank: %d | Score: %.4f (Keyword: %.4f, Semantic: %.4f)\n",
		source, result.Rank, result.Score, result.KeywordScore, result.SemanticScore)
	fmt.Printf("ID: %s\n", result.Document.ID)
	if result.Document.Title != "" {
		fmt.Printf("Title: %s\n", result.Document.Title)
	}
	fmt.Printf("\n%s\n", Truncate(result.Document.Content, 200))
	fmt.Println()
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
