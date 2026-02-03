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
	if m["b"] != 1.0 {
		t.Errorf("max score should be 1.0, got %f", m["b"])
	}
	if m["a"] != 0.5 {
		t.Errorf("a should be 0.5, got %f", m["a"])
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

func TestFuse(t *testing.T) {
	kw := map[string]float64{"d1": 1.0, "d2": 0.5}
	sem := map[string]float64{"d1": 0.5, "d2": 1.0}
	results := Fuse(kw, sem, 0.5, 0.5)
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	if results[0].Score < results[1].Score {
		t.Error("results should be sorted by score descending")
	}
}
