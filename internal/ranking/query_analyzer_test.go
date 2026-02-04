package ranking

import (
	"testing"
)

func TestQueryAnalyzer_Analyze(t *testing.T) {
	qa := NewQueryAnalyzer()

	tests := []struct {
		name           string
		query          string
		wantTerms      []string
		wantPhrases    []string
		wantNegated    []string
		wantQueryType  QueryType
		wantWildcard   bool
	}{
		{
			name:          "single word",
			query:         "budget",
			wantTerms:     []string{"budget"},
			wantPhrases:   []string{},
			wantNegated:   []string{},
			wantQueryType: QueryTypeSingleWord,
			wantWildcard:  false,
		},
		{
			name:          "multiple words",
			query:         "annual budget report",
			wantTerms:     []string{"annual", "budget", "report"},
			wantPhrases:   []string{},
			wantNegated:   []string{},
			wantQueryType: QueryTypeMultiWord,
			wantWildcard:  false,
		},
		{
			name:          "quoted phrase double quotes",
			query:         `"net profit margin"`,
			wantTerms:     []string{},
			wantPhrases:   []string{"net profit margin"},
			wantNegated:   []string{},
			wantQueryType: QueryTypePhrase,
			wantWildcard:  false,
		},
		{
			name:          "quoted phrase single quotes",
			query:         `'annual report'`,
			wantTerms:     []string{},
			wantPhrases:   []string{"annual report"},
			wantNegated:   []string{},
			wantQueryType: QueryTypePhrase,
			wantWildcard:  false,
		},
		{
			name:          "mixed phrase and terms",
			query:         `"budget analysis" 2024`,
			wantTerms:     []string{"2024"},
			wantPhrases:   []string{"budget analysis"},
			wantNegated:   []string{},
			wantQueryType: QueryTypePhrase,
			wantWildcard:  false,
		},
		{
			name:          "negated term with dash",
			query:         "budget -draft",
			wantTerms:     []string{"budget"},
			wantPhrases:   []string{},
			wantNegated:   []string{"draft"},
			wantQueryType: QueryTypeBoolean,
			wantWildcard:  false,
		},
		{
			name:          "wildcard asterisk",
			query:         "budget*",
			wantTerms:     []string{"budget"}, // Wildcards detected but normalized
			wantPhrases:   []string{},
			wantNegated:   []string{},
			wantQueryType: QueryTypeWildcard,
			wantWildcard:  true,
		},
		{
			name:          "wildcard question mark",
			query:         "budge?",
			wantTerms:     []string{"budge"}, // Wildcards detected but normalized
			wantPhrases:   []string{},
			wantNegated:   []string{},
			wantQueryType: QueryTypeWildcard,
			wantWildcard:  true,
		},
		{
			name:          "empty query",
			query:         "",
			wantTerms:     []string{},
			wantPhrases:   []string{},
			wantNegated:   []string{},
			wantQueryType: QueryTypeSingleWord,
			wantWildcard:  false,
		},
		{
			name:          "case normalization",
			query:         "Budget REPORT Annual",
			wantTerms:     []string{"budget", "report", "annual"},
			wantPhrases:   []string{},
			wantNegated:   []string{},
			wantQueryType: QueryTypeMultiWord,
			wantWildcard:  false,
		},
		{
			name:          "punctuation handling",
			query:         "budget, report.",
			wantTerms:     []string{"budget", "report"},
			wantPhrases:   []string{},
			wantNegated:   []string{},
			wantQueryType: QueryTypeMultiWord,
			wantWildcard:  false,
		},
		{
			name:          "special characters in email",
			query:         "user@email.com",
			wantTerms:     []string{"user@email.com"},
			wantPhrases:   []string{},
			wantNegated:   []string{},
			wantQueryType: QueryTypeSingleWord,
			wantWildcard:  false,
		},
		{
			name:          "skip AND OR operators",
			query:         "budget AND report OR draft",
			wantTerms:     []string{"budget", "report", "draft"},
			wantPhrases:   []string{},
			wantNegated:   []string{},
			wantQueryType: QueryTypeMultiWord,
			wantWildcard:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := qa.Analyze(tt.query)

			// Check terms
			if len(result.Terms) != len(tt.wantTerms) {
				t.Errorf("Terms count = %d, want %d", len(result.Terms), len(tt.wantTerms))
			} else {
				for i, term := range result.Terms {
					if term != tt.wantTerms[i] {
						t.Errorf("Term[%d] = %q, want %q", i, term, tt.wantTerms[i])
					}
				}
			}

			// Check phrases
			if len(result.Phrases) != len(tt.wantPhrases) {
				t.Errorf("Phrases count = %d, want %d", len(result.Phrases), len(tt.wantPhrases))
			} else {
				for i, phrase := range result.Phrases {
					if phrase != tt.wantPhrases[i] {
						t.Errorf("Phrase[%d] = %q, want %q", i, phrase, tt.wantPhrases[i])
					}
				}
			}

			// Check negated terms
			if len(result.NegatedTerms) != len(tt.wantNegated) {
				t.Errorf("NegatedTerms count = %d, want %d", len(result.NegatedTerms), len(tt.wantNegated))
			}

			// Check query type
			if result.QueryType != tt.wantQueryType {
				t.Errorf("QueryType = %v, want %v", result.QueryType, tt.wantQueryType)
			}

			// Check wildcard
			if result.HasWildcard != tt.wantWildcard {
				t.Errorf("HasWildcard = %v, want %v", result.HasWildcard, tt.wantWildcard)
			}
		})
	}
}

func TestAllTermsMatch(t *testing.T) {
	tests := []struct {
		name  string
		terms []string
		text  string
		want  bool
	}{
		{
			name:  "all terms present",
			terms: []string{"budget", "report"},
			text:  "This is the annual budget report for 2024",
			want:  true,
		},
		{
			name:  "some terms missing",
			terms: []string{"budget", "missing"},
			text:  "This is the annual budget report",
			want:  false,
		},
		{
			name:  "case insensitive",
			terms: []string{"budget", "report"},
			text:  "BUDGET REPORT",
			want:  true,
		},
		{
			name:  "empty terms",
			terms: []string{},
			text:  "any text",
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := AllTermsMatch(tt.terms, tt.text)
			if got != tt.want {
				t.Errorf("AllTermsMatch() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCountMatchingTerms(t *testing.T) {
	tests := []struct {
		name  string
		terms []string
		text  string
		want  int
	}{
		{
			name:  "all terms match",
			terms: []string{"budget", "report"},
			text:  "annual budget report",
			want:  2,
		},
		{
			name:  "partial match",
			terms: []string{"budget", "missing", "report"},
			text:  "annual budget report",
			want:  2,
		},
		{
			name:  "no match",
			terms: []string{"xyz", "abc"},
			text:  "annual budget report",
			want:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CountMatchingTerms(tt.terms, tt.text)
			if got != tt.want {
				t.Errorf("CountMatchingTerms() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTermsInOrder(t *testing.T) {
	tests := []struct {
		name  string
		terms []string
		text  string
		want  bool
	}{
		{
			name:  "terms in order",
			terms: []string{"annual", "budget", "report"},
			text:  "This is the annual budget report",
			want:  true,
		},
		{
			name:  "terms out of order",
			terms: []string{"report", "annual"},
			text:  "This is the annual budget report",
			want:  false,
		},
		{
			name:  "terms with gap",
			terms: []string{"annual", "report"},
			text:  "annual budget report",
			want:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := TermsInOrder(tt.terms, tt.text)
			if got != tt.want {
				t.Errorf("TermsInOrder() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNormalizeFilename(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		want     string
	}{
		{
			name:     "simple filename with extension",
			filename: "budget.txt",
			want:     "budget",
		},
		{
			name:     "underscores to spaces",
			filename: "annual_budget_report.pdf",
			want:     "annual budget report",
		},
		{
			name:     "hyphens to spaces",
			filename: "annual-budget-report.pdf",
			want:     "annual budget report",
		},
		{
			name:     "mixed separators",
			filename: "annual_budget-report.2024.xlsx",
			want:     "annual budget report 2024",
		},
		{
			name:     "no extension",
			filename: "budget",
			want:     "budget",
		},
		{
			name:     "uppercase",
			filename: "BUDGET_REPORT.PDF",
			want:     "budget report",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NormalizeFilename(tt.filename)
			if got != tt.want {
				t.Errorf("NormalizeFilename() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestExtractExtension(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		want     string
	}{
		{
			name:     "txt extension",
			filename: "budget.txt",
			want:     "txt",
		},
		{
			name:     "pdf extension",
			filename: "report.PDF",
			want:     "pdf",
		},
		{
			name:     "no extension",
			filename: "budget",
			want:     "",
		},
		{
			name:     "multiple dots",
			filename: "budget.2024.xlsx",
			want:     "xlsx",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractExtension(tt.filename)
			if got != tt.want {
				t.Errorf("ExtractExtension() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestIsPrefixMatch(t *testing.T) {
	tests := []struct {
		name string
		term string
		text string
		want bool
	}{
		{
			name: "prefix match",
			term: "budg",
			text: "budget report",
			want: true,
		},
		{
			name: "exact word match",
			term: "budget",
			text: "budget report",
			want: true,
		},
		{
			name: "no prefix match",
			term: "rep",
			text: "budget report",
			want: true,
		},
		{
			name: "substring not prefix",
			term: "udge",
			text: "budget report",
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsPrefixMatch(tt.term, tt.text)
			if got != tt.want {
				t.Errorf("IsPrefixMatch() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCountOccurrences(t *testing.T) {
	tests := []struct {
		name    string
		term    string
		text    string
		want    int
	}{
		{
			name: "single occurrence",
			term: "budget",
			text: "This is the budget report",
			want: 1,
		},
		{
			name: "multiple occurrences",
			term: "budget",
			text: "Budget report: budget is important for budget planning",
			want: 3,
		},
		{
			name: "no occurrence",
			term: "missing",
			text: "This is the budget report",
			want: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CountOccurrences(tt.term, tt.text)
			if got != tt.want {
				t.Errorf("CountOccurrences() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFindPhrasePosition(t *testing.T) {
	tests := []struct {
		name    string
		phrase  string
		text    string
		wantPos int
	}{
		{
			name:    "phrase found",
			phrase:  "budget report",
			text:    "This is the budget report for 2024",
			wantPos: 12,
		},
		{
			name:    "phrase not found",
			phrase:  "missing phrase",
			text:    "This is the budget report",
			wantPos: -1,
		},
		{
			name:    "case insensitive",
			phrase:  "BUDGET REPORT",
			text:    "This is the budget report",
			wantPos: 12,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FindPhrasePosition(tt.phrase, tt.text)
			if got != tt.wantPos {
				t.Errorf("FindPhrasePosition() = %v, want %v", got, tt.wantPos)
			}
		})
	}
}

func TestQueryTypeString(t *testing.T) {
	tests := []struct {
		qt   QueryType
		want string
	}{
		{QueryTypeSingleWord, "single_word"},
		{QueryTypeMultiWord, "multi_word"},
		{QueryTypePhrase, "phrase"},
		{QueryTypeWildcard, "wildcard"},
		{QueryTypeBoolean, "boolean"},
		{QueryType(99), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.qt.String(); got != tt.want {
				t.Errorf("QueryType.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMatchTypeString(t *testing.T) {
	tests := []struct {
		mt   MatchType
		want string
	}{
		{MatchTypeNone, "none"},
		{MatchTypePartial, "partial"},
		{MatchTypeAllWords, "all_words"},
		{MatchTypePhrase, "phrase"},
		{MatchTypeExact, "exact"},
		{MatchType(99), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.mt.String(); got != tt.want {
				t.Errorf("MatchType.String() = %v, want %v", got, tt.want)
			}
		})
	}
}
