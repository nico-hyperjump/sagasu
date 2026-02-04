// Package search provides hybrid search (keyword + semantic) with split results.
package search

import (
	"sort"

	"github.com/hyperjump/sagasu/internal/keyword"
	"github.com/hyperjump/sagasu/internal/vector"
)

// FusedResult holds a document ID and keyword/semantic scores for split result lists.
type FusedResult struct {
	DocumentID    string
	Score         float64
	KeywordScore  float64
	SemanticScore float64
}

// NormalizeKeywordScores returns keyword scores without normalization.
// The smart ranking in keyword search (term coverage, phrase boost, additive title+content)
// already produces comparable scores, and normalizing by max would compress content-only
// matches too aggressively when title matches exist.
func NormalizeKeywordScores(results []*keyword.KeywordResult) map[string]float64 {
	scores := make(map[string]float64)
	for _, r := range results {
		scores[r.ID] = r.Score
	}
	return scores
}

// NormalizeSemanticScores returns semantic scores as-is (already 0-1 for cosine).
func NormalizeSemanticScores(results []*vector.VectorResult) map[string]float64 {
	normalized := make(map[string]float64)
	for _, r := range results {
		normalized[r.ID] = r.Score
	}
	return normalized
}

// AggregateSemanticByDocument converts chunk ID -> score to document ID -> max score.
// chunkToDoc maps chunk ID to document ID; semanticScores is chunk ID -> score.
func AggregateSemanticByDocument(chunkToDoc map[string]string, semanticScores map[string]float64) map[string]float64 {
	byDoc := make(map[string]float64)
	for chunkID, score := range semanticScores {
		docID := chunkToDoc[chunkID]
		if docID == "" {
			continue
		}
		if s, ok := byDoc[docID]; !ok || score > s {
			byDoc[docID] = score
		}
	}
	return byDoc
}

// SplitBySource splits keyword and semantic score maps into two disjoint result lists:
// nonSemantic = all documents from keyword (sorted by keyword score desc),
// semantic = documents only in semantic, not in keyword (sorted by semantic score desc).
// Documents that appear in both are assigned to non-semantic only so there are no duplicates.
func SplitBySource(keywordScores, semanticScores map[string]float64) (nonSemantic, semantic []*FusedResult) {
	kwDocIDs := make(map[string]struct{})
	for id := range keywordScores {
		kwDocIDs[id] = struct{}{}
	}
	for docID, score := range keywordScores {
		nonSemantic = append(nonSemantic, &FusedResult{
			DocumentID:   docID,
			KeywordScore: score,
			Score:        score,
		})
	}
	for docID, score := range semanticScores {
		if _, inKw := kwDocIDs[docID]; inKw {
			continue
		}
		semantic = append(semantic, &FusedResult{
			DocumentID:    docID,
			SemanticScore: score,
			Score:         score,
		})
	}
	sort.Slice(nonSemantic, func(i, j int) bool { return nonSemantic[i].KeywordScore > nonSemantic[j].KeywordScore })
	sort.Slice(semantic, func(i, j int) bool { return semantic[i].SemanticScore > semantic[j].SemanticScore })
	return nonSemantic, semantic
}
