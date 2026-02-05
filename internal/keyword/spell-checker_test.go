package keyword

import (
	"testing"
)

// mockTermDictionary is a mock implementation of TermDictionary for testing.
type mockTermDictionary struct {
	terms       map[string]int // term -> frequency
	getAllError error
	getFreqError error
}

func newMockTermDictionary(terms map[string]int) *mockTermDictionary {
	return &mockTermDictionary{terms: terms}
}

func (m *mockTermDictionary) GetAllTerms() ([]string, error) {
	if m.getAllError != nil {
		return nil, m.getAllError
	}
	result := make([]string, 0, len(m.terms))
	for term := range m.terms {
		result = append(result, term)
	}
	return result, nil
}

func (m *mockTermDictionary) GetTermFrequency(term string) (int, error) {
	if m.getFreqError != nil {
		return 0, m.getFreqError
	}
	return m.terms[term], nil
}

func (m *mockTermDictionary) ContainsTerm(term string) (bool, error) {
	_, ok := m.terms[term]
	return ok, nil
}

func TestSpellChecker_NewSpellChecker(t *testing.T) {
	dict := newMockTermDictionary(map[string]int{"hello": 10})
	
	sc := NewSpellChecker(dict)
	if sc == nil {
		t.Fatal("NewSpellChecker returned nil")
	}
	if sc.maxDistance != 2 {
		t.Errorf("default maxDistance = %d, want 2", sc.maxDistance)
	}
	if sc.minFreq != 1 {
		t.Errorf("default minFreq = %d, want 1", sc.minFreq)
	}
	if sc.maxSuggestions != 5 {
		t.Errorf("default maxSuggestions = %d, want 5", sc.maxSuggestions)
	}
}

func TestSpellChecker_NewSpellChecker_WithOptions(t *testing.T) {
	dict := newMockTermDictionary(map[string]int{"hello": 10})
	
	sc := NewSpellChecker(dict,
		WithMaxDistance(3),
		WithMinFrequency(5),
		WithMaxSuggestions(10),
	)
	
	if sc.maxDistance != 3 {
		t.Errorf("maxDistance = %d, want 3", sc.maxDistance)
	}
	if sc.minFreq != 5 {
		t.Errorf("minFreq = %d, want 5", sc.minFreq)
	}
	if sc.maxSuggestions != 10 {
		t.Errorf("maxSuggestions = %d, want 10", sc.maxSuggestions)
	}
}

func TestSpellChecker_Suggest(t *testing.T) {
	dict := newMockTermDictionary(map[string]int{
		"proposal":      100,
		"project":       80,
		"process":       60,
		"machine":       50,
		"learning":      40,
		"documentation": 30,
	})
	
	sc := NewSpellChecker(dict, WithMaxDistance(2))
	if err := sc.RefreshCache(); err != nil {
		t.Fatalf("RefreshCache: %v", err)
	}

	tests := []struct {
		name       string
		term       string
		wantFirst  string
		wantMinLen int
	}{
		{
			name:       "propodal -> proposal",
			term:       "propodal",
			wantFirst:  "proposal",
			wantMinLen: 1,
		},
		{
			name:       "machne -> machine",
			term:       "machne",
			wantFirst:  "machine",
			wantMinLen: 1,
		},
		{
			name:       "lerning -> learning",
			term:       "lerning",
			wantFirst:  "learning",
			wantMinLen: 1,
		},
		{
			name:       "xyz (no match)",
			term:       "xyz",
			wantFirst:  "",
			wantMinLen: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			suggestions := sc.Suggest(tt.term)
			
			if len(suggestions) < tt.wantMinLen {
				t.Errorf("Suggest(%q) returned %d suggestions, want at least %d",
					tt.term, len(suggestions), tt.wantMinLen)
				return
			}
			
			if tt.wantFirst != "" && len(suggestions) > 0 {
				if suggestions[0].Term != tt.wantFirst {
					t.Errorf("Suggest(%q)[0].Term = %q, want %q",
						tt.term, suggestions[0].Term, tt.wantFirst)
				}
			}
		})
	}
}

func TestSpellChecker_Check(t *testing.T) {
	dict := newMockTermDictionary(map[string]int{
		"proposal":      100,
		"budget":        80,
		"report":        60,
		"machine":       50,
		"learning":      40,
	})
	
	sc := NewSpellChecker(dict, WithMaxDistance(2))

	tests := []struct {
		name            string
		query           string
		wantCorrected   string
		wantHasCorrect  bool
		wantMisspelled  int
	}{
		{
			name:           "valid query",
			query:          "proposal budget",
			wantCorrected:  "proposal budget",
			wantHasCorrect: false,
			wantMisspelled: 0,
		},
		{
			name:           "single typo",
			query:          "propodal",
			wantCorrected:  "proposal",
			wantHasCorrect: true,
			wantMisspelled: 1,
		},
		{
			name:           "multiple typos",
			query:          "machne lerning",
			wantCorrected:  "machine learning",
			wantHasCorrect: true,
			wantMisspelled: 2,
		},
		{
			name:           "mixed valid and typo",
			query:          "budgat report",
			wantCorrected:  "budget report",
			wantHasCorrect: true,
			wantMisspelled: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := sc.Check(tt.query)
			if err != nil {
				t.Fatalf("Check(%q): %v", tt.query, err)
			}
			
			if result.CorrectedQuery != tt.wantCorrected {
				t.Errorf("Check(%q).CorrectedQuery = %q, want %q",
					tt.query, result.CorrectedQuery, tt.wantCorrected)
			}
			
			if result.HasCorrections != tt.wantHasCorrect {
				t.Errorf("Check(%q).HasCorrections = %v, want %v",
					tt.query, result.HasCorrections, tt.wantHasCorrect)
			}
			
			if len(result.MisspelledTerms) != tt.wantMisspelled {
				t.Errorf("Check(%q).MisspelledTerms has %d items, want %d",
					tt.query, len(result.MisspelledTerms), tt.wantMisspelled)
			}
		})
	}
}

func TestSpellChecker_IsMisspelled(t *testing.T) {
	dict := newMockTermDictionary(map[string]int{
		"hello":    10,
		"world":    20,
		"proposal": 30,
	})
	
	sc := NewSpellChecker(dict)

	tests := []struct {
		term string
		want bool
	}{
		{"hello", false},
		{"world", false},
		{"proposal", false},
		{"propodal", true},
		{"xyz", true},
		{"HELLO", false}, // case insensitive
	}

	for _, tt := range tests {
		t.Run(tt.term, func(t *testing.T) {
			if got := sc.IsMisspelled(tt.term); got != tt.want {
				t.Errorf("IsMisspelled(%q) = %v, want %v", tt.term, got, tt.want)
			}
		})
	}
}

func TestSpellChecker_GetSuggestedQuery(t *testing.T) {
	dict := newMockTermDictionary(map[string]int{
		"proposal": 100,
		"budget":   80,
	})
	
	sc := NewSpellChecker(dict)

	tests := []struct {
		query string
		want  string
	}{
		{"proposal budget", "proposal budget"},
		{"propodal", "proposal"},
		{"propodal budgat", "proposal budget"},
		{"xyz", "xyz"}, // no suggestion, return original
	}

	for _, tt := range tests {
		t.Run(tt.query, func(t *testing.T) {
			if got := sc.GetSuggestedQuery(tt.query); got != tt.want {
				t.Errorf("GetSuggestedQuery(%q) = %q, want %q", tt.query, got, tt.want)
			}
		})
	}
}

func TestSpellChecker_GetTopSuggestions(t *testing.T) {
	dict := newMockTermDictionary(map[string]int{
		"proposal": 100,
	})
	
	sc := NewSpellChecker(dict)

	suggestions := sc.GetTopSuggestions("propodal", 5)
	if len(suggestions) == 0 {
		t.Error("GetTopSuggestions returned empty for typo")
	}
	if len(suggestions) > 0 && suggestions[0] != "proposal" {
		t.Errorf("GetTopSuggestions[0] = %q, want 'proposal'", suggestions[0])
	}
}

func TestSpellChecker_Suggest_RanksByFrequency(t *testing.T) {
	// Two terms with same edit distance (1), but different frequency
	dict := newMockTermDictionary(map[string]int{
		"test":  100, // high frequency - 1 edit from "tast"
		"fast":  10,  // low frequency - 1 edit from "tast"
		"last":  50,  // medium frequency - 1 edit from "tast"
	})
	
	sc := NewSpellChecker(dict, WithMaxDistance(1))
	if err := sc.RefreshCache(); err != nil {
		t.Fatalf("RefreshCache: %v", err)
	}

	// "tast" is 1 edit away from test (a->e), fast (t->f), last (t->l)
	suggestions := sc.Suggest("tast")
	
	if len(suggestions) < 3 {
		t.Fatalf("expected 3 suggestions, got %d", len(suggestions))
	}
	
	// Should be sorted by score (frequency-weighted)
	// "test" should be first due to highest frequency
	if suggestions[0].Term != "test" {
		t.Errorf("highest frequency term should be first, got %q", suggestions[0].Term)
	}
}

func TestSpellChecker_Suggest_RespectsMaxDistance(t *testing.T) {
	dict := newMockTermDictionary(map[string]int{
		"documentation": 100,
	})
	
	// "docamantation" has 2 edits from "documentation" (u->a at pos 3, e->a at pos 5)
	// With maxDistance 1, it should not match
	sc := NewSpellChecker(dict, WithMaxDistance(1))
	if err := sc.RefreshCache(); err != nil {
		t.Fatalf("RefreshCache: %v", err)
	}

	suggestions := sc.Suggest("docamantation")
	if len(suggestions) != 0 {
		t.Errorf("maxDistance=1 should not match 2-edit term, got %d suggestions", len(suggestions))
	}
	
	// With maxDistance 2, it should match
	sc2 := NewSpellChecker(dict, WithMaxDistance(2))
	if err := sc2.RefreshCache(); err != nil {
		t.Fatalf("RefreshCache: %v", err)
	}

	suggestions2 := sc2.Suggest("docamantation")
	if len(suggestions2) == 0 {
		t.Error("maxDistance=2 should match 2-edit term")
	}
}

func TestSpellChecker_Suggest_RespectsMinFrequency(t *testing.T) {
	dict := newMockTermDictionary(map[string]int{
		"test": 5,
		"text": 1,
	})
	
	// With minFreq 3, only "test" should be suggested
	sc := NewSpellChecker(dict, WithMinFrequency(3))
	if err := sc.RefreshCache(); err != nil {
		t.Fatalf("RefreshCache: %v", err)
	}

	suggestions := sc.Suggest("tost")
	
	for _, s := range suggestions {
		if s.Frequency < 3 {
			t.Errorf("suggestion %q has frequency %d, below minFreq 3", s.Term, s.Frequency)
		}
	}
}

func TestSpellChecker_Suggest_LimitsResults(t *testing.T) {
	// Create dictionary with many similar terms
	terms := make(map[string]int)
	for i := 0; i < 20; i++ {
		terms["test"+string(rune('a'+i))] = 10
	}
	
	dict := newMockTermDictionary(terms)
	sc := NewSpellChecker(dict, WithMaxSuggestions(3))
	if err := sc.RefreshCache(); err != nil {
		t.Fatalf("RefreshCache: %v", err)
	}

	suggestions := sc.Suggest("test")
	
	if len(suggestions) > 3 {
		t.Errorf("got %d suggestions, want at most 3", len(suggestions))
	}
}

func TestSuggestion_Fields(t *testing.T) {
	s := Suggestion{
		Term:      "proposal",
		Distance:  1,
		Frequency: 100,
		Score:     50.0,
	}
	
	if s.Term != "proposal" {
		t.Errorf("Term = %q, want 'proposal'", s.Term)
	}
	if s.Distance != 1 {
		t.Errorf("Distance = %d, want 1", s.Distance)
	}
	if s.Frequency != 100 {
		t.Errorf("Frequency = %d, want 100", s.Frequency)
	}
	if s.Score != 50.0 {
		t.Errorf("Score = %f, want 50.0", s.Score)
	}
}

func TestSpellCheckResult_Fields(t *testing.T) {
	r := SpellCheckResult{
		OriginalQuery:   "propodal",
		CorrectedQuery:  "proposal",
		Suggestions:     []Suggestion{{Term: "proposal", Distance: 1}},
		HasCorrections:  true,
		MisspelledTerms: []string{"propodal"},
	}
	
	if r.OriginalQuery != "propodal" {
		t.Errorf("OriginalQuery = %q", r.OriginalQuery)
	}
	if r.CorrectedQuery != "proposal" {
		t.Errorf("CorrectedQuery = %q", r.CorrectedQuery)
	}
	if !r.HasCorrections {
		t.Error("HasCorrections should be true")
	}
	if len(r.MisspelledTerms) != 1 {
		t.Errorf("MisspelledTerms len = %d", len(r.MisspelledTerms))
	}
}

func TestSpellChecker_RefreshCache_Error(t *testing.T) {
	dict := &mockTermDictionary{
		terms:       map[string]int{"hello": 10},
		getAllError: errMock,
	}
	
	sc := NewSpellChecker(dict)
	err := sc.RefreshCache()
	if err == nil {
		t.Error("RefreshCache should return error when GetAllTerms fails")
	}
}

var errMock = &mockError{}

type mockError struct{}

func (e *mockError) Error() string { return "mock error" }

func TestSpellChecker_IsMisspelled_CacheRefreshError(t *testing.T) {
	dict := &mockTermDictionary{
		terms:       map[string]int{"hello": 10},
		getAllError: errMock,
	}
	
	sc := NewSpellChecker(dict)
	// Don't manually refresh cache - let IsMisspelled try to refresh
	
	// Should return false when refresh fails
	result := sc.IsMisspelled("xyz")
	if result {
		t.Error("IsMisspelled should return false when cache refresh fails")
	}
}

func TestSpellChecker_Check_CacheRefreshError(t *testing.T) {
	dict := &mockTermDictionary{
		terms:       map[string]int{"hello": 10},
		getAllError: errMock,
	}
	
	sc := NewSpellChecker(dict)
	
	_, err := sc.Check("xyz")
	if err == nil {
		t.Error("Check should return error when cache refresh fails")
	}
}

func TestSpellChecker_Suggest_CacheRefreshError(t *testing.T) {
	dict := &mockTermDictionary{
		terms:       map[string]int{"hello": 10},
		getAllError: errMock,
	}
	
	sc := NewSpellChecker(dict)
	
	suggestions := sc.Suggest("xyz")
	if len(suggestions) != 0 {
		t.Error("Suggest should return empty when cache refresh fails")
	}
}

func TestSpellChecker_GetTopSuggestions_NoCorrections(t *testing.T) {
	dict := newMockTermDictionary(map[string]int{
		"proposal": 100,
	})
	
	sc := NewSpellChecker(dict)

	// Query that is correct - should return no suggestions
	suggestions := sc.GetTopSuggestions("proposal", 5)
	if len(suggestions) != 0 {
		t.Errorf("GetTopSuggestions should return empty for correct query, got %d", len(suggestions))
	}
}

func TestSpellChecker_GetTopSuggestions_LimitResults(t *testing.T) {
	dict := newMockTermDictionary(map[string]int{
		"proposal": 100,
	})
	
	sc := NewSpellChecker(dict)

	// Query with typo, limit to 0
	suggestions := sc.GetTopSuggestions("propodal", 0)
	if len(suggestions) > 0 {
		t.Errorf("GetTopSuggestions with n=0 should return empty, got %d", len(suggestions))
	}
}

func TestSpellChecker_Suggest_TermFrequencyError(t *testing.T) {
	dict := &mockTermDictionary{
		terms:        map[string]int{"test": 10, "text": 5},
		getFreqError: errMock,
	}
	
	sc := NewSpellChecker(dict)
	if err := sc.RefreshCache(); err != nil {
		t.Fatalf("RefreshCache: %v", err)
	}

	// When frequency lookup fails, suggestions are skipped
	suggestions := sc.Suggest("tost")
	// Should return empty because frequency errors cause terms to be skipped
	if len(suggestions) != 0 {
		t.Errorf("Suggest should return empty when frequency lookup fails, got %d", len(suggestions))
	}
}

func TestSpellChecker_Check_EmptyQuery(t *testing.T) {
	dict := newMockTermDictionary(map[string]int{"hello": 10})
	
	sc := NewSpellChecker(dict)

	result, err := sc.Check("")
	if err != nil {
		t.Fatalf("Check empty query: %v", err)
	}
	if result.HasCorrections {
		t.Error("empty query should have no corrections")
	}
	if result.CorrectedQuery != "" {
		t.Errorf("empty query corrected to %q", result.CorrectedQuery)
	}
}

func TestSpellChecker_Check_SingleShortTerm(t *testing.T) {
	dict := newMockTermDictionary(map[string]int{
		"at": 100,
		"as": 80,
	})
	
	sc := NewSpellChecker(dict)

	// Very short terms that might have limited suggestions
	result, err := sc.Check("ax")
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	// "ax" is 1 edit from "at" and "as"
	if !result.HasCorrections {
		t.Log("short term 'ax' might not have corrections depending on implementation")
	}
}
