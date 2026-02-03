package utils

import (
	"testing"
)

func TestTruncate(t *testing.T) {
	if Truncate("hello", 10) != "hello" {
		t.Error("short string unchanged")
	}
	if Truncate("hello world", 5) != "hello..." {
		t.Errorf("got %s", Truncate("hello world", 5))
	}
	if Truncate("x", 0) != "x" {
		t.Error("maxLen 0 returns as-is")
	}
}
