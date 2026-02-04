package search

import (
	"testing"

	"github.com/hyperjump/sagasu/internal/keyword"
	"github.com/hyperjump/sagasu/internal/vector"
)

func TestNormalizeKeywordScores(t *testing.T) {
	results := []*keyword.KeywordResult{
		{ID: "a", Score: 2},
		{ID: "b", Score: 4},
		{ID: "c", Score: 1},
	}
	m := NormalizeKeywordScores(results)
	// No longer normalizes - returns raw scores for smart ranking
	if m["b"] != 4.0 {
		t.Errorf("b should be 4.0 (raw score), got %f", m["b"])
	}
	if m["a"] != 2.0 {
		t.Errorf("a should be 2.0 (raw score), got %f", m["a"])
	}
	if m["c"] != 1.0 {
		t.Errorf("c should be 1.0 (raw score), got %f", m["c"])
	}
	if len(m) != 3 {
		t.Errorf("expected 3 entries, got %d", len(m))
	}
}

func TestNormalizeSemanticScores(t *testing.T) {
	results := []*vector.VectorResult{
		{ID: "c1", Score: 0.9},
		{ID: "c2", Score: 0.5},
	}
	m := NormalizeSemanticScores(results)
	if m["c1"] != 0.9 || m["c2"] != 0.5 {
		t.Errorf("unexpected map %v", m)
	}
}

func TestAggregateSemanticByDocument(t *testing.T) {
	chunkToDoc := map[string]string{"c1": "doc1", "c2": "doc1", "c3": "doc2"}
	semantic := map[string]float64{"c1": 0.3, "c2": 0.8, "c3": 0.5}
	byDoc := AggregateSemanticByDocument(chunkToDoc, semantic)
	if byDoc["doc1"] != 0.8 {
		t.Errorf("doc1 max should be 0.8, got %f", byDoc["doc1"])
	}
	if byDoc["doc2"] != 0.5 {
		t.Errorf("doc2 should be 0.5, got %f", byDoc["doc2"])
	}
}

func TestSplitBySource(t *testing.T) {
	kw := map[string]float64{"d1": 1.0, "d2": 0.5, "d3": 0.3}
	semScores := map[string]float64{"d1": 0.8, "d4": 0.9, "d5": 0.2}
	nonSem, semRes := SplitBySource(kw, semScores)
	// d1 in both -> non-semantic only. d2, d3 non-semantic. d4, d5 semantic-only.
	if len(nonSem) != 3 {
		t.Fatalf("expected 3 non-semantic, got %d", len(nonSem))
	}
	if len(semRes) != 2 {
		t.Fatalf("expected 2 semantic, got %d", len(semRes))
	}
	docIDs := make(map[string]struct{})
	for _, r := range nonSem {
		docIDs[r.DocumentID] = struct{}{}
	}
	for _, r := range semRes {
		if _, ok := docIDs[r.DocumentID]; ok {
			t.Errorf("duplicate document %s in both lists", r.DocumentID)
		}
		docIDs[r.DocumentID] = struct{}{}
		if r.KeywordScore != 0 {
			t.Errorf("semantic result %s should have KeywordScore 0", r.DocumentID)
		}
	}
	// nonSem sorted by keyword score: d1(1), d2(0.5), d3(0.3)
	if nonSem[0].DocumentID != "d1" || nonSem[0].KeywordScore != 1.0 {
		t.Errorf("nonSem[0] expected d1/1.0, got %s/%f", nonSem[0].DocumentID, nonSem[0].KeywordScore)
	}
	if nonSem[1].DocumentID != "d2" || nonSem[1].KeywordScore != 0.5 {
		t.Errorf("nonSem[1] expected d2/0.5, got %s/%f", nonSem[1].DocumentID, nonSem[1].KeywordScore)
	}
	if nonSem[2].DocumentID != "d3" || nonSem[2].KeywordScore != 0.3 {
		t.Errorf("nonSem[2] expected d3/0.3, got %s/%f", nonSem[2].DocumentID, nonSem[2].KeywordScore)
	}
	if semRes[0].DocumentID != "d4" || semRes[0].SemanticScore != 0.9 {
		t.Errorf("semRes[0] expected d4/0.9, got %s/%f", semRes[0].DocumentID, semRes[0].SemanticScore)
	}
	if semRes[1].DocumentID != "d5" || semRes[1].SemanticScore != 0.2 {
		t.Errorf("semRes[1] expected d5/0.2, got %s/%f", semRes[1].DocumentID, semRes[1].SemanticScore)
	}
}

func TestSplitBySource_EmptyKeyword(t *testing.T) {
	kw := map[string]float64{}
	sem := map[string]float64{"d1": 0.5}
	nonSem, semRes := SplitBySource(kw, sem)
	if len(nonSem) != 0 {
		t.Errorf("expected 0 non-semantic, got %d", len(nonSem))
	}
	if len(semRes) != 1 || semRes[0].DocumentID != "d1" {
		t.Errorf("expected 1 semantic d1, got %v", semRes)
	}
}

func TestSplitBySource_EmptySemantic(t *testing.T) {
	kw := map[string]float64{"d1": 0.5}
	sem := map[string]float64{}
	nonSem, semRes := SplitBySource(kw, sem)
	if len(nonSem) != 1 || nonSem[0].DocumentID != "d1" {
		t.Errorf("expected 1 non-semantic d1, got %v", nonSem)
	}
	if len(semRes) != 0 {
		t.Errorf("expected 0 semantic, got %d", len(semRes))
	}
}
