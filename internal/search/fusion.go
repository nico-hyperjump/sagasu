// Package search provides hybrid search (keyword + semantic) and result fusion.
package search

import (
	"sort"

	"github.com/hyperjump/sagasu/internal/keyword"
	"github.com/hyperjump/sagasu/internal/vector"
)

// FusedResult holds a document ID and fused keyword/semantic scores.
type FusedResult struct {
	DocumentID    string
	Score         float64
	KeywordScore  float64
	SemanticScore float64
}

// NormalizeKeywordScores normalizes keyword scores to [0,1] by max.
func NormalizeKeywordScores(results []*keyword.KeywordResult) map[string]float64 {
	if len(results) == 0 {
		return make(map[string]float64)
	}
	maxScore := results[0].Score
	for _, r := range results {
		if r.Score > maxScore {
			maxScore = r.Score
		}
	}
	normalized := make(map[string]float64)
	for _, r := range results {
		if maxScore > 0 {
			normalized[r.ID] = r.Score / maxScore
		} else {
			normalized[r.ID] = 0
		}
	}
	return normalized
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

// Fuse merges keyword and semantic score maps with weights and returns sorted FusedResults.
func Fuse(keywordScores, semanticScores map[string]float64, keywordWeight, semanticWeight float64) []*FusedResult {
	scoreMap := make(map[string]*FusedResult)
	for id, score := range keywordScores {
		scoreMap[id] = &FusedResult{
			DocumentID:   id,
			KeywordScore: score,
		}
	}
	for id, score := range semanticScores {
		if result, exists := scoreMap[id]; exists {
			result.SemanticScore = score
		} else {
			scoreMap[id] = &FusedResult{
				DocumentID:    id,
				SemanticScore: score,
			}
		}
	}
	results := make([]*FusedResult, 0, len(scoreMap))
	for _, result := range scoreMap {
		result.Score = (keywordWeight * result.KeywordScore) + (semanticWeight * result.SemanticScore)
		results = append(results, result)
	}
	sort.Slice(results, func(i, j int) bool { return results[i].Score > results[j].Score })
	return results
}
