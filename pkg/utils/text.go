// Package utils provides shared utilities for text, math, and logging.
package utils

// Truncate returns s truncated to maxLen characters, with "..." appended if truncated.
// If maxLen is 0 or negative, returns s unchanged.
func Truncate(s string, maxLen int) string {
	if maxLen <= 0 || len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
