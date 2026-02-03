package fileid

import (
	"path/filepath"
	"testing"
)

func TestFileDocID(t *testing.T) {
	// Deterministic: same path gives same ID
	id1 := FileDocID("/foo/bar.txt")
	id2 := FileDocID("/foo/bar.txt")
	if id1 != id2 {
		t.Errorf("same path should give same ID: %q vs %q", id1, id2)
	}
	if id1 == "" {
		t.Error("ID should not be empty")
	}
	if len(id1) < 10 {
		t.Errorf("ID too short: %q", id1)
	}
	if id1[:len(prefix)] != prefix {
		t.Errorf("ID should have prefix %q: got %q", prefix, id1)
	}
}

func TestFileDocID_differentPaths(t *testing.T) {
	id1 := FileDocID("/foo/bar.txt")
	id2 := FileDocID("/foo/baz.txt")
	if id1 == id2 {
		t.Errorf("different paths should give different IDs: %q", id1)
	}
}

func TestFileDocID_normalized(t *testing.T) {
	// Clean path: /foo/bar and /foo/bar/ and /foo/./bar should match
	id1 := FileDocID("/foo/bar")
	id2 := FileDocID("/foo/bar/")
	id3 := FileDocID("/foo/./bar")
	if id1 != id2 {
		t.Errorf("paths differing only by trailing slash should match: %q vs %q", id1, id2)
	}
	if id1 != id3 {
		t.Errorf("paths with . should normalize: %q vs %q", id1, id3)
	}
}

func TestFileDocID_relativeBecomesClean(t *testing.T) {
	// We expect callers to pass absolute path; but Clean("a/b") stays "a/b"
	id := FileDocID("a/b.txt")
	if id == "" || id[:len(prefix)] != prefix {
		t.Errorf("relative path still gets valid ID: %q", id)
	}
	// Same relative path gives same ID
	if FileDocID("a/b.txt") != FileDocID("a/b.txt") {
		t.Error("same relative path should be deterministic")
	}
}

func TestFileDocID_absoluteFromFilepath(t *testing.T) {
	abs, _ := filepath.Abs(".")
	id := FileDocID(abs)
	if id == "" || id[:len(prefix)] != prefix {
		t.Errorf("absolute path: got %q", id)
	}
}
