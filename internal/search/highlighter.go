package search

// Highlight truncates content to maxLen and optionally wraps query terms (simplified: truncate only).
func Highlight(content string, maxLen int) string {
	if maxLen <= 0 || len(content) <= maxLen {
		return content
	}
	return content[:maxLen] + "..."
}
