package search

import (
	"testing"
)

func TestHighlight(t *testing.T) {
	if Highlight("short", 10) != "short" {
		t.Error("short string should be unchanged")
	}
	if Highlight("long text here", 4) != "long..." {
		t.Errorf("got %s", Highlight("long text here", 4))
	}
	if Highlight("x", 0) != "x" {
		t.Error("maxLen 0 should return as-is")
	}
}
