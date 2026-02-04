package e2e

import (
	"testing"
)

func TestBuildCorpus_Returns100Documents(t *testing.T) {
	c := BuildCorpus()
	if c.TotalDocs != 100 {
		t.Errorf("expected 100 documents, got %d", c.TotalDocs)
	}
	if len(c.Documents) != 100 {
		t.Errorf("expected len(Documents)=100, got %d", len(c.Documents))
	}
}

func TestBuildCorpus_QueryTestCasesExist(t *testing.T) {
	c := BuildCorpus()
	if c.TotalQueries == 0 {
		t.Fatal("expected at least one query test case")
	}
	for i, tc := range c.TestCases {
		if tc.Query == "" {
			t.Errorf("test case %d: empty query", i)
		}
		if len(tc.ExpectedDocIDs) == 0 {
			t.Errorf("test case %d: no expected doc IDs", i)
		}
	}
}

func TestBuildCorpus_ExpectedDocsContainQueryPhrase(t *testing.T) {
	c := BuildCorpus()
	docByID := make(map[string]E2EDocument)
	for _, d := range c.Documents {
		docByID[d.ID] = d
	}
	for _, tc := range c.TestCases {
		for _, docID := range tc.ExpectedDocIDs {
			doc, ok := docByID[docID]
			if !ok {
				t.Errorf("expected doc ID %q not in corpus", docID)
				continue
			}
			if !containsPhrase(doc, tc.Query) {
				t.Errorf("doc %q (title=%q) does not contain query phrase %q", docID, doc.Title, tc.Query)
			}
		}
	}
}

func TestCorpus_ToDocumentInputs(t *testing.T) {
	c := BuildCorpus()
	inputs := c.ToDocumentInputs()
	if len(inputs) != len(c.Documents) {
		t.Errorf("expected %d inputs, got %d", len(c.Documents), len(inputs))
	}
	for i := range inputs {
		if inputs[i].ID != c.Documents[i].ID {
			t.Errorf("input[%d].ID = %q, want %q", i, inputs[i].ID, c.Documents[i].ID)
		}
		if inputs[i].Title != c.Documents[i].Title {
			t.Errorf("input[%d].Title = %q, want %q", i, inputs[i].Title, c.Documents[i].Title)
		}
		if inputs[i].Content != c.Documents[i].Content {
			t.Errorf("input[%d].Content mismatch", i)
		}
	}
}

func TestContainsPhrase(t *testing.T) {
	tests := []struct {
		doc     E2EDocument
		phrase  string
		contain bool
	}{
		{E2EDocument{Title: "Go", Content: "Go golang concurrency"}, "golang", true},
		{E2EDocument{Title: "Go", Content: "Go golang concurrency"}, "Rust", false},
		{E2EDocument{Title: "Python programming", Content: "Python is great"}, "Python programming", true},
	}
	for i, tt := range tests {
		got := containsPhrase(tt.doc, tt.phrase)
		if got != tt.contain {
			t.Errorf("test %d: containsPhrase(%q) = %v, want %v", i, tt.phrase, got, tt.contain)
		}
	}
}
