package embedding

import (
	"testing"
)

func TestEmbeddingCache_GetSet(t *testing.T) {
	c := NewEmbeddingCache(2)
	if v, ok := c.Get("a"); ok || v != nil {
		t.Fatal("expected miss")
	}
	c.Set("a", []float32{1, 2, 3})
	v, ok := c.Get("a")
	if !ok || len(v) != 3 || v[0] != 1 {
		t.Errorf("Get: got %v, %v", v, ok)
	}
	c.Set("b", []float32{4, 5})
	c.Set("c", []float32{6}) // evicts a
	if _, ok := c.Get("a"); ok {
		t.Error("expected a to be evicted")
	}
	if _, ok := c.Get("b"); !ok {
		t.Error("expected b to remain")
	}
	if _, ok := c.Get("c"); !ok {
		t.Error("expected c to be present")
	}
}
