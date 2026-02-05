package keyword

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/hyperjump/sagasu/internal/models"
)

func TestBleveIndex_SearchFindsContent(t *testing.T) {
	dir := t.TempDir()
	indexPath := filepath.Join(dir, "bleve")

	idx, err := NewBleveIndex(indexPath)
	if err != nil {
		t.Fatalf("NewBleveIndex: %v", err)
	}
	defer func() {
		_ = idx.Close()
	}()

	ctx := context.Background()
	docID := "file:abc123"
	doc := &models.Document{
		ID:      docID,
		Title:   "Ausvet Monthly Report 17 - May 2023.docx",
		Content: "This report mentions Omnisyan and other findings. The Bayes app is also referenced.",
	}

	if err := idx.Index(ctx, doc.ID, doc); err != nil {
		t.Fatalf("Index: %v", err)
	}

	results, err := idx.Search(ctx, "Omnisyan", 10, nil)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(results) == 0 {
		t.Fatal("expected at least one keyword result for \"Omnisyan\" in document content")
	}
	if results[0].ID != docID {
		t.Errorf("first result ID = %q, want %q", results[0].ID, docID)
	}

	// Standard analyzer (no stemming) so "bayes" matches "Bayes" in content
	results2, err := idx.Search(ctx, "bayes", 10, nil)
	if err != nil {
		t.Fatalf("Search bayes: %v", err)
	}
	if len(results2) == 0 {
		t.Fatal("expected at least one keyword result for \"bayes\" in document content (standard analyzer, no stop/stem)")
	}
	if results2[0].ID != docID {
		t.Errorf("first result ID = %q, want %q", results2[0].ID, docID)
	}
}

func TestBleveIndex_SearchFindsTitle(t *testing.T) {
	dir := t.TempDir()
	indexPath := filepath.Join(dir, "bleve")

	idx, err := NewBleveIndex(indexPath)
	if err != nil {
		t.Fatalf("NewBleveIndex: %v", err)
	}
	defer func() {
		_ = idx.Close()
	}()

	ctx := context.Background()
	docID := "file:xyz"
	doc := &models.Document{
		ID:      docID,
		Title:   "Ausvet Monthly Report 17 - May 2023.docx",
		Content: "Some body text.",
	}

	if err := idx.Index(ctx, doc.ID, doc); err != nil {
		t.Fatalf("Index: %v", err)
	}

	// Query "Report" (English analyzer stems so "Report" matches title)
	results, err := idx.Search(ctx, "Report", 10, nil)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(results) == 0 {
		t.Fatal("expected at least one keyword result for \"Report\" in title")
	}
	if results[0].ID != docID {
		t.Errorf("first result ID = %q, want %q", results[0].ID, docID)
	}
}

func TestBleveIndex_OpenExistingReusesIndex(t *testing.T) {
	dir := t.TempDir()
	indexPath := filepath.Join(dir, "bleve")

	idx1, err := NewBleveIndex(indexPath)
	if err != nil {
		t.Fatalf("NewBleveIndex: %v", err)
	}
	ctx := context.Background()
	doc := &models.Document{ID: "doc1", Title: "T", Content: "uniqueword"}
	if err := idx1.Index(ctx, doc.ID, doc); err != nil {
		t.Fatalf("Index: %v", err)
	}
	if err := idx1.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	// Opening an existing index reuses it so keyword search works with incremental sync.
	idx2, err := NewBleveIndex(indexPath)
	if err != nil {
		t.Fatalf("NewBleveIndex (open existing): %v", err)
	}
	defer func() {
		_ = idx2.Close()
	}()

	results, err := idx2.Search(ctx, "uniqueword", 10, nil)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(results) != 1 || results[0].ID != "doc1" {
		t.Errorf("after open existing, index should persist; got %d results", len(results))
	}
}

func TestBleveIndex_Delete(t *testing.T) {
	dir := t.TempDir()
	indexPath := filepath.Join(dir, "bleve")

	idx, err := NewBleveIndex(indexPath)
	if err != nil {
		t.Fatalf("NewBleveIndex: %v", err)
	}
	defer func() {
		_ = idx.Close()
	}()

	ctx := context.Background()
	doc := &models.Document{ID: "doc1", Title: "T", Content: "onlyindoc1"}
	if err := idx.Index(ctx, doc.ID, doc); err != nil {
		t.Fatalf("Index: %v", err)
	}

	if err := idx.Delete(ctx, doc.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	results, err := idx.Search(ctx, "onlyindoc1", 10, nil)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results after delete, got %d", len(results))
	}
}

func TestNewBleveIndex_createsDir(t *testing.T) {
	dir := t.TempDir()
	indexPath := filepath.Join(dir, "sub", "bleve")

	idx, err := NewBleveIndex(indexPath)
	if err != nil {
		t.Fatalf("NewBleveIndex: %v", err)
	}
	_ = idx.Close()

	if _, err := os.Stat(indexPath); err != nil {
		t.Errorf("index path should exist: %v", err)
	}
}

func TestBleveIndex_Search_titleBoostRanksTitleMatchHigher(t *testing.T) {
	dir := t.TempDir()
	indexPath := filepath.Join(dir, "bleve")

	idx, err := NewBleveIndex(indexPath)
	if err != nil {
		t.Fatalf("NewBleveIndex: %v", err)
	}
	defer func() {
		_ = idx.Close()
	}()

	ctx := context.Background()
	// Doc A: query terms only in title (filename), minimal content
	docA := &models.Document{
		ID:      "docA",
		Title:   "hyperjump company profile 2021.pptx",
		Content: "Slide content with no hyperjump or profile words.",
	}
	// Doc B: query terms only in content, generic title
	docB := &models.Document{
		ID:      "docB",
		Title:   "Generic Report.docx",
		Content: "This report discusses hyperjump and profile at length. Hyperjump profile hyperjump profile.",
	}

	if err := idx.Index(ctx, docA.ID, docA); err != nil {
		t.Fatalf("Index docA: %v", err)
	}
	if err := idx.Index(ctx, docB.ID, docB); err != nil {
		t.Fatalf("Index docB: %v", err)
	}

	// Without boost: content-heavy doc may rank first
	resultsNoBoost, err := idx.Search(ctx, "hyperjump profile", 10, nil)
	if err != nil {
		t.Fatalf("Search (no boost): %v", err)
	}
	if len(resultsNoBoost) < 2 {
		t.Fatalf("expected at least 2 results, got %d", len(resultsNoBoost))
	}

	// With title boost: doc with both terms in title should rank first
	resultsBoosted, err := idx.Search(ctx, "hyperjump profile", 10, &SearchOptions{TitleBoost: 5.0})
	if err != nil {
		t.Fatalf("Search (title boost): %v", err)
	}
	if len(resultsBoosted) < 2 {
		t.Fatalf("expected at least 2 results, got %d", len(resultsBoosted))
	}
	if resultsBoosted[0].ID != "docA" {
		t.Errorf("with title boost, expected first result docA (title match), got %q (score %f)", resultsBoosted[0].ID, resultsBoosted[0].Score)
	}
}

// TestBleveIndex_Search_termCoverageBoostsAllTermMatches tests that documents matching
// ALL query terms rank higher than documents matching only SOME terms.
func TestBleveIndex_Search_termCoverageBoostsAllTermMatches(t *testing.T) {
	dir := t.TempDir()
	indexPath := filepath.Join(dir, "bleve")

	idx, err := NewBleveIndex(indexPath)
	if err != nil {
		t.Fatalf("NewBleveIndex: %v", err)
	}
	defer func() {
		_ = idx.Close()
	}()

	ctx := context.Background()

	// Doc with BOTH query terms in content
	docBothTerms := &models.Document{
		ID:      "docBothTerms",
		Title:   "01.docx",
		Content: "January 2023 Symon-Monika-Neosense Highlights. The symon highlights are important.",
	}
	// Doc with only ONE query term in title
	docOneTerm := &models.Document{
		ID:      "docOneTerm",
		Title:   "Symon Deck 2026.pptx",
		Content: "This presentation covers various topics without mentioning highlights.",
	}

	if err := idx.Index(ctx, docBothTerms.ID, docBothTerms); err != nil {
		t.Fatalf("Index docBothTerms: %v", err)
	}
	if err := idx.Index(ctx, docOneTerm.ID, docOneTerm); err != nil {
		t.Fatalf("Index docOneTerm: %v", err)
	}

	// Search for "symon highlights" - doc with BOTH terms should rank first
	results, err := idx.Search(ctx, "symon highlights", 10, &SearchOptions{TitleBoost: 3.0, PhraseBoost: 1.5})
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(results) < 2 {
		t.Fatalf("expected at least 2 results, got %d", len(results))
	}

	// Document with both terms should rank higher due to term coverage bonus
	if results[0].ID != "docBothTerms" {
		t.Errorf("expected docBothTerms (matches both 'symon' and 'highlights') to rank first, got %q", results[0].ID)
	}
}

// TestBleveIndex_Search_phraseBoostBoostedAdjacentTerms tests that documents with
// query terms appearing close together rank higher than those with scattered terms.
func TestBleveIndex_Search_phraseBoostBoostedAdjacentTerms(t *testing.T) {
	dir := t.TempDir()
	indexPath := filepath.Join(dir, "bleve")

	idx, err := NewBleveIndex(indexPath)
	if err != nil {
		t.Fatalf("NewBleveIndex: %v", err)
	}
	defer func() {
		_ = idx.Close()
	}()

	ctx := context.Background()

	// Doc with terms adjacent (phrase match)
	docPhrase := &models.Document{
		ID:      "docPhrase",
		Title:   "report.docx",
		Content: "The machine learning algorithm showed excellent results in our tests.",
	}
	// Doc with terms scattered
	docScattered := &models.Document{
		ID:      "docScattered",
		Title:   "notes.docx",
		Content: "The machine was broken. We had to wait for learning materials to fix it. The algorithm failed.",
	}

	if err := idx.Index(ctx, docPhrase.ID, docPhrase); err != nil {
		t.Fatalf("Index docPhrase: %v", err)
	}
	if err := idx.Index(ctx, docScattered.ID, docScattered); err != nil {
		t.Fatalf("Index docScattered: %v", err)
	}

	// Search for "machine learning" with phrase boost
	results, err := idx.Search(ctx, "machine learning", 10, &SearchOptions{TitleBoost: 3.0, PhraseBoost: 2.0})
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(results) < 2 {
		t.Fatalf("expected at least 2 results, got %d", len(results))
	}

	// Document with phrase "machine learning" should rank first
	if results[0].ID != "docPhrase" {
		t.Errorf("expected docPhrase (has 'machine learning' as phrase) to rank first, got %q", results[0].ID)
	}
}

// TestBleveIndex_Search_additiveScoring tests that title and content scores are added
// (not max'd), so a document with matches in both ranks higher.
func TestBleveIndex_Search_additiveScoring(t *testing.T) {
	dir := t.TempDir()
	indexPath := filepath.Join(dir, "bleve")

	idx, err := NewBleveIndex(indexPath)
	if err != nil {
		t.Fatalf("NewBleveIndex: %v", err)
	}
	defer func() {
		_ = idx.Close()
	}()

	ctx := context.Background()

	// Doc with term in BOTH title and content
	docBoth := &models.Document{
		ID:      "docBoth",
		Title:   "Neural Network Tutorial.docx",
		Content: "This tutorial explains neural network architectures and training methods for neural networks.",
	}
	// Doc with term only in title
	docTitleOnly := &models.Document{
		ID:      "docTitleOnly",
		Title:   "Neural Network Basics.pptx",
		Content: "This presentation covers machine learning fundamentals.",
	}

	if err := idx.Index(ctx, docBoth.ID, docBoth); err != nil {
		t.Fatalf("Index docBoth: %v", err)
	}
	if err := idx.Index(ctx, docTitleOnly.ID, docTitleOnly); err != nil {
		t.Fatalf("Index docTitleOnly: %v", err)
	}

	// Search for "neural network"
	results, err := idx.Search(ctx, "neural network", 10, &SearchOptions{TitleBoost: 3.0, PhraseBoost: 1.5})
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(results) < 2 {
		t.Fatalf("expected at least 2 results, got %d", len(results))
	}

	// Document with matches in both title AND content should rank higher (additive)
	if results[0].ID != "docBoth" {
		t.Errorf("expected docBoth (matches in title AND content) to rank first, got %q", results[0].ID)
	}
}

// TestBleveIndex_Search_contentOnlyMatchNotFiltered tests that content-only matches
// are not filtered out when there are title matches with higher scores.
func TestBleveIndex_Search_contentOnlyMatchNotFiltered(t *testing.T) {
	dir := t.TempDir()
	indexPath := filepath.Join(dir, "bleve")

	idx, err := NewBleveIndex(indexPath)
	if err != nil {
		t.Fatalf("NewBleveIndex: %v", err)
	}
	defer func() {
		_ = idx.Close()
	}()

	ctx := context.Background()

	// Doc with term in title
	docTitle := &models.Document{
		ID:      "docTitle",
		Title:   "Kubernetes Deployment Guide.docx",
		Content: "General deployment procedures.",
	}
	// Doc with term only in content (no title match)
	docContent := &models.Document{
		ID:      "docContent",
		Title:   "Infrastructure Notes.docx",
		Content: "Our kubernetes cluster runs multiple pods. Kubernetes is essential for our deployment.",
	}

	if err := idx.Index(ctx, docTitle.ID, docTitle); err != nil {
		t.Fatalf("Index docTitle: %v", err)
	}
	if err := idx.Index(ctx, docContent.ID, docContent); err != nil {
		t.Fatalf("Index docContent: %v", err)
	}

	// Search for "kubernetes"
	results, err := idx.Search(ctx, "kubernetes", 10, &SearchOptions{TitleBoost: 3.0})
	if err != nil {
		t.Fatalf("Search: %v", err)
	}

	// Both documents should be found (content-only match should not be filtered)
	if len(results) < 2 {
		t.Fatalf("expected 2 results (title and content matches), got %d", len(results))
	}

	// Verify both documents are in results
	foundTitle, foundContent := false, false
	for _, r := range results {
		if r.ID == "docTitle" {
			foundTitle = true
		}
		if r.ID == "docContent" {
			foundContent = true
		}
	}
	if !foundTitle {
		t.Error("docTitle should be in results")
	}
	if !foundContent {
		t.Error("docContent should be in results (content-only match should not be filtered)")
	}
}

// TestBleveIndex_Search_fuzzyMatchesTypos tests that fuzzy search finds documents
// even when the query contains typos.
func TestBleveIndex_Search_fuzzyMatchesTypos(t *testing.T) {
	dir := t.TempDir()
	indexPath := filepath.Join(dir, "bleve")

	idx, err := NewBleveIndex(indexPath)
	if err != nil {
		t.Fatalf("NewBleveIndex: %v", err)
	}
	defer func() {
		_ = idx.Close()
	}()

	ctx := context.Background()

	// Index a document with "proposal" in the content
	doc := &models.Document{
		ID:      "doc1",
		Title:   "Project Proposal.docx",
		Content: "This proposal outlines the project scope and deliverables.",
	}

	if err := idx.Index(ctx, doc.ID, doc); err != nil {
		t.Fatalf("Index: %v", err)
	}

	// Without fuzzy: searching for "propodal" (typo) should NOT find the document
	resultsNoFuzzy, err := idx.Search(ctx, "propodal", 10, nil)
	if err != nil {
		t.Fatalf("Search (no fuzzy): %v", err)
	}
	if len(resultsNoFuzzy) != 0 {
		t.Errorf("expected 0 results without fuzzy for typo 'propodal', got %d", len(resultsNoFuzzy))
	}

	// With fuzzy: searching for "propodal" should find "proposal"
	resultsFuzzy, err := idx.Search(ctx, "propodal", 10, &SearchOptions{FuzzyEnabled: true, Fuzziness: 2})
	if err != nil {
		t.Fatalf("Search (fuzzy): %v", err)
	}
	if len(resultsFuzzy) == 0 {
		t.Fatal("expected at least 1 result with fuzzy for typo 'propodal' -> 'proposal'")
	}
	if resultsFuzzy[0].ID != doc.ID {
		t.Errorf("expected doc1, got %s", resultsFuzzy[0].ID)
	}
}

// TestBleveIndex_Search_fuzzyWithTitleBoost tests fuzzy search combined with title boost.
func TestBleveIndex_Search_fuzzyWithTitleBoost(t *testing.T) {
	dir := t.TempDir()
	indexPath := filepath.Join(dir, "bleve")

	idx, err := NewBleveIndex(indexPath)
	if err != nil {
		t.Fatalf("NewBleveIndex: %v", err)
	}
	defer func() {
		_ = idx.Close()
	}()

	ctx := context.Background()

	// Doc A: typo-correctable term in title
	docA := &models.Document{
		ID:      "docA",
		Title:   "Budget Report 2024.xlsx",
		Content: "Financial summary for the year.",
	}
	// Doc B: typo-correctable term in content only
	docB := &models.Document{
		ID:      "docB",
		Title:   "Meeting Notes.docx",
		Content: "Discussed the budget for next quarter.",
	}

	if err := idx.Index(ctx, docA.ID, docA); err != nil {
		t.Fatalf("Index docA: %v", err)
	}
	if err := idx.Index(ctx, docB.ID, docB); err != nil {
		t.Fatalf("Index docB: %v", err)
	}

	// Search for "budgat" (typo for "budget") with fuzzy + title boost
	results, err := idx.Search(ctx, "budgat", 10, &SearchOptions{
		FuzzyEnabled: true,
		Fuzziness:    2,
		TitleBoost:   3.0,
	})
	if err != nil {
		t.Fatalf("Search: %v", err)
	}

	if len(results) < 2 {
		t.Fatalf("expected at least 2 results, got %d", len(results))
	}

	// Document with term in title should rank first due to title boost
	if results[0].ID != "docA" {
		t.Errorf("expected docA (title match with boost) to rank first, got %s", results[0].ID)
	}
}

// TestBleveIndex_Search_fuzzyMultipleTerms tests fuzzy search with multiple query terms.
func TestBleveIndex_Search_fuzzyMultipleTerms(t *testing.T) {
	dir := t.TempDir()
	indexPath := filepath.Join(dir, "bleve")

	idx, err := NewBleveIndex(indexPath)
	if err != nil {
		t.Fatalf("NewBleveIndex: %v", err)
	}
	defer func() {
		_ = idx.Close()
	}()

	ctx := context.Background()

	doc := &models.Document{
		ID:      "doc1",
		Title:   "Machine Learning Tutorial.pdf",
		Content: "This tutorial covers neural networks and deep learning fundamentals.",
	}

	if err := idx.Index(ctx, doc.ID, doc); err != nil {
		t.Fatalf("Index: %v", err)
	}

	// Search for "machne lerning" (typos in both words) with fuzzy
	results, err := idx.Search(ctx, "machne lerning", 10, &SearchOptions{
		FuzzyEnabled: true,
		Fuzziness:    2,
		TitleBoost:   3.0,
	})
	if err != nil {
		t.Fatalf("Search: %v", err)
	}

	if len(results) == 0 {
		t.Fatal("expected at least 1 result with fuzzy for typos 'machne lerning' -> 'machine learning'")
	}
	if results[0].ID != doc.ID {
		t.Errorf("expected doc1, got %s", results[0].ID)
	}
}

// TestBleveIndex_Search_fuzzyFuzzinessLevel tests different fuzziness levels.
func TestBleveIndex_Search_fuzzyFuzzinessLevel(t *testing.T) {
	dir := t.TempDir()
	indexPath := filepath.Join(dir, "bleve")

	idx, err := NewBleveIndex(indexPath)
	if err != nil {
		t.Fatalf("NewBleveIndex: %v", err)
	}
	defer func() {
		_ = idx.Close()
	}()

	ctx := context.Background()

	doc := &models.Document{
		ID:      "doc1",
		Title:   "document.txt",
		Content: "The documentation explains everything.",
	}

	if err := idx.Index(ctx, doc.ID, doc); err != nil {
		t.Fatalf("Index: %v", err)
	}

	// "documantation" has 1 character difference from "documentation" (e->a)
	// Fuzziness 1 should match
	results1, err := idx.Search(ctx, "documantation", 10, &SearchOptions{
		FuzzyEnabled: true,
		Fuzziness:    1,
	})
	if err != nil {
		t.Fatalf("Search (fuzziness 1): %v", err)
	}
	if len(results1) == 0 {
		t.Error("fuzziness 1 should match 'documantation' to 'documentation' (1 edit)")
	}

	// "docamantation" has 2 character differences from "documentation" (u->a, e->a)
	// Fuzziness 1 should NOT match
	results2, err := idx.Search(ctx, "docamantation", 10, &SearchOptions{
		FuzzyEnabled: true,
		Fuzziness:    1,
	})
	if err != nil {
		t.Fatalf("Search (fuzziness 1, 2 edits): %v", err)
	}
	if len(results2) != 0 {
		t.Errorf("fuzziness 1 should NOT match 'docamantation' to 'documentation' (2 edits), got %d results", len(results2))
	}

	// Fuzziness 2 SHOULD match "docamantation"
	results3, err := idx.Search(ctx, "docamantation", 10, &SearchOptions{
		FuzzyEnabled: true,
		Fuzziness:    2,
	})
	if err != nil {
		t.Fatalf("Search (fuzziness 2): %v", err)
	}
	if len(results3) == 0 {
		t.Error("fuzziness 2 should match 'docamantation' to 'documentation'")
	}
}

// TestBleveIndex_Search_fuzzyDefaultFuzziness tests that default fuzziness is 2.
func TestBleveIndex_Search_fuzzyDefaultFuzziness(t *testing.T) {
	dir := t.TempDir()
	indexPath := filepath.Join(dir, "bleve")

	idx, err := NewBleveIndex(indexPath)
	if err != nil {
		t.Fatalf("NewBleveIndex: %v", err)
	}
	defer func() {
		_ = idx.Close()
	}()

	ctx := context.Background()

	doc := &models.Document{
		ID:      "doc1",
		Title:   "test.txt",
		Content: "The proposal was accepted.",
	}

	if err := idx.Index(ctx, doc.ID, doc); err != nil {
		t.Fatalf("Index: %v", err)
	}

	// FuzzyEnabled with default fuzziness (0 means use default of 2)
	results, err := idx.Search(ctx, "propodal", 10, &SearchOptions{
		FuzzyEnabled: true,
		// Fuzziness not set, should default to 2
	})
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(results) == 0 {
		t.Error("default fuzziness should be 2, allowing 'propodal' to match 'proposal'")
	}
}

// TestBleveIndex_DocCount tests the DocCount method.
func TestBleveIndex_DocCount(t *testing.T) {
	dir := t.TempDir()
	indexPath := filepath.Join(dir, "bleve")

	idx, err := NewBleveIndex(indexPath)
	if err != nil {
		t.Fatalf("NewBleveIndex: %v", err)
	}
	defer func() {
		_ = idx.Close()
	}()

	ctx := context.Background()

	// Empty index should have 0 docs
	count, err := idx.DocCount()
	if err != nil {
		t.Fatalf("DocCount: %v", err)
	}
	if count != 0 {
		t.Errorf("empty index DocCount = %d, want 0", count)
	}

	// Add a document
	doc := &models.Document{ID: "doc1", Title: "Test", Content: "content"}
	if err := idx.Index(ctx, doc.ID, doc); err != nil {
		t.Fatalf("Index: %v", err)
	}

	// Should have 1 doc
	count, err = idx.DocCount()
	if err != nil {
		t.Fatalf("DocCount: %v", err)
	}
	if count != 1 {
		t.Errorf("DocCount = %d, want 1", count)
	}
}

// TestBleveIndex_GetTermDocFrequency tests the GetTermDocFrequency method.
func TestBleveIndex_GetTermDocFrequency(t *testing.T) {
	dir := t.TempDir()
	indexPath := filepath.Join(dir, "bleve")

	idx, err := NewBleveIndex(indexPath)
	if err != nil {
		t.Fatalf("NewBleveIndex: %v", err)
	}
	defer func() {
		_ = idx.Close()
	}()

	ctx := context.Background()

	// Index two documents with "machine" and one with "learning"
	doc1 := &models.Document{ID: "doc1", Title: "T1", Content: "machine learning algorithms"}
	doc2 := &models.Document{ID: "doc2", Title: "T2", Content: "machine vision systems"}
	if err := idx.Index(ctx, doc1.ID, doc1); err != nil {
		t.Fatalf("Index doc1: %v", err)
	}
	if err := idx.Index(ctx, doc2.ID, doc2); err != nil {
		t.Fatalf("Index doc2: %v", err)
	}

	// "machine" should appear in 2 documents
	freq, err := idx.GetTermDocFrequency("machine")
	if err != nil {
		t.Fatalf("GetTermDocFrequency: %v", err)
	}
	if freq != 2 {
		t.Errorf("GetTermDocFrequency('machine') = %d, want 2", freq)
	}

	// "learning" should appear in 1 document
	freq, err = idx.GetTermDocFrequency("learning")
	if err != nil {
		t.Fatalf("GetTermDocFrequency: %v", err)
	}
	if freq != 1 {
		t.Errorf("GetTermDocFrequency('learning') = %d, want 1", freq)
	}

	// "xyz" should appear in 0 documents
	freq, err = idx.GetTermDocFrequency("xyz")
	if err != nil {
		t.Fatalf("GetTermDocFrequency: %v", err)
	}
	if freq != 0 {
		t.Errorf("GetTermDocFrequency('xyz') = %d, want 0", freq)
	}
}

// TestBleveIndex_GetCorpusStats tests the GetCorpusStats method.
func TestBleveIndex_GetCorpusStats(t *testing.T) {
	dir := t.TempDir()
	indexPath := filepath.Join(dir, "bleve")

	idx, err := NewBleveIndex(indexPath)
	if err != nil {
		t.Fatalf("NewBleveIndex: %v", err)
	}
	defer func() {
		_ = idx.Close()
	}()

	ctx := context.Background()

	// Index documents
	doc1 := &models.Document{ID: "doc1", Title: "T1", Content: "machine learning"}
	doc2 := &models.Document{ID: "doc2", Title: "T2", Content: "machine vision"}
	if err := idx.Index(ctx, doc1.ID, doc1); err != nil {
		t.Fatalf("Index doc1: %v", err)
	}
	if err := idx.Index(ctx, doc2.ID, doc2); err != nil {
		t.Fatalf("Index doc2: %v", err)
	}

	totalDocs, docFreqs, err := idx.GetCorpusStats([]string{"machine", "learning", "xyz"})
	if err != nil {
		t.Fatalf("GetCorpusStats: %v", err)
	}

	if totalDocs != 2 {
		t.Errorf("totalDocs = %d, want 2", totalDocs)
	}
	if docFreqs["machine"] != 2 {
		t.Errorf("docFreqs['machine'] = %d, want 2", docFreqs["machine"])
	}
	if docFreqs["learning"] != 1 {
		t.Errorf("docFreqs['learning'] = %d, want 1", docFreqs["learning"])
	}
	if docFreqs["xyz"] != 0 {
		t.Errorf("docFreqs['xyz'] = %d, want 0", docFreqs["xyz"])
	}
}

// TestBleveIndex_GetAllTerms tests the GetAllTerms method.
func TestBleveIndex_GetAllTerms(t *testing.T) {
	dir := t.TempDir()
	indexPath := filepath.Join(dir, "bleve")

	idx, err := NewBleveIndex(indexPath)
	if err != nil {
		t.Fatalf("NewBleveIndex: %v", err)
	}
	defer func() {
		_ = idx.Close()
	}()

	ctx := context.Background()

	// Index a document
	doc := &models.Document{ID: "doc1", Title: "Hello World", Content: "machine learning algorithms"}
	if err := idx.Index(ctx, doc.ID, doc); err != nil {
		t.Fatalf("Index: %v", err)
	}

	terms, err := idx.GetAllTerms()
	if err != nil {
		t.Fatalf("GetAllTerms: %v", err)
	}

	// Should contain terms from both title and content
	termSet := make(map[string]bool)
	for _, term := range terms {
		termSet[term] = true
	}

	// Check for expected terms (lowercase due to standard analyzer)
	expectedTerms := []string{"hello", "world", "machine", "learning", "algorithms"}
	for _, expected := range expectedTerms {
		if !termSet[expected] {
			t.Errorf("expected term %q in GetAllTerms result", expected)
		}
	}
}

// TestBleveIndex_ContainsTerm tests the ContainsTerm method.
func TestBleveIndex_ContainsTerm(t *testing.T) {
	dir := t.TempDir()
	indexPath := filepath.Join(dir, "bleve")

	idx, err := NewBleveIndex(indexPath)
	if err != nil {
		t.Fatalf("NewBleveIndex: %v", err)
	}
	defer func() {
		_ = idx.Close()
	}()

	ctx := context.Background()

	// Index a document
	doc := &models.Document{ID: "doc1", Title: "Test", Content: "machine learning"}
	if err := idx.Index(ctx, doc.ID, doc); err != nil {
		t.Fatalf("Index: %v", err)
	}

	// Should contain "machine"
	contains, err := idx.ContainsTerm("machine")
	if err != nil {
		t.Fatalf("ContainsTerm: %v", err)
	}
	if !contains {
		t.Error("ContainsTerm('machine') should be true")
	}

	// Should not contain "xyz"
	contains, err = idx.ContainsTerm("xyz")
	if err != nil {
		t.Fatalf("ContainsTerm: %v", err)
	}
	if contains {
		t.Error("ContainsTerm('xyz') should be false")
	}
}

// TestBleveIndex_GetTermFrequency tests the GetTermFrequency method (alias).
func TestBleveIndex_GetTermFrequency(t *testing.T) {
	dir := t.TempDir()
	indexPath := filepath.Join(dir, "bleve")

	idx, err := NewBleveIndex(indexPath)
	if err != nil {
		t.Fatalf("NewBleveIndex: %v", err)
	}
	defer func() {
		_ = idx.Close()
	}()

	ctx := context.Background()

	// Index a document
	doc := &models.Document{ID: "doc1", Title: "Test", Content: "machine learning"}
	if err := idx.Index(ctx, doc.ID, doc); err != nil {
		t.Fatalf("Index: %v", err)
	}

	// GetTermFrequency should work the same as GetTermDocFrequency
	freq, err := idx.GetTermFrequency("machine")
	if err != nil {
		t.Fatalf("GetTermFrequency: %v", err)
	}
	if freq != 1 {
		t.Errorf("GetTermFrequency('machine') = %d, want 1", freq)
	}
}
