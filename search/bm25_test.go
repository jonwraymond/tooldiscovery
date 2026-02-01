package search

import (
	"fmt"
	"strings"
	"sync"
	"testing"

	"github.com/jonwraymond/tooldiscovery/index"
)

// Cycle 1: Interface Compliance & Basic Structure

func TestBM25Searcher_ImplementsSearcher(t *testing.T) {
	t.Helper()
	// Compile-time interface check
	var _ index.Searcher = (*BM25Searcher)(nil)
}

func TestNewBM25Searcher_DefaultConfig(t *testing.T) {
	s := NewBM25Searcher(BM25Config{})
	if s == nil {
		t.Fatal("NewBM25Searcher returned nil")
	}
}

func TestBM25Config_Defaults(t *testing.T) {
	s := NewBM25Searcher(BM25Config{})

	// Check that defaults are applied
	if s.cfg.NameBoost != 3 {
		t.Errorf("NameBoost: got %d, want 3", s.cfg.NameBoost)
	}
	if s.cfg.NamespaceBoost != 2 {
		t.Errorf("NamespaceBoost: got %d, want 2", s.cfg.NamespaceBoost)
	}
	if s.cfg.TagsBoost != 2 {
		t.Errorf("TagsBoost: got %d, want 2", s.cfg.TagsBoost)
	}
}

func TestBM25Config_CustomValues(t *testing.T) {
	cfg := BM25Config{
		NameBoost:      5,
		NamespaceBoost: 4,
		TagsBoost:      3,
		MaxDocs:        100,
		MaxDocTextLen:  500,
	}
	s := NewBM25Searcher(cfg)

	if s.cfg.NameBoost != 5 {
		t.Errorf("NameBoost: got %d, want 5", s.cfg.NameBoost)
	}
	if s.cfg.NamespaceBoost != 4 {
		t.Errorf("NamespaceBoost: got %d, want 4", s.cfg.NamespaceBoost)
	}
	if s.cfg.TagsBoost != 3 {
		t.Errorf("TagsBoost: got %d, want 3", s.cfg.TagsBoost)
	}
	if s.cfg.MaxDocs != 100 {
		t.Errorf("MaxDocs: got %d, want 100", s.cfg.MaxDocs)
	}
	if s.cfg.MaxDocTextLen != 500 {
		t.Errorf("MaxDocTextLen: got %d, want 500", s.cfg.MaxDocTextLen)
	}
}

// Cycle 2: Empty Query Behavior

func makeTestDocs(n int) []index.SearchDoc {
	docs := make([]index.SearchDoc, n)
	for i := 0; i < n; i++ {
		id := fmt.Sprintf("tool-%d", i)
		docs[i] = index.SearchDoc{
			ID:      id,
			DocText: fmt.Sprintf("tool %d description", i),
			Summary: index.Summary{
				ID:               id,
				Name:             fmt.Sprintf("Tool%d", i),
				Namespace:        "test",
				ShortDescription: fmt.Sprintf("Description for tool %d", i),
				Tags:             []string{"test"},
			},
		}
	}
	return docs
}

func TestSearch_EmptyQuery_ReturnsFirstNDocs(t *testing.T) {
	s := NewBM25Searcher(BM25Config{})
	docs := makeTestDocs(10)

	results, err := s.Search("", 5, docs)
	if err != nil {
		t.Fatalf("Search returned error: %v", err)
	}
	if len(results) != 5 {
		t.Errorf("got %d results, want 5", len(results))
	}
	// Should return first 5 in order
	for i := 0; i < 5; i++ {
		if results[i].ID != docs[i].ID {
			t.Errorf("result[%d].ID = %s, want %s", i, results[i].ID, docs[i].ID)
		}
	}
}

func TestSearch_EmptyQuery_WhitespaceOnly(t *testing.T) {
	s := NewBM25Searcher(BM25Config{})
	docs := makeTestDocs(5)

	// Various whitespace-only queries should be treated as empty
	whitespaceQueries := []string{"   ", "\t", "\n", "  \t\n  "}
	for _, q := range whitespaceQueries {
		results, err := s.Search(q, 3, docs)
		if err != nil {
			t.Fatalf("Search(%q) returned error: %v", q, err)
		}
		if len(results) != 3 {
			t.Errorf("Search(%q): got %d results, want 3", q, len(results))
		}
	}
}

func TestSearch_EmptyQuery_LimitExceedsDocs(t *testing.T) {
	s := NewBM25Searcher(BM25Config{})
	docs := makeTestDocs(3)

	results, err := s.Search("", 10, docs)
	if err != nil {
		t.Fatalf("Search returned error: %v", err)
	}
	if len(results) != 3 {
		t.Errorf("got %d results, want 3 (all available)", len(results))
	}
}

func TestSearch_EmptyQuery_EmptyDocs(t *testing.T) {
	s := NewBM25Searcher(BM25Config{})
	var docs []index.SearchDoc

	results, err := s.Search("", 10, docs)
	if err != nil {
		t.Fatalf("Search returned error: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("got %d results, want 0", len(results))
	}
}

// Cycle 4: Weighted Document Construction

func TestBuildWeightedDoc_NameDuplication(t *testing.T) {
	cfg := BM25Config{NameBoost: 3, NamespaceBoost: 0, TagsBoost: 0}
	doc := index.SearchDoc{
		Summary: index.Summary{Name: "MyTool"},
		DocText: "",
	}

	result := buildWeightedDoc(cfg, doc)

	// Name should appear 3 times (NameBoost=3)
	count := strings.Count(result, "mytool")
	if count != 3 {
		t.Errorf("name 'mytool' appears %d times, want 3", count)
	}
}

func TestBuildWeightedDoc_NamespaceBoost(t *testing.T) {
	cfg := BM25Config{NameBoost: 0, NamespaceBoost: 2, TagsBoost: 0}
	doc := index.SearchDoc{
		Summary: index.Summary{Namespace: "myns"},
		DocText: "",
	}

	result := buildWeightedDoc(cfg, doc)

	count := strings.Count(result, "myns")
	if count != 2 {
		t.Errorf("namespace 'myns' appears %d times, want 2", count)
	}
}

func TestBuildWeightedDoc_TagsBoost(t *testing.T) {
	cfg := BM25Config{NameBoost: 0, NamespaceBoost: 0, TagsBoost: 2}
	doc := index.SearchDoc{
		Summary: index.Summary{Tags: []string{"tag1", "tag2"}},
		DocText: "",
	}

	result := buildWeightedDoc(cfg, doc)

	// Each tag should appear TagsBoost times
	if strings.Count(result, "tag1") != 2 {
		t.Errorf("tag1 appears %d times, want 2", strings.Count(result, "tag1"))
	}
	if strings.Count(result, "tag2") != 2 {
		t.Errorf("tag2 appears %d times, want 2", strings.Count(result, "tag2"))
	}
}

func TestBuildWeightedDoc_IncludesDocText(t *testing.T) {
	cfg := BM25Config{NameBoost: 0, NamespaceBoost: 0, TagsBoost: 0}
	doc := index.SearchDoc{
		DocText: "unique description text",
	}

	result := buildWeightedDoc(cfg, doc)

	if !strings.Contains(result, "unique description text") {
		t.Errorf("result should contain DocText, got: %s", result)
	}
}

func TestBuildWeightedDoc_ZeroBoost(t *testing.T) {
	cfg := BM25Config{NameBoost: 0, NamespaceBoost: 0, TagsBoost: 0}
	doc := index.SearchDoc{
		Summary: index.Summary{
			Name:      "MyTool",
			Namespace: "myns",
			Tags:      []string{"tag1"},
		},
		DocText: "base text",
	}

	result := buildWeightedDoc(cfg, doc)

	// With zero boosts, name/namespace/tags should not be duplicated
	// but DocText is always included
	if !strings.Contains(result, "base text") {
		t.Error("should contain base text")
	}
	// Should not contain boosted fields when boost is 0
	if strings.Contains(result, "mytool") {
		t.Error("should not contain name when NameBoost=0")
	}
}

func TestBuildWeightedDoc_Lowercased(t *testing.T) {
	cfg := BM25Config{NameBoost: 1, NamespaceBoost: 1, TagsBoost: 1}
	doc := index.SearchDoc{
		Summary: index.Summary{
			Name:      "MyTool",
			Namespace: "MyNamespace",
			Tags:      []string{"MyTag"},
		},
		DocText: "BASE TEXT",
	}

	result := buildWeightedDoc(cfg, doc)

	// All uppercase should be converted to lowercase
	if strings.Contains(result, "MyTool") || strings.Contains(result, "MyNamespace") ||
		strings.Contains(result, "MyTag") || strings.Contains(result, "BASE TEXT") {
		t.Errorf("result should be lowercased, got: %s", result)
	}
	if !strings.Contains(result, "mytool") || !strings.Contains(result, "mynamespace") ||
		!strings.Contains(result, "mytag") || !strings.Contains(result, "base text") {
		t.Errorf("result should contain lowercased text, got: %s", result)
	}
}

func TestBuildWeightedDoc_MaxDocTextLen(t *testing.T) {
	cfg := BM25Config{NameBoost: 0, NamespaceBoost: 0, TagsBoost: 0, MaxDocTextLen: 10}
	doc := index.SearchDoc{
		DocText: "this is a very long description that should be truncated",
	}

	result := buildWeightedDoc(cfg, doc)

	// Should truncate to MaxDocTextLen
	if len(result) > 10 {
		t.Errorf("result length %d exceeds MaxDocTextLen 10", len(result))
	}
}

// Cycle 5: BM25 Search with Bleve

func TestSearch_SingleTermMatch(t *testing.T) {
	s := NewBM25Searcher(BM25Config{})
	docs := []index.SearchDoc{
		{
			ID:      "git-commit",
			DocText: "git commit create a commit",
			Summary: index.Summary{ID: "git-commit", Name: "git-commit"},
		},
		{
			ID:      "git-push",
			DocText: "git push to remote",
			Summary: index.Summary{ID: "git-push", Name: "git-push"},
		},
		{
			ID:      "docker-run",
			DocText: "docker run container",
			Summary: index.Summary{ID: "docker-run", Name: "docker-run"},
		},
	}

	results, err := s.Search("commit", 10, docs)
	if err != nil {
		t.Fatalf("Search error: %v", err)
	}
	if len(results) == 0 {
		t.Fatal("expected at least one result")
	}
	if results[0].ID != "git-commit" {
		t.Errorf("expected git-commit first, got %s", results[0].ID)
	}
}

func TestSearch_MultiTermQuery(t *testing.T) {
	s := NewBM25Searcher(BM25Config{})
	docs := []index.SearchDoc{
		{
			ID:      "git-commit",
			DocText: "git commit create a commit",
			Summary: index.Summary{ID: "git-commit", Name: "git-commit"},
		},
		{
			ID:      "git-push",
			DocText: "git push to remote repository",
			Summary: index.Summary{ID: "git-push", Name: "git-push"},
		},
	}

	results, err := s.Search("git commit", 10, docs)
	if err != nil {
		t.Fatalf("Search error: %v", err)
	}
	if len(results) == 0 {
		t.Fatal("expected results")
	}
	// git-commit should rank higher (has both terms)
	if results[0].ID != "git-commit" {
		t.Errorf("expected git-commit first, got %s", results[0].ID)
	}
}

func TestSearch_NoMatches(t *testing.T) {
	s := NewBM25Searcher(BM25Config{})
	docs := []index.SearchDoc{
		{
			ID:      "git-commit",
			DocText: "git commit",
			Summary: index.Summary{ID: "git-commit", Name: "git-commit"},
		},
	}

	results, err := s.Search("kubernetes", 10, docs)
	if err != nil {
		t.Fatalf("Search error: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results, got %d", len(results))
	}
}

func TestSearch_RespectsLimit(t *testing.T) {
	s := NewBM25Searcher(BM25Config{})
	docs := []index.SearchDoc{
		{ID: "git-1", DocText: "git operation one", Summary: index.Summary{ID: "git-1"}},
		{ID: "git-2", DocText: "git operation two", Summary: index.Summary{ID: "git-2"}},
		{ID: "git-3", DocText: "git operation three", Summary: index.Summary{ID: "git-3"}},
		{ID: "git-4", DocText: "git operation four", Summary: index.Summary{ID: "git-4"}},
		{ID: "git-5", DocText: "git operation five", Summary: index.Summary{ID: "git-5"}},
	}

	results, err := s.Search("git", 2, docs)
	if err != nil {
		t.Fatalf("Search error: %v", err)
	}
	if len(results) != 2 {
		t.Errorf("expected 2 results, got %d", len(results))
	}
}

func TestSearch_CaseInsensitive(t *testing.T) {
	s := NewBM25Searcher(BM25Config{})
	docs := []index.SearchDoc{
		{
			ID:      "docker-run",
			DocText: "DOCKER RUN CONTAINER",
			Summary: index.Summary{ID: "docker-run", Name: "Docker-Run"},
		},
	}

	// Lowercase query should match uppercase content
	results, err := s.Search("docker", 10, docs)
	if err != nil {
		t.Fatalf("Search error: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("expected 1 result, got %d", len(results))
	}

	// Uppercase query should also work
	results, err = s.Search("DOCKER", 10, docs)
	if err != nil {
		t.Fatalf("Search error: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("expected 1 result for uppercase query, got %d", len(results))
	}
}

func TestSearch_NameBoostAffectsRanking(t *testing.T) {
	s := NewBM25Searcher(BM25Config{NameBoost: 10, NamespaceBoost: 0, TagsBoost: 0})
	docs := []index.SearchDoc{
		{
			ID:      "commit-in-desc",
			DocText: "this is about commit operations commit commit commit",
			Summary: index.Summary{ID: "commit-in-desc", Name: "other-tool"},
		},
		{
			ID:      "commit-in-name",
			DocText: "does something",
			Summary: index.Summary{ID: "commit-in-name", Name: "commit"},
		},
	}

	results, err := s.Search("commit", 10, docs)
	if err != nil {
		t.Fatalf("Search error: %v", err)
	}
	if len(results) < 2 {
		t.Fatal("expected 2 results")
	}
	// With high name boost, the one with "commit" in name should rank first
	if results[0].ID != "commit-in-name" {
		t.Errorf("expected commit-in-name first due to name boost, got %s", results[0].ID)
	}
}

func TestSearch_TagsBoostAffectsRanking(t *testing.T) {
	// Use high tags boost to ensure tag matches rank higher
	s := NewBM25Searcher(BM25Config{NameBoost: 0, NamespaceBoost: 0, TagsBoost: 10})
	docs := []index.SearchDoc{
		{
			ID:      "vcs-in-desc",
			DocText: "vcs operations version control",
			Summary: index.Summary{ID: "vcs-in-desc", Name: "other"},
		},
		{
			ID:      "vcs-in-tag",
			DocText: "does something else entirely",
			Summary: index.Summary{ID: "vcs-in-tag", Name: "tool", Tags: []string{"vcs", "version"}},
		},
	}

	results, err := s.Search("vcs", 10, docs)
	if err != nil {
		t.Fatalf("Search error: %v", err)
	}
	if len(results) < 2 {
		t.Fatal("expected 2 results")
	}
	// With high tags boost, the one with "vcs" tag should rank first
	if results[0].ID != "vcs-in-tag" {
		t.Errorf("expected vcs-in-tag first due to tags boost, got %s", results[0].ID)
	}
}

// Cycle 6: Deterministic Tie-Breaking

func TestSearch_TieBreaking_ScoreFirst(t *testing.T) {
	s := NewBM25Searcher(BM25Config{NameBoost: 5})
	docs := []index.SearchDoc{
		{
			ID:      "aaa-low-score",
			DocText: "git operations",
			Summary: index.Summary{ID: "aaa-low-score", Name: "other"},
		},
		{
			ID:      "zzz-high-score",
			DocText: "other stuff",
			Summary: index.Summary{ID: "zzz-high-score", Name: "git"},
		},
	}

	results, err := s.Search("git", 10, docs)
	if err != nil {
		t.Fatalf("Search error: %v", err)
	}
	if len(results) < 2 {
		t.Fatal("expected 2 results")
	}
	// Higher score should come first regardless of ID
	if results[0].ID != "zzz-high-score" {
		t.Errorf("expected zzz-high-score first (higher score), got %s", results[0].ID)
	}
}

func TestSearch_TieBreaking_LexicographicID(t *testing.T) {
	s := NewBM25Searcher(BM25Config{NameBoost: 0, NamespaceBoost: 0, TagsBoost: 0})
	// All docs have identical content, so they should have identical scores
	docs := []index.SearchDoc{
		{
			ID:      "charlie",
			DocText: "git commit",
			Summary: index.Summary{ID: "charlie", Name: "tool"},
		},
		{
			ID:      "alpha",
			DocText: "git commit",
			Summary: index.Summary{ID: "alpha", Name: "tool"},
		},
		{
			ID:      "bravo",
			DocText: "git commit",
			Summary: index.Summary{ID: "bravo", Name: "tool"},
		},
	}

	results, err := s.Search("git", 10, docs)
	if err != nil {
		t.Fatalf("Search error: %v", err)
	}
	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}
	// Same score means lexicographic ID ordering
	expected := []string{"alpha", "bravo", "charlie"}
	for i, want := range expected {
		if results[i].ID != want {
			t.Errorf("result[%d] = %s, want %s", i, results[i].ID, want)
		}
	}
}

func TestSearch_TieBreaking_Deterministic(t *testing.T) {
	s := NewBM25Searcher(BM25Config{})
	docs := []index.SearchDoc{
		{ID: "tool-a", DocText: "git operations", Summary: index.Summary{ID: "tool-a"}},
		{ID: "tool-b", DocText: "git operations", Summary: index.Summary{ID: "tool-b"}},
		{ID: "tool-c", DocText: "git operations", Summary: index.Summary{ID: "tool-c"}},
	}

	// Run search multiple times
	var firstResult []string
	for i := 0; i < 10; i++ {
		results, err := s.Search("git", 10, docs)
		if err != nil {
			t.Fatalf("Search error: %v", err)
		}

		ids := make([]string, len(results))
		for j, r := range results {
			ids[j] = r.ID
		}

		if firstResult == nil {
			firstResult = ids
		} else {
			// Results should be identical every time
			for j, id := range ids {
				if id != firstResult[j] {
					t.Errorf("non-deterministic: run %d result[%d] = %s, first run = %s", i, j, id, firstResult[j])
				}
			}
		}
	}
}

// Cycle 7: Index Caching & Thread Safety

func TestSearch_IndexCaching_SameDocsSameIndex(t *testing.T) {
	s := NewBM25Searcher(BM25Config{})
	docs := []index.SearchDoc{
		{ID: "tool-1", DocText: "git commit", Summary: index.Summary{ID: "tool-1"}},
	}

	// First search builds index
	_, err := s.Search("git", 10, docs)
	if err != nil {
		t.Fatalf("first search error: %v", err)
	}
	indexCount1 := s.IndexBuildCount()

	// Second search with same docs should reuse index
	_, err = s.Search("commit", 10, docs)
	if err != nil {
		t.Fatalf("second search error: %v", err)
	}
	indexCount2 := s.IndexBuildCount()

	if indexCount2 != indexCount1 {
		t.Errorf("index was rebuilt unnecessarily: build count went from %d to %d", indexCount1, indexCount2)
	}
}

func TestSearch_PermutedDocsNoRebuild(t *testing.T) {
	s := NewBM25Searcher(BM25Config{})

	// Create docs in two different orders (simulating map iteration variance)
	docsOrderA := []index.SearchDoc{
		{ID: "tool-a", DocText: "git commit", Summary: index.Summary{ID: "tool-a"}},
		{ID: "tool-b", DocText: "docker run", Summary: index.Summary{ID: "tool-b"}},
		{ID: "tool-c", DocText: "npm install", Summary: index.Summary{ID: "tool-c"}},
	}
	docsOrderB := []index.SearchDoc{
		{ID: "tool-c", DocText: "npm install", Summary: index.Summary{ID: "tool-c"}},
		{ID: "tool-a", DocText: "git commit", Summary: index.Summary{ID: "tool-a"}},
		{ID: "tool-b", DocText: "docker run", Summary: index.Summary{ID: "tool-b"}},
	}

	// First search builds index
	_, err := s.Search("git", 10, docsOrderA)
	if err != nil {
		t.Fatalf("first search error: %v", err)
	}
	buildCount1 := s.IndexBuildCount()

	// Second search with same docs in different order should NOT rebuild
	_, err = s.Search("docker", 10, docsOrderB)
	if err != nil {
		t.Fatalf("second search error: %v", err)
	}
	buildCount2 := s.IndexBuildCount()

	if buildCount2 != buildCount1 {
		t.Errorf("index was rebuilt for permuted docs: count went from %d to %d", buildCount1, buildCount2)
	}
}

func TestSearch_IndexCaching_DocsChangeRebuild(t *testing.T) {
	s := NewBM25Searcher(BM25Config{})
	docs1 := []index.SearchDoc{
		{ID: "tool-1", DocText: "git commit", Summary: index.Summary{ID: "tool-1"}},
	}
	docs2 := []index.SearchDoc{
		{ID: "tool-1", DocText: "docker run", Summary: index.Summary{ID: "tool-1"}},
	}

	// First search
	_, err := s.Search("git", 10, docs1)
	if err != nil {
		t.Fatalf("first search error: %v", err)
	}
	indexCount1 := s.IndexBuildCount()

	// Second search with different docs should rebuild
	_, err = s.Search("docker", 10, docs2)
	if err != nil {
		t.Fatalf("second search error: %v", err)
	}
	indexCount2 := s.IndexBuildCount()

	if indexCount2 <= indexCount1 {
		t.Errorf("index was not rebuilt when docs changed: build count %d", indexCount2)
	}
}

func TestSearch_ConcurrentAccess(t *testing.T) {
	s := NewBM25Searcher(BM25Config{})
	docs := makeTestDocs(100)

	var wg sync.WaitGroup
	errChan := make(chan error, 100)

	// Launch 100 concurrent searches
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			query := fmt.Sprintf("tool %d", i%10)
			_, err := s.Search(query, 10, docs)
			if err != nil {
				errChan <- err
			}
		}(i)
	}

	wg.Wait()
	close(errChan)

	for err := range errChan {
		t.Errorf("concurrent search error: %v", err)
	}
}

func TestSearch_MaxDocs(t *testing.T) {
	s := NewBM25Searcher(BM25Config{MaxDocs: 5})
	docs := makeTestDocs(20) // Create 20 docs

	results, err := s.Search("tool", 100, docs)
	if err != nil {
		t.Fatalf("Search error: %v", err)
	}

	// Should be limited by MaxDocs, not limit
	if len(results) > 5 {
		t.Errorf("got %d results, expected at most 5 (MaxDocs)", len(results))
	}
}

// Determinism Tests - Verifies fixes for non-deterministic behavior

func TestSearch_EmptyQuery_DeterministicUnderPermutedInput(t *testing.T) {
	s := NewBM25Searcher(BM25Config{})

	// Same 4 docs in two different orders
	docsOrderA := []index.SearchDoc{
		{ID: "delta", DocText: "fourth tool", Summary: index.Summary{ID: "delta", Name: "Delta"}},
		{ID: "alpha", DocText: "first tool", Summary: index.Summary{ID: "alpha", Name: "Alpha"}},
		{ID: "charlie", DocText: "third tool", Summary: index.Summary{ID: "charlie", Name: "Charlie"}},
		{ID: "bravo", DocText: "second tool", Summary: index.Summary{ID: "bravo", Name: "Bravo"}},
	}
	docsOrderB := []index.SearchDoc{
		{ID: "bravo", DocText: "second tool", Summary: index.Summary{ID: "bravo", Name: "Bravo"}},
		{ID: "charlie", DocText: "third tool", Summary: index.Summary{ID: "charlie", Name: "Charlie"}},
		{ID: "alpha", DocText: "first tool", Summary: index.Summary{ID: "alpha", Name: "Alpha"}},
		{ID: "delta", DocText: "fourth tool", Summary: index.Summary{ID: "delta", Name: "Delta"}},
	}

	// Empty query with limit=2, both orders should return same results
	resultsA, err := s.Search("", 2, docsOrderA)
	if err != nil {
		t.Fatalf("Search A error: %v", err)
	}
	resultsB, err := s.Search("", 2, docsOrderB)
	if err != nil {
		t.Fatalf("Search B error: %v", err)
	}

	if len(resultsA) != 2 || len(resultsB) != 2 {
		t.Fatalf("expected 2 results each, got %d and %d", len(resultsA), len(resultsB))
	}

	// Results should be identical regardless of input order
	// (sorted by ID: alpha, bravo should be first 2)
	for i := range resultsA {
		if resultsA[i].ID != resultsB[i].ID {
			t.Errorf("result[%d] mismatch: order A = %s, order B = %s", i, resultsA[i].ID, resultsB[i].ID)
		}
	}

	// Verify we get the lexicographically first IDs
	if resultsA[0].ID != "alpha" {
		t.Errorf("expected alpha first, got %s", resultsA[0].ID)
	}
	if resultsA[1].ID != "bravo" {
		t.Errorf("expected bravo second, got %s", resultsA[1].ID)
	}
}

func TestSearch_MaxDocs_DeterministicUnderPermutedInput(t *testing.T) {
	s := NewBM25Searcher(BM25Config{MaxDocs: 2})

	// Same 4 docs in two different orders
	docsOrderA := []index.SearchDoc{
		{ID: "delta", DocText: "git tool", Summary: index.Summary{ID: "delta", Name: "Delta"}},
		{ID: "alpha", DocText: "git tool", Summary: index.Summary{ID: "alpha", Name: "Alpha"}},
		{ID: "charlie", DocText: "git tool", Summary: index.Summary{ID: "charlie", Name: "Charlie"}},
		{ID: "bravo", DocText: "git tool", Summary: index.Summary{ID: "bravo", Name: "Bravo"}},
	}
	docsOrderB := []index.SearchDoc{
		{ID: "bravo", DocText: "git tool", Summary: index.Summary{ID: "bravo", Name: "Bravo"}},
		{ID: "charlie", DocText: "git tool", Summary: index.Summary{ID: "charlie", Name: "Charlie"}},
		{ID: "alpha", DocText: "git tool", Summary: index.Summary{ID: "alpha", Name: "Alpha"}},
		{ID: "delta", DocText: "git tool", Summary: index.Summary{ID: "delta", Name: "Delta"}},
	}

	// First search with order A
	resultsA, err := s.Search("git", 10, docsOrderA)
	if err != nil {
		t.Fatalf("Search A error: %v", err)
	}
	buildCountA := s.IndexBuildCount()

	// Second search with order B - same content, different order
	resultsB, err := s.Search("git", 10, docsOrderB)
	if err != nil {
		t.Fatalf("Search B error: %v", err)
	}
	buildCountB := s.IndexBuildCount()

	// Results should be identical (MaxDocs applied AFTER sorting)
	if len(resultsA) != len(resultsB) {
		t.Fatalf("result count mismatch: A=%d, B=%d", len(resultsA), len(resultsB))
	}
	for i := range resultsA {
		if resultsA[i].ID != resultsB[i].ID {
			t.Errorf("result[%d] mismatch: order A = %s, order B = %s", i, resultsA[i].ID, resultsB[i].ID)
		}
	}

	// Should NOT rebuild index - same docs (after sorting) means same fingerprint
	if buildCountB != buildCountA {
		t.Errorf("index was rebuilt for permuted docs: count went from %d to %d", buildCountA, buildCountB)
	}
}

// Cycle 8: Edge Cases & Integration

func TestSearch_EmptyDocs_NonEmptyQuery(t *testing.T) {
	s := NewBM25Searcher(BM25Config{})
	var docs []index.SearchDoc

	results, err := s.Search("git", 10, docs)
	if err != nil {
		t.Fatalf("Search error: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results for empty docs with non-empty query, got %d", len(results))
	}
}

func TestSearch_SpecialCharactersInQuery(t *testing.T) {
	s := NewBM25Searcher(BM25Config{})
	docs := []index.SearchDoc{
		{ID: "tool-1", DocText: "git commit", Summary: index.Summary{ID: "tool-1"}},
	}

	// Various special characters that should not cause panics
	queries := []string{
		"git+commit",
		"git-commit",
		"git/commit",
		"git\\commit",
		"git:commit",
		"git;commit",
		"git'commit",
		"git\"commit",
		"(git)",
		"[git]",
		"{git}",
		"git*",
		"git?",
		"git!",
		"git@#$%^&",
		"",
		"   ",
	}

	for _, q := range queries {
		_, err := s.Search(q, 10, docs)
		// We don't care about results, just that it doesn't panic
		if err != nil {
			// Some special chars may cause query parse errors, which is OK
			t.Logf("Query %q returned error (expected for some special chars): %v", q, err)
		}
	}
}

func TestSearch_UnicodeContent(t *testing.T) {
	s := NewBM25Searcher(BM25Config{})
	docs := []index.SearchDoc{
		{
			ID:      "tool-jp",
			DocText: "日本語 tool japanese",
			Summary: index.Summary{ID: "tool-jp", Name: "日本語ツール"},
		},
		{
			ID:      "tool-emoji",
			DocText: "🚀 rocket deploy tool",
			Summary: index.Summary{ID: "tool-emoji", Name: "RocketDeploy"},
		},
		{
			ID:      "tool-accent",
			DocText: "café résumé naïve tool",
			Summary: index.Summary{ID: "tool-accent", Name: "AccentTool"},
		},
	}

	// Should handle unicode in query
	results, err := s.Search("日本語", 10, docs)
	if err != nil {
		t.Fatalf("Search error for Japanese: %v", err)
	}
	if len(results) == 0 {
		t.Log("Note: Japanese query returned 0 results (tokenization dependent)")
	}

	// Should handle emoji search
	results, err = s.Search("rocket", 10, docs)
	if err != nil {
		t.Fatalf("Search error for rocket: %v", err)
	}
	if len(results) != 1 || results[0].ID != "tool-emoji" {
		t.Errorf("expected tool-emoji for 'rocket' query, got %v", results)
	}

	// Should handle accented characters
	results, err = s.Search("café", 10, docs)
	if err != nil {
		t.Fatalf("Search error for café: %v", err)
	}
	// Note: Bleve tokenizer behavior may vary
	if len(results) == 0 {
		t.Log("Note: café query returned 0 results (tokenization dependent)")
	}
}

func TestSearch_VeryLongDocText(t *testing.T) {
	s := NewBM25Searcher(BM25Config{})

	// Create a very long description
	longText := strings.Repeat("git ", 10000)
	docs := []index.SearchDoc{
		{
			ID:      "long-doc",
			DocText: longText,
			Summary: index.Summary{ID: "long-doc", Name: "LongDoc"},
		},
	}

	results, err := s.Search("git", 10, docs)
	if err != nil {
		t.Fatalf("Search error: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("expected 1 result, got %d", len(results))
	}
}

func TestSearch_VeryLongDocText_WithTruncation(t *testing.T) {
	s := NewBM25Searcher(BM25Config{MaxDocTextLen: 100})

	// Create text with unique words at different positions
	longText := strings.Repeat("padding ", 1000) + "uniqueword"
	docs := []index.SearchDoc{
		{
			ID:      "long-doc",
			DocText: longText,
			Summary: index.Summary{ID: "long-doc", Name: "LongDoc"},
		},
	}

	// The unique word should not be found since it's past the truncation point
	results, err := s.Search("uniqueword", 10, docs)
	if err != nil {
		t.Fatalf("Search error: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results (word should be truncated), got %d", len(results))
	}
}

// Integration test demonstrating index.Searcher compatibility
func TestBM25Searcher_ToolindexIntegration(t *testing.T) {
	// Create searcher
	searcher := NewBM25Searcher(BM25Config{
		NameBoost:      3,
		NamespaceBoost: 2,
		TagsBoost:      2,
	})

	// Verify it satisfies the interface
	var _ index.Searcher = searcher

	// Create realistic tool docs
	docs := []index.SearchDoc{
		{
			ID:      "git/commit",
			DocText: "git commit create a git commit with a message version control",
			Summary: index.Summary{
				ID:               "git/commit",
				Name:             "commit",
				Namespace:        "git",
				ShortDescription: "Create a git commit",
				Tags:             []string{"git", "vcs", "commit"},
			},
		},
		{
			ID:      "git/push",
			DocText: "git push push commits to remote repository",
			Summary: index.Summary{
				ID:               "git/push",
				Name:             "push",
				Namespace:        "git",
				ShortDescription: "Push commits to remote",
				Tags:             []string{"git", "vcs", "remote"},
			},
		},
		{
			ID:      "docker/run",
			DocText: "docker run run a container from image",
			Summary: index.Summary{
				ID:               "docker/run",
				Name:             "run",
				Namespace:        "docker",
				ShortDescription: "Run a Docker container",
				Tags:             []string{"docker", "container"},
			},
		},
	}

	// Test search
	results, err := searcher.Search("commit", 10, docs)
	if err != nil {
		t.Fatalf("Search error: %v", err)
	}

	if len(results) == 0 {
		t.Fatal("expected results for 'commit'")
	}

	// git/commit should rank first due to name boost
	if results[0].ID != "git/commit" {
		t.Errorf("expected git/commit first, got %s", results[0].ID)
	}

	// Test namespace search
	results, err = searcher.Search("docker", 10, docs)
	if err != nil {
		t.Fatalf("Search error: %v", err)
	}
	if len(results) == 0 || results[0].ID != "docker/run" {
		t.Errorf("expected docker/run first for 'docker' query")
	}

	// Clean up
	if err := searcher.Close(); err != nil {
		t.Fatalf("close failed: %v", err)
	}
}

// ============================================================
// Tests for Deterministic method
// ============================================================

func TestBM25Searcher_Deterministic(t *testing.T) {
	s := NewBM25Searcher(BM25Config{})
	if !s.Deterministic() {
		t.Error("BM25Searcher.Deterministic() should return true")
	}
}

func TestBM25Searcher_ImplementsDeterministicSearcher(t *testing.T) {
	// Compile-time interface check
	var _ index.DeterministicSearcher = (*BM25Searcher)(nil)
}

// ============================================================
// Tests for Close edge cases
// ============================================================

func TestBM25Searcher_Close_Idempotent(t *testing.T) {
	s := NewBM25Searcher(BM25Config{})
	docs := makeTestDocs(5)

	// Trigger index creation
	_, err := s.Search("test", 5, docs)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	// First close should succeed
	if err := s.Close(); err != nil {
		t.Fatalf("First Close failed: %v", err)
	}

	// Second close should also succeed (no-op)
	if err := s.Close(); err != nil {
		t.Fatalf("Second Close should be idempotent: %v", err)
	}
}

func TestBM25Searcher_Close_NeverUsed(t *testing.T) {
	s := NewBM25Searcher(BM25Config{})

	// Close without ever using the searcher
	if err := s.Close(); err != nil {
		t.Fatalf("Close on unused searcher failed: %v", err)
	}
}

// ============================================================
// Tests for IndexBuildCount
// ============================================================

func TestBM25Searcher_IndexBuildCount(t *testing.T) {
	s := NewBM25Searcher(BM25Config{})

	// Initially zero
	if count := s.IndexBuildCount(); count != 0 {
		t.Errorf("expected initial build count 0, got %d", count)
	}

	docs := makeTestDocs(3)

	// First search triggers build
	_, err := s.Search("test", 3, docs)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	if count := s.IndexBuildCount(); count != 1 {
		t.Errorf("expected build count 1 after first search, got %d", count)
	}

	// Same docs should not trigger rebuild
	_, err = s.Search("test", 3, docs)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	if count := s.IndexBuildCount(); count != 1 {
		t.Errorf("expected build count still 1, got %d", count)
	}

	// Different docs should trigger rebuild
	newDocs := makeTestDocs(5)
	_, err = s.Search("test", 5, newDocs)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	if count := s.IndexBuildCount(); count != 2 {
		t.Errorf("expected build count 2 after new docs, got %d", count)
	}
}

// ============================================================
// Tests for concurrent rebuild races
// ============================================================

func TestBM25Searcher_ConcurrentRebuild(t *testing.T) {
	s := NewBM25Searcher(BM25Config{})

	const goroutines = 10
	const iterations = 20

	var wg sync.WaitGroup
	wg.Add(goroutines)

	// Multiple goroutines searching with different docs
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				// Alternate between two different doc sets
				var docs []index.SearchDoc
				if j%2 == 0 {
					docs = makeTestDocs(3)
				} else {
					docs = makeTestDocs(5)
				}
				_, _ = s.Search("tool", 10, docs)
			}
		}()
	}

	wg.Wait()

	// Just verify we didn't crash and the searcher is still usable
	docs := makeTestDocs(2)
	results, err := s.Search("tool", 2, docs)
	if err != nil {
		t.Fatalf("Search after concurrent access failed: %v", err)
	}
	if len(results) != 2 {
		t.Errorf("expected 2 results, got %d", len(results))
	}
}

// ============================================================
// Tests for fingerprint edge cases
// ============================================================

func TestBM25Searcher_FingerprintSameAfterClose(t *testing.T) {
	s := NewBM25Searcher(BM25Config{})
	docs := makeTestDocs(3)

	// Search, close, search again
	_, err := s.Search("test", 3, docs)
	if err != nil {
		t.Fatalf("First search failed: %v", err)
	}

	initialCount := s.IndexBuildCount()

	if err := s.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	// After close, same docs should trigger rebuild
	_, err = s.Search("test", 3, docs)
	if err != nil {
		t.Fatalf("Search after close failed: %v", err)
	}

	if count := s.IndexBuildCount(); count != initialCount+1 {
		t.Errorf("expected rebuild after close, got count %d (was %d)", count, initialCount)
	}
}
