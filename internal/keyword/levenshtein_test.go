package keyword

import "testing"

func TestLevenshteinDistance(t *testing.T) {
	tests := []struct {
		name     string
		a        string
		b        string
		expected int
	}{
		// Identical strings
		{"identical empty", "", "", 0},
		{"identical word", "hello", "hello", 0},
		{"identical unicode", "こんにちは", "こんにちは", 0},

		// Empty string cases
		{"empty a", "", "hello", 5},
		{"empty b", "hello", "", 5},

		// Single character differences
		{"one substitution", "cat", "bat", 1},
		{"one insertion", "cat", "cart", 1},
		{"one deletion", "cart", "cat", 1},

		// Multiple differences
		{"two substitutions", "cat", "dog", 3},
		{"kitten to sitting", "kitten", "sitting", 3},
		{"saturday to sunday", "saturday", "sunday", 3},

		// Common typos
		{"proposal to propodal", "proposal", "propodal", 1},
		{"documentation to documantation", "documentation", "documantation", 1},
		{"machine to machne", "machine", "machne", 1},
		{"learning to lerning", "learning", "lerning", 1},

		// Case sensitivity
		{"case difference", "Hello", "hello", 1},
		{"all caps", "HELLO", "hello", 5},

		// Unicode
		{"unicode substitution", "café", "cafe", 1},
		{"unicode insertion", "naïve", "naive", 1},

		// Longer strings
		{"longer strings", "algorithm", "altruistic", 6},

		// Transposition (not counted as single edit in Levenshtein)
		{"transposition ab-ba", "ab", "ba", 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := LevenshteinDistance(tt.a, tt.b)
			if result != tt.expected {
				t.Errorf("LevenshteinDistance(%q, %q) = %d, want %d", tt.a, tt.b, result, tt.expected)
			}
			// Test symmetry: distance(a,b) should equal distance(b,a)
			resultReverse := LevenshteinDistance(tt.b, tt.a)
			if result != resultReverse {
				t.Errorf("LevenshteinDistance is not symmetric: (%q,%q)=%d, (%q,%q)=%d",
					tt.a, tt.b, result, tt.b, tt.a, resultReverse)
			}
		})
	}
}

func TestDamerauLevenshteinDistance(t *testing.T) {
	tests := []struct {
		name     string
		a        string
		b        string
		expected int
	}{
		// Identical strings
		{"identical empty", "", "", 0},
		{"identical word", "hello", "hello", 0},

		// Empty string cases
		{"empty a", "", "hello", 5},
		{"empty b", "hello", "", 5},

		// Single character differences
		{"one substitution", "cat", "bat", 1},
		{"one insertion", "cat", "cart", 1},
		{"one deletion", "cart", "cat", 1},

		// Transposition (should be 1 in Damerau-Levenshtein)
		{"transposition ab-ba", "ab", "ba", 1},
		{"transposition teh-the", "teh", "the", 1},
		{"transposition recieve-receive", "recieve", "receive", 1},

		// Multiple differences
		{"kitten to sitting", "kitten", "sitting", 3},
		{"saturday to sunday", "saturday", "sunday", 3},

		// Common typos with transposition
		{"hte to the", "hte", "the", 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DamerauLevenshteinDistance(tt.a, tt.b)
			if result != tt.expected {
				t.Errorf("DamerauLevenshteinDistance(%q, %q) = %d, want %d", tt.a, tt.b, result, tt.expected)
			}
			// Test symmetry
			resultReverse := DamerauLevenshteinDistance(tt.b, tt.a)
			if result != resultReverse {
				t.Errorf("DamerauLevenshteinDistance is not symmetric: (%q,%q)=%d, (%q,%q)=%d",
					tt.a, tt.b, result, tt.b, tt.a, resultReverse)
			}
		})
	}
}

func TestMin(t *testing.T) {
	tests := []struct {
		a, b, c  int
		expected int
	}{
		{1, 2, 3, 1},
		{3, 1, 2, 1},
		{2, 3, 1, 1},
		{1, 1, 1, 1},
		{0, 0, 0, 0},
		{-1, 0, 1, -1},
	}

	for _, tt := range tests {
		result := min(tt.a, tt.b, tt.c)
		if result != tt.expected {
			t.Errorf("min(%d, %d, %d) = %d, want %d", tt.a, tt.b, tt.c, result, tt.expected)
		}
	}
}

func TestMinTwo(t *testing.T) {
	tests := []struct {
		a, b     int
		expected int
	}{
		{1, 2, 1},
		{2, 1, 1},
		{1, 1, 1},
		{0, 0, 0},
		{-1, 1, -1},
	}

	for _, tt := range tests {
		result := minTwo(tt.a, tt.b)
		if result != tt.expected {
			t.Errorf("minTwo(%d, %d) = %d, want %d", tt.a, tt.b, result, tt.expected)
		}
	}
}

// Benchmark tests
func BenchmarkLevenshteinDistance_Short(b *testing.B) {
	for i := 0; i < b.N; i++ {
		LevenshteinDistance("kitten", "sitting")
	}
}

func BenchmarkLevenshteinDistance_Medium(b *testing.B) {
	for i := 0; i < b.N; i++ {
		LevenshteinDistance("documentation", "documantation")
	}
}

func BenchmarkLevenshteinDistance_Long(bench *testing.B) {
	strA := "the quick brown fox jumps over the lazy dog"
	strB := "the quikc brown foz jumsp over teh lazy dog"
	for i := 0; i < bench.N; i++ {
		LevenshteinDistance(strA, strB)
	}
}

func BenchmarkDamerauLevenshteinDistance_Short(b *testing.B) {
	for i := 0; i < b.N; i++ {
		DamerauLevenshteinDistance("kitten", "sitting")
	}
}
