package cli

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/hyperjump/sagasu/internal/models"
)

func TestWriteSearchResults_JSON(t *testing.T) {
	response := &models.SearchResponse{
		Query:            "test query",
		QueryTime:        42,
		TotalNonSemantic: 1,
		TotalSemantic:    0,
		NonSemanticResults: []*models.SearchResult{
			{
				Rank:          1,
				Score:         0.9,
				KeywordScore:  0.9,
				SemanticScore: 0,
				Document: &models.Document{
					ID:       "doc-1",
					Title:    "Test Doc",
					Content:  "Content here",
					Metadata: nil,
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				},
			},
		},
		SemanticResults: []*models.SearchResult{},
	}
	var buf bytes.Buffer
	err := WriteSearchResults(&buf, response, OutputJSON)
	if err != nil {
		t.Fatalf("WriteSearchResults(json): %v", err)
	}
	out := buf.String()
	if out == "" {
		t.Fatal("expected non-empty JSON output")
	}
	var decoded models.SearchResponse
	if err := json.NewDecoder(strings.NewReader(out)).Decode(&decoded); err != nil {
		t.Fatalf("output is not valid JSON: %v\n%s", err, out)
	}
	if decoded.Query != response.Query || decoded.QueryTime != response.QueryTime {
		t.Errorf("decoded query=%q query_time=%d, want query=%q query_time=%d",
			decoded.Query, decoded.QueryTime, response.Query, response.QueryTime)
	}
	if len(decoded.NonSemanticResults) != 1 || decoded.NonSemanticResults[0].Document.ID != "doc-1" {
		t.Errorf("decoded non_semantic_results: want one result with id doc-1, got %+v", decoded.NonSemanticResults)
	}
}

func TestWriteSearchResults_JSON_empty(t *testing.T) {
	response := &models.SearchResponse{
		Query:            "q",
		QueryTime:        0,
		TotalNonSemantic: 0,
		TotalSemantic:    0,
		NonSemanticResults: nil,
		SemanticResults:   nil,
	}
	var buf bytes.Buffer
	err := WriteSearchResults(&buf, response, OutputJSON)
	if err != nil {
		t.Fatalf("WriteSearchResults(json): %v", err)
	}
	var decoded models.SearchResponse
	if err := json.NewDecoder(&buf).Decode(&decoded); err != nil {
		t.Fatalf("empty response JSON decode: %v", err)
	}
	if decoded.TotalNonSemantic != 0 || decoded.TotalSemantic != 0 {
		t.Errorf("expected zeros, got total_non_semantic=%d total_semantic=%d",
			decoded.TotalNonSemantic, decoded.TotalSemantic)
	}
}

func TestWriteSearchResults_text(t *testing.T) {
	response := &models.SearchResponse{
		Query:            "foo",
		QueryTime:        10,
		TotalNonSemantic: 1,
		TotalSemantic:    0,
		NonSemanticResults: []*models.SearchResult{
			{
				Rank:          1,
				Score:         0.5,
				KeywordScore:  0.5,
				SemanticScore: 0,
				Document: &models.Document{
					ID:      "id1",
					Title:   "Title One",
					Content: "Short content",
				},
			},
		},
		SemanticResults: nil,
	}
	var buf bytes.Buffer
	err := WriteSearchResults(&buf, response, OutputText)
	if err != nil {
		t.Fatalf("WriteSearchResults(text): %v", err)
	}
	out := buf.String()
	for _, sub := range []string{"Found 1 results", "10ms", "keyword-only", "Non-semantic", "Rank: 1", "ID: id1", "Title One", "Short content"} {
		if !strings.Contains(out, sub) {
			t.Errorf("text output missing %q:\n%s", sub, out)
		}
	}
}

func TestWriteSearchResults_text_semanticOnly(t *testing.T) {
	response := &models.SearchResponse{
		Query:            "bar",
		QueryTime:        5,
		TotalNonSemantic: 0,
		TotalSemantic:    1,
		NonSemanticResults: nil,
		SemanticResults: []*models.SearchResult{
			{
				Rank:          1,
				Score:         0.8,
				KeywordScore:  0,
				SemanticScore: 0.8,
				Document: &models.Document{
					ID:      "id2",
					Title:   "",
					Content: "Semantic hit content",
				},
			},
		},
	}
	var buf bytes.Buffer
	err := WriteSearchResults(&buf, response, OutputText)
	if err != nil {
		t.Fatalf("WriteSearchResults(text): %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "Semantic results") {
		t.Errorf("expected 'Semantic results' in output:\n%s", out)
	}
	if !strings.Contains(out, "id2") || !strings.Contains(out, "Semantic hit content") {
		t.Errorf("expected id2 and content in output:\n%s", out)
	}
}

func TestWriteSearchResults_unknownFormatTreatedAsText(t *testing.T) {
	response := &models.SearchResponse{Query: "x", QueryTime: 0}
	var buf bytes.Buffer
	err := WriteSearchResults(&buf, response, SearchOutputFormat("unknown"))
	if err != nil {
		t.Fatalf("WriteSearchResults(unknown): %v", err)
	}
	if !strings.Contains(buf.String(), "Found") {
		t.Errorf("unknown format should fall back to text; got %q", buf.String())
	}
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		name   string
		s      string
		maxLen int
		want   string
	}{
		{"empty", "", 5, ""},
		{"short", "hi", 5, "hi"},
		{"exact", "hello", 5, "hello"},
		{"long", "hello world", 5, "hello..."},
		{"maxLen zero", "ab", 0, "ab"},
		{"maxLen negative", "ab", -1, "ab"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Truncate(tt.s, tt.maxLen)
			if got != tt.want {
				t.Errorf("Truncate(%q, %d) = %q, want %q", tt.s, tt.maxLen, got, tt.want)
			}
		})
	}
}

func TestTruncateWords(t *testing.T) {
	tests := []struct {
		name     string
		s        string
		maxWords int
		want     string
	}{
		{"empty", "", 3, ""},
		{"few words", "one two", 3, "one two"},
		{"exact", "one two three", 3, "one two three"},
		{"more", "one two three four", 3, "one two three..."},
		{"single long", "word", 1, "word"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := TruncateWords(tt.s, tt.maxWords)
			if got != tt.want {
				t.Errorf("TruncateWords(%q, %d) = %q, want %q", tt.s, tt.maxWords, got, tt.want)
			}
		})
	}
}

func TestPrintSearchResults(t *testing.T) {
	response := &models.SearchResponse{
		Query:            "print test",
		QueryTime:        1,
		TotalNonSemantic: 0,
		TotalSemantic:    0,
	}
	oldStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	os.Stdout = w
	defer func() {
		os.Stdout = oldStdout
		_ = w.Close()
	}()
	PrintSearchResults(response)
	_ = w.Close()
	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	out := buf.String()
	if !strings.Contains(out, "Found 0 results") {
		t.Errorf("PrintSearchResults should write to stdout; got %q", out)
	}
}
