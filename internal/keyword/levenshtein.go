// Package keyword provides keyword (BM25) search indexing and search.
package keyword

// LevenshteinDistance calculates the minimum number of single-character edits
// (insertions, deletions, or substitutions) required to change one string into another.
// This is a pure function with no side effects.
func LevenshteinDistance(a, b string) int {
	if a == b {
		return 0
	}
	if len(a) == 0 {
		return len(b)
	}
	if len(b) == 0 {
		return len(a)
	}

	// Convert to runes for proper Unicode handling
	runesA := []rune(a)
	runesB := []rune(b)
	lenA := len(runesA)
	lenB := len(runesB)

	// Create a matrix to store distances
	// We only need two rows at a time for space efficiency
	prev := make([]int, lenB+1)
	curr := make([]int, lenB+1)

	// Initialize first row
	for j := 0; j <= lenB; j++ {
		prev[j] = j
	}

	// Fill in the rest of the matrix
	for i := 1; i <= lenA; i++ {
		curr[0] = i
		for j := 1; j <= lenB; j++ {
			cost := 0
			if runesA[i-1] != runesB[j-1] {
				cost = 1
			}
			// Minimum of: deletion, insertion, substitution
			curr[j] = min(
				prev[j]+1,      // deletion
				curr[j-1]+1,    // insertion
				prev[j-1]+cost, // substitution
			)
		}
		// Swap rows
		prev, curr = curr, prev
	}

	return prev[lenB]
}

// min returns the minimum of three integers.
func min(a, b, c int) int {
	if a <= b && a <= c {
		return a
	}
	if b <= c {
		return b
	}
	return c
}

// DamerauLevenshteinDistance calculates the Damerau-Levenshtein distance,
// which also considers transpositions (swapping of two adjacent characters)
// as a single edit operation.
func DamerauLevenshteinDistance(a, b string) int {
	if a == b {
		return 0
	}
	if len(a) == 0 {
		return len(b)
	}
	if len(b) == 0 {
		return len(a)
	}

	// Convert to runes for proper Unicode handling
	runesA := []rune(a)
	runesB := []rune(b)
	lenA := len(runesA)
	lenB := len(runesB)

	// Create matrix (need full matrix for transposition check)
	d := make([][]int, lenA+1)
	for i := range d {
		d[i] = make([]int, lenB+1)
	}

	// Initialize first row and column
	for i := 0; i <= lenA; i++ {
		d[i][0] = i
	}
	for j := 0; j <= lenB; j++ {
		d[0][j] = j
	}

	// Fill in the rest of the matrix
	for i := 1; i <= lenA; i++ {
		for j := 1; j <= lenB; j++ {
			cost := 0
			if runesA[i-1] != runesB[j-1] {
				cost = 1
			}

			d[i][j] = min(
				d[i-1][j]+1,      // deletion
				d[i][j-1]+1,      // insertion
				d[i-1][j-1]+cost, // substitution
			)

			// Check for transposition
			if i > 1 && j > 1 &&
				runesA[i-1] == runesB[j-2] &&
				runesA[i-2] == runesB[j-1] {
				d[i][j] = minTwo(d[i][j], d[i-2][j-2]+cost)
			}
		}
	}

	return d[lenA][lenB]
}

// minTwo returns the minimum of two integers.
func minTwo(a, b int) int {
	if a <= b {
		return a
	}
	return b
}
