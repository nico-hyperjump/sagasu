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

func TestWriteSearchResults_compact(t *testing.T) {
	response := &models.SearchResponse{
		Query:            "q",
		QueryTime:        5,
		TotalNonSemantic: 1,
		TotalSemantic:    1,
		NonSemanticResults: []*models.SearchResult{
			{
				Rank:          1,
				Score:         0.5,
				KeywordScore:  0.5,
				SemanticScore: 0,
				Document: &models.Document{
					ID:       "id-k",
					Title:    "Keyword Doc",
					Content:  "Some keyword content",
					Metadata: map[string]interface{}{"source_path": "/path/to/keyword-file.txt"},
				},
			},
		},
		SemanticResults: []*models.SearchResult{
			{
				Rank:          1,
				Score:         0.8,
				KeywordScore:  0,
				SemanticScore: 0.8,
				Document: &models.Document{
					ID:       "id-s",
					Title:    "",
					Content:  "Semantic content with\nnewline",
					Metadata: map[string]interface{}{"source_path": "/home/user/semantic.pdf"},
				},
			},
		},
	}
	var buf bytes.Buffer
	err := WriteSearchResults(&buf, response, OutputCompact)
	if err != nil {
		t.Fatalf("WriteSearchResults(compact): %v", err)
	}
	out := buf.String()
	lines := strings.Split(strings.TrimSuffix(out, "\n"), "\n")
	if len(lines) != 3 {
		t.Errorf("compact should have 3 lines (header + 2 results), got %d:\n%s", len(lines), out)
	}
	if !strings.HasPrefix(lines[0], "Found 2 results") {
		t.Errorf("first line should be header: %q", lines[0])
	}
	if !strings.Contains(lines[1], "[keyword]") || !strings.Contains(lines[1], "/path/to/keyword-file.txt") {
		t.Errorf("second line should be keyword result with file path: %q", lines[1])
	}
	if !strings.Contains(lines[2], "[semantic]") || !strings.Contains(lines[2], "/home/user/semantic.pdf") {
		t.Errorf("third line should be semantic result with file path: %q", lines[2])
	}
	// No ID in output
	if strings.Contains(lines[1], "id-k") || strings.Contains(lines[2], "id-s") {
		t.Errorf("compact output should not show document ID")
	}
	// Newline in content should be collapsed when path is missing (not in this case)
	if strings.Contains(lines[2], "\n") {
		t.Errorf("compact result line must not contain newline: %q", lines[2])
	}
}

func TestWriteSearchResults_compact_fallbackToTitleThenPreview(t *testing.T) {
	// No source_path in metadata: fallback to title, then content preview
	response := &models.SearchResponse{
		Query:             "q",
		QueryTime:         1,
		TotalNonSemantic:  1,
		TotalSemantic:     0,
		NonSemanticResults: []*models.SearchResult{
			{
				Rank: 1, Score: 0.5,
				Document: &models.Document{
					ID: "x", Title: "Fallback Title", Content: "Content here",
					Metadata: nil, // no source_path
				},
			},
		},
	}
	var buf bytes.Buffer
	err := WriteSearchResults(&buf, response, OutputCompact)
	if err != nil {
		t.Fatalf("WriteSearchResults(compact fallback): %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "Fallback Title") {
		t.Errorf("compact with no source_path should show title: %q", out)
	}
}

func TestWriteSearchResults_compact_fallbackToContentPreview(t *testing.T) {
	// No source_path and no title: fallback to content preview
	response := &models.SearchResponse{
		Query:             "q",
		QueryTime:         1,
		TotalNonSemantic:  1,
		TotalSemantic:     0,
		NonSemanticResults: []*models.SearchResult{
			{
				Rank: 1, Score: 0.5,
				Document: &models.Document{
					ID: "x", Title: "", Content: "Only content preview appears here",
					Metadata: nil,
				},
			},
		},
	}
	var buf bytes.Buffer
	err := WriteSearchResults(&buf, response, OutputCompact)
	if err != nil {
		t.Fatalf("WriteSearchResults(compact content fallback): %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "Only content preview") {
		t.Errorf("compact with no path/title should show content preview: %q", out)
	}
}

func TestDocumentFilePath(t *testing.T) {
	tests := []struct {
		name string
		doc  *models.Document
		want string
	}{
		{"nil doc", nil, ""},
		{"nil metadata", &models.Document{Metadata: nil}, ""},
		{"empty metadata", &models.Document{Metadata: map[string]interface{}{}}, ""},
		{"source_path set", &models.Document{Metadata: map[string]interface{}{"source_path": "/a/b.txt"}}, "/a/b.txt"},
		{"source_path not string", &models.Document{Metadata: map[string]interface{}{"source_path": 123}}, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DocumentFilePath(tt.doc)
			if got != tt.want {
				t.Errorf("DocumentFilePath() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestWriteSearchResults_compact_empty(t *testing.T) {
	response := &models.SearchResponse{
		Query:            "q",
		QueryTime:        0,
		TotalNonSemantic: 0,
		TotalSemantic:    0,
	}
	var buf bytes.Buffer
	err := WriteSearchResults(&buf, response, OutputCompact)
	if err != nil {
		t.Fatalf("WriteSearchResults(compact empty): %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "Found 0 results") {
		t.Errorf("expected header with 0 results: %q", out)
	}
	lines := strings.Split(strings.TrimSuffix(out, "\n"), "\n")
	if len(lines) != 1 {
		t.Errorf("compact empty should have 1 line, got %d", len(lines))
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

func TestSanitizeForLine(t *testing.T) {
	tests := []struct {
		name string
		s    string
		want string
	}{
		{"empty", "", ""},
		{"no change", "hello world", "hello world"},
		{"newline", "a\nb", "a b"},
		{"multiple newlines", "a\n\nb", "a  b"},
		{"tab", "a\tb", "a b"},
		{"newline and tab", "a\nb\tc", "a b c"},
		{"leading trailing space", "  x  ", "x"},
		{"leading newline", "\nhello", "hello"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SanitizeForLine(tt.s)
			if got != tt.want {
				t.Errorf("SanitizeForLine(%q) = %q, want %q", tt.s, got, tt.want)
			}
		})
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
