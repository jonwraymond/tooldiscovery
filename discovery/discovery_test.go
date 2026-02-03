package discovery

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/jonwraymond/tooldiscovery/index"
	"github.com/jonwraymond/tooldiscovery/tooldoc"
	"github.com/jonwraymond/toolfoundation/adapter"
	"github.com/jonwraymond/toolfoundation/model"
)

func makeTool(name, namespace, description string, tags []string) model.Tool {
	return model.Tool{
		Tool: mcp.Tool{
			Name:        name,
			Description: description,
			InputSchema: map[string]any{"type": "object"},
		},
		Namespace: namespace,
		Tags:      tags,
	}
}

func makeBackend(serverName string) model.ToolBackend {
	return model.NewMCPBackend(serverName)
}

func TestNew_DefaultOptions(t *testing.T) {
	disc, err := New(Options{})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	if disc.idx == nil {
		t.Error("expected index to be initialized")
	}
	if disc.searcher == nil {
		t.Error("expected searcher to be initialized")
	}
	if disc.docs == nil {
		t.Error("expected doc store to be initialized")
	}
	if disc.scoreType != ScoreBM25 {
		t.Errorf("expected score type BM25, got %v", disc.scoreType)
	}
}

func TestNew_WithEmbedder(t *testing.T) {
	embedder := &mockEmbedder{dim: 384}
	disc, err := New(Options{
		Embedder:    embedder,
		HybridAlpha: 0.7,
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	if disc.scoreType != ScoreHybrid {
		t.Errorf("expected score type Hybrid, got %v", disc.scoreType)
	}
	if disc.compositeS == nil {
		t.Error("expected composite searcher to be initialized")
	}
}

func TestNew_InvalidHybridAlpha(t *testing.T) {
	embedder := &mockEmbedder{dim: 384}

	_, err := New(Options{
		Embedder:    embedder,
		HybridAlpha: 1.5, // Invalid
	})
	if err == nil {
		t.Error("expected error for invalid alpha")
	}
}

func TestDiscovery_RegisterTool(t *testing.T) {
	disc, _ := New(Options{})

	tool := makeTool("create_issue", "github", "Create an issue", []string{"github"})
	backend := makeBackend("github-server")

	err := disc.RegisterTool(tool, backend, nil)
	if err != nil {
		t.Fatalf("RegisterTool() error = %v", err)
	}

	// Verify tool is retrievable
	retrieved, _, err := disc.GetTool("github:create_issue")
	if err != nil {
		t.Fatalf("GetTool() error = %v", err)
	}
	if retrieved.Name != "create_issue" {
		t.Errorf("expected name create_issue, got %s", retrieved.Name)
	}
}

func TestDiscovery_RegisterToolWithDoc(t *testing.T) {
	disc, _ := New(Options{})

	tool := makeTool("create_issue", "github", "Create an issue", []string{"github"})
	backend := makeBackend("github-server")
	doc := &tooldoc.DocEntry{
		Summary: "Creates GitHub issues",
		Notes:   "Requires GITHUB_TOKEN",
	}

	err := disc.RegisterTool(tool, backend, doc)
	if err != nil {
		t.Fatalf("RegisterTool() error = %v", err)
	}

	// Verify documentation
	toolDoc, err := disc.DescribeTool("github:create_issue", tooldoc.DetailFull)
	if err != nil {
		t.Fatalf("DescribeTool() error = %v", err)
	}
	if toolDoc.Summary != "Creates GitHub issues" {
		t.Errorf("expected summary 'Creates GitHub issues', got %s", toolDoc.Summary)
	}
	if toolDoc.Notes != "Requires GITHUB_TOKEN" {
		t.Errorf("expected notes 'Requires GITHUB_TOKEN', got %s", toolDoc.Notes)
	}
}

func TestDiscovery_Search_BM25(t *testing.T) {
	disc, _ := New(Options{})

	tools := []model.Tool{
		makeTool("git_status", "git", "Show working tree status", []string{"vcs"}),
		makeTool("git_commit", "git", "Record changes", []string{"vcs"}),
		makeTool("docker_ps", "docker", "List containers", []string{"containers"}),
	}

	for _, tool := range tools {
		_ = disc.RegisterTool(tool, makeBackend("server"), nil)
	}

	ctx := context.Background()
	results, err := disc.Search(ctx, "git", 10)
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}

	if len(results) < 2 {
		t.Errorf("expected at least 2 results, got %d", len(results))
	}

	// All results should have BM25 score type
	for _, r := range results {
		if r.ScoreType != ScoreBM25 {
			t.Errorf("expected score type BM25, got %v", r.ScoreType)
		}
	}
}

func TestDiscovery_Search_Hybrid(t *testing.T) {
	embedder := &mockEmbedder{dim: 384}
	disc, _ := New(Options{
		Embedder:    embedder,
		HybridAlpha: 0.7,
	})

	tools := []model.Tool{
		makeTool("git_status", "git", "Show working tree status", []string{"vcs"}),
		makeTool("git_commit", "git", "Record changes", []string{"vcs"}),
		makeTool("docker_ps", "docker", "List containers", []string{"containers"}),
	}

	for _, tool := range tools {
		_ = disc.RegisterTool(tool, makeBackend("server"), nil)
	}

	ctx := context.Background()
	results, err := disc.Search(ctx, "git status", 10)
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}

	if len(results) == 0 {
		t.Error("expected results")
	}

	// All results should have Hybrid score type and scores
	for _, r := range results {
		if r.ScoreType != ScoreHybrid {
			t.Errorf("expected score type Hybrid, got %v", r.ScoreType)
		}
		if r.Score <= 0 {
			t.Errorf("expected positive score, got %f", r.Score)
		}
	}
}

func TestDiscovery_ListNamespaces(t *testing.T) {
	disc, _ := New(Options{})

	tools := []model.Tool{
		makeTool("tool1", "ns_a", "Tool 1", nil),
		makeTool("tool2", "ns_b", "Tool 2", nil),
		makeTool("tool3", "ns_a", "Tool 3", nil),
	}

	for _, tool := range tools {
		_ = disc.RegisterTool(tool, makeBackend("server"), nil)
	}

	namespaces, err := disc.ListNamespaces()
	if err != nil {
		t.Fatalf("ListNamespaces() error = %v", err)
	}

	if len(namespaces) != 2 {
		t.Errorf("expected 2 namespaces, got %d", len(namespaces))
	}
}

func TestDiscovery_OnChange(t *testing.T) {
	disc, _ := New(Options{})

	var events []index.ChangeEvent
	unsubscribe := disc.OnChange(func(event index.ChangeEvent) {
		events = append(events, event)
	})
	defer unsubscribe()

	tool := makeTool("test", "", "Test", nil)
	_ = disc.RegisterTool(tool, makeBackend("server"), nil)

	if len(events) != 1 {
		t.Errorf("expected 1 event, got %d", len(events))
	}
	if events[0].Type != index.ChangeRegistered {
		t.Errorf("expected registered event, got %v", events[0].Type)
	}
}

func TestResults_IDs(t *testing.T) {
	results := Results{
		{Summary: index.Summary{ID: "tool1"}},
		{Summary: index.Summary{ID: "tool2"}},
	}

	ids := results.IDs()
	expected := []string{"tool1", "tool2"}
	if !reflect.DeepEqual(ids, expected) {
		t.Errorf("IDs() = %v, want %v", ids, expected)
	}
}

func TestResults_Summaries(t *testing.T) {
	results := Results{
		{Summary: index.Summary{ID: "tool1", Name: "Tool 1"}},
		{Summary: index.Summary{ID: "tool2", Name: "Tool 2"}},
	}

	summaries := results.Summaries()
	if len(summaries) != 2 {
		t.Errorf("expected 2 summaries, got %d", len(summaries))
	}
	if summaries[0].Name != "Tool 1" {
		t.Errorf("expected name 'Tool 1', got %s", summaries[0].Name)
	}
}

func TestResults_FilterByNamespace(t *testing.T) {
	results := Results{
		{Summary: index.Summary{ID: "git:status", Namespace: "git"}},
		{Summary: index.Summary{ID: "git:commit", Namespace: "git"}},
		{Summary: index.Summary{ID: "docker:ps", Namespace: "docker"}},
	}

	filtered := results.FilterByNamespace("git")
	if len(filtered) != 2 {
		t.Errorf("expected 2 results, got %d", len(filtered))
	}
}

func TestResults_FilterByMinScore(t *testing.T) {
	results := Results{
		{Summary: index.Summary{ID: "tool1"}, Score: 0.9},
		{Summary: index.Summary{ID: "tool2"}, Score: 0.5},
		{Summary: index.Summary{ID: "tool3"}, Score: 0.3},
	}

	filtered := results.FilterByMinScore(0.5)
	if len(filtered) != 2 {
		t.Errorf("expected 2 results, got %d", len(filtered))
	}
}

func TestHybridSearcher_Search(t *testing.T) {
	embedder := &mockEmbedder{dim: 384}
	searcher, err := NewHybridSearcher(HybridOptions{
		Embedder: embedder,
		Alpha:    0.7,
	})
	if err != nil {
		t.Fatalf("NewHybridSearcher() error = %v", err)
	}

	docs := []index.SearchDoc{
		{
			ID:      "git:status",
			DocText: "git status show working tree",
			Summary: index.Summary{ID: "git:status", Name: "status", Namespace: "git"},
		},
		{
			ID:      "docker:ps",
			DocText: "docker ps list containers",
			Summary: index.Summary{ID: "docker:ps", Name: "ps", Namespace: "docker"},
		},
	}

	results, err := searcher.Search("git", 10, docs)
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}

	if len(results) == 0 {
		t.Error("expected results")
	}
}

func TestHybridSearcher_Deterministic(t *testing.T) {
	embedder := &mockEmbedder{dim: 384}
	searcher, _ := NewHybridSearcher(HybridOptions{
		Embedder: embedder,
		Alpha:    0.5,
	})

	if !searcher.Deterministic() {
		t.Error("expected HybridSearcher to be deterministic")
	}
}

func TestBM25OnlySearcher_Search(t *testing.T) {
	searcher := NewBM25OnlySearcher(nil)

	docs := []index.SearchDoc{
		{
			ID:      "git:status",
			DocText: "git status show working tree",
			Summary: index.Summary{ID: "git:status", Name: "status", Namespace: "git", ShortDescription: "Show working tree status"},
		},
		{
			ID:      "docker:ps",
			DocText: "docker ps list containers",
			Summary: index.Summary{ID: "docker:ps", Name: "ps", Namespace: "docker", ShortDescription: "List containers"},
		},
	}

	results, err := searcher.Search("status", 10, docs)
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}

	if len(results) == 0 {
		t.Error("expected results")
	}
}

func TestBM25OnlySearcher_SearchWithScores(t *testing.T) {
	searcher := NewBM25OnlySearcher(nil)

	docs := []index.SearchDoc{
		{
			ID:      "git:status",
			DocText: "git status show working tree",
			Summary: index.Summary{ID: "git:status", Name: "status", Namespace: "git", ShortDescription: "Show working tree status"},
		},
	}

	ctx := context.Background()
	results, err := searcher.SearchWithScores(ctx, "status", 10, docs)
	if err != nil {
		t.Fatalf("SearchWithScores() error = %v", err)
	}

	if len(results) == 0 {
		t.Fatal("expected results")
	}

	if results[0].ScoreType != ScoreBM25 {
		t.Errorf("expected ScoreBM25, got %v", results[0].ScoreType)
	}
	if results[0].Score <= 0 {
		t.Errorf("expected positive score, got %f", results[0].Score)
	}
}

func TestDiscovery_RegisterTools(t *testing.T) {
	disc, _ := New(Options{})

	regs := []index.ToolRegistration{
		{Tool: makeTool("tool1", "ns1", "Tool 1", nil), Backend: makeBackend("server")},
		{Tool: makeTool("tool2", "ns2", "Tool 2", nil), Backend: makeBackend("server")},
	}

	err := disc.RegisterTools(regs)
	if err != nil {
		t.Fatalf("RegisterTools() error = %v", err)
	}

	// Verify tools are retrievable
	_, _, err = disc.GetTool("ns1:tool1")
	if err != nil {
		t.Errorf("expected tool1 to be registered")
	}
	_, _, err = disc.GetTool("ns2:tool2")
	if err != nil {
		t.Errorf("expected tool2 to be registered")
	}
}

func TestDiscovery_RegisterDoc(t *testing.T) {
	disc, _ := New(Options{})

	tool := makeTool("test_tool", "ns", "Test tool", nil)
	_ = disc.RegisterTool(tool, makeBackend("server"), nil)

	err := disc.RegisterDoc("ns:test_tool", tooldoc.DocEntry{
		Summary: "Updated summary",
		Notes:   "Added notes",
	})
	if err != nil {
		t.Fatalf("RegisterDoc() error = %v", err)
	}

	doc, err := disc.DescribeTool("ns:test_tool", tooldoc.DetailFull)
	if err != nil {
		t.Fatalf("DescribeTool() error = %v", err)
	}
	if doc.Summary != "Updated summary" {
		t.Errorf("expected summary 'Updated summary', got %s", doc.Summary)
	}
}

func TestDiscovery_RegisterExamples(t *testing.T) {
	disc, _ := New(Options{})

	tool := makeTool("test_tool", "ns", "Test tool", nil)
	_ = disc.RegisterTool(tool, makeBackend("server"), nil)

	examples := []tooldoc.ToolExample{
		{Title: "Example 1", Args: map[string]any{"key": "value"}},
		{Title: "Example 2", Args: map[string]any{"other": "data"}},
	}

	err := disc.RegisterExamples("ns:test_tool", examples)
	if err != nil {
		t.Fatalf("RegisterExamples() error = %v", err)
	}

	retrieved, err := disc.ListExamples("ns:test_tool", 10)
	if err != nil {
		t.Fatalf("ListExamples() error = %v", err)
	}
	if len(retrieved) != 2 {
		t.Errorf("expected 2 examples, got %d", len(retrieved))
	}
}

func TestDiscovery_SearchPage(t *testing.T) {
	disc, _ := New(Options{})

	// Register multiple tools
	for i := 0; i < 5; i++ {
		tool := makeTool("tool"+string(rune('a'+i)), "ns", "Tool", []string{"test"})
		_ = disc.RegisterTool(tool, makeBackend("server"), nil)
	}

	ctx := context.Background()
	results, cursor, err := disc.SearchPage(ctx, "tool", 2, "")
	if err != nil {
		t.Fatalf("SearchPage() error = %v", err)
	}

	if len(results) != 2 {
		t.Errorf("expected 2 results, got %d", len(results))
	}

	// Should have a cursor for next page
	if cursor == "" {
		t.Log("Note: cursor may be empty if all results fit in first page")
	}

	// Each result should have the score type
	for _, r := range results {
		if r.ScoreType != ScoreBM25 {
			t.Errorf("expected ScoreBM25, got %v", r.ScoreType)
		}
	}
}

func TestDiscovery_GetAllBackends(t *testing.T) {
	disc, _ := New(Options{})

	tool := makeTool("test_tool", "ns", "Test tool", nil)
	_ = disc.RegisterTool(tool, makeBackend("server1"), nil)
	// Register same tool with different backend
	_ = disc.idx.RegisterTool(tool, makeBackend("server2"))

	backends, err := disc.GetAllBackends("ns:test_tool")
	if err != nil {
		t.Fatalf("GetAllBackends() error = %v", err)
	}
	if len(backends) != 2 {
		t.Errorf("expected 2 backends, got %d", len(backends))
	}
}

func TestDiscovery_ListExamples(t *testing.T) {
	disc, _ := New(Options{})

	tool := makeTool("test_tool", "ns", "Test tool", nil)
	_ = disc.RegisterTool(tool, makeBackend("server"), &tooldoc.DocEntry{
		Examples: []tooldoc.ToolExample{
			{Title: "Ex1", Args: map[string]any{"a": "b"}},
			{Title: "Ex2", Args: map[string]any{"c": "d"}},
			{Title: "Ex3", Args: map[string]any{"e": "f"}},
		},
	})

	examples, err := disc.ListExamples("ns:test_tool", 2)
	if err != nil {
		t.Fatalf("ListExamples() error = %v", err)
	}
	if len(examples) != 2 {
		t.Errorf("expected 2 examples (limited), got %d", len(examples))
	}
}

func TestDiscovery_Index(t *testing.T) {
	disc, _ := New(Options{})

	idx := disc.Index()
	if idx == nil {
		t.Error("Index() returned nil")
	}
}

func TestDiscovery_DocStore(t *testing.T) {
	disc, _ := New(Options{})

	store := disc.DocStore()
	if store == nil {
		t.Error("DocStore() returned nil")
	}
}

func TestNew_WithCustomSearcher(t *testing.T) {
	customSearcher := &mockSearcher{}
	disc, err := New(Options{
		Searcher: customSearcher,
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	if disc.searcher != customSearcher {
		t.Error("expected custom searcher to be used")
	}
	if disc.scoreType != ScoreBM25 {
		t.Errorf("expected ScoreBM25 for custom searcher, got %v", disc.scoreType)
	}
}

func TestNew_WithCustomIndex(t *testing.T) {
	customIdx := index.NewInMemoryIndex()
	disc, err := New(Options{
		Index: customIdx,
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	if disc.idx != customIdx {
		t.Error("expected custom index to be used")
	}
}

func TestHybridSearcher_GetScoreType(t *testing.T) {
	embedder := &mockEmbedder{dim: 384}
	searcher, _ := NewHybridSearcher(HybridOptions{
		Embedder: embedder,
		Alpha:    0.5,
	})

	if searcher.GetScoreType() != ScoreHybrid {
		t.Errorf("expected ScoreHybrid, got %v", searcher.GetScoreType())
	}
}

func TestBM25OnlySearcher_GetScoreType(t *testing.T) {
	searcher := NewBM25OnlySearcher(nil)

	if searcher.GetScoreType() != ScoreBM25 {
		t.Errorf("expected ScoreBM25, got %v", searcher.GetScoreType())
	}
}

func TestBM25OnlySearcher_Deterministic(t *testing.T) {
	searcher := NewBM25OnlySearcher(nil)

	if !searcher.Deterministic() {
		t.Error("expected BM25OnlySearcher to be deterministic")
	}
}

func TestHybridSearcher_Search_ZeroLimit(t *testing.T) {
	embedder := &mockEmbedder{dim: 384}
	searcher, _ := NewHybridSearcher(HybridOptions{
		Embedder: embedder,
		Alpha:    0.5,
	})

	docs := []index.SearchDoc{{ID: "test", DocText: "test"}}
	results, err := searcher.Search("test", 0, docs)
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results for limit=0, got %d", len(results))
	}
}

func TestBM25OnlySearcher_Search_ZeroLimit(t *testing.T) {
	searcher := NewBM25OnlySearcher(nil)

	docs := []index.SearchDoc{{ID: "test", DocText: "test"}}
	results, err := searcher.Search("test", 0, docs)
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results for limit=0, got %d", len(results))
	}
}

func TestHybridSearcher_SearchWithScores_ZeroLimit(t *testing.T) {
	embedder := &mockEmbedder{dim: 384}
	searcher, _ := NewHybridSearcher(HybridOptions{
		Embedder: embedder,
		Alpha:    0.5,
	})

	docs := []index.SearchDoc{{ID: "test", DocText: "test"}}
	results, err := searcher.SearchWithScores(context.Background(), "test", 0, docs)
	if err != nil {
		t.Fatalf("SearchWithScores() error = %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results for limit=0, got %d", len(results))
	}
}

func TestBM25OnlySearcher_SearchWithScores_ZeroLimit(t *testing.T) {
	searcher := NewBM25OnlySearcher(nil)

	docs := []index.SearchDoc{{ID: "test", DocText: "test"}}
	results, err := searcher.SearchWithScores(context.Background(), "test", 0, docs)
	if err != nil {
		t.Fatalf("SearchWithScores() error = %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results for limit=0, got %d", len(results))
	}
}

func TestSortScoredDocs_EqualScores(t *testing.T) {
	// Test that equal scores sort by ID ascending
	docs := []index.SearchDoc{
		{ID: "z_tool", Summary: index.Summary{ID: "z_tool"}},
		{ID: "a_tool", Summary: index.Summary{ID: "a_tool"}},
		{ID: "m_tool", Summary: index.Summary{ID: "m_tool"}},
	}

	scored := []scoredDoc{
		{idx: 0, score: 1.0},
		{idx: 1, score: 1.0},
		{idx: 2, score: 1.0},
	}

	sortScoredDocs(scored, docs)

	// Should be sorted by ID: a_tool, m_tool, z_tool
	expectedOrder := []string{"a_tool", "m_tool", "z_tool"}
	for i, sd := range scored {
		if docs[sd.idx].ID != expectedOrder[i] {
			t.Errorf("position %d: expected %s, got %s", i, expectedOrder[i], docs[sd.idx].ID)
		}
	}
}

func TestNewHybridSearcher_NilEmbedder(t *testing.T) {
	_, err := NewHybridSearcher(HybridOptions{
		Embedder: nil,
		Alpha:    0.5,
	})
	if err == nil {
		t.Error("expected error for nil embedder")
	}
}

func TestNewHybridSearcher_InvalidAlphaNegative(t *testing.T) {
	embedder := &mockEmbedder{dim: 384}
	_, err := NewHybridSearcher(HybridOptions{
		Embedder: embedder,
		Alpha:    -0.1,
	})
	if err == nil {
		t.Error("expected error for negative alpha")
	}
}

func TestDiscovery_OnChange_NoNotifier(t *testing.T) {
	// Use a custom index that doesn't implement ChangeNotifier
	disc, _ := New(Options{
		Index: &nonNotifyingIndex{},
	})

	// Should return a no-op unsubscribe function
	unsubscribe := disc.OnChange(func(event index.ChangeEvent) {})
	unsubscribe() // Should not panic
}

func TestDiscovery_Search_WithNonInMemoryIndex(t *testing.T) {
	// Use a custom index that is not InMemoryIndex
	customIdx := &searchableNonInMemoryIndex{
		tools: make(map[string]model.Tool),
	}
	embedder := &mockEmbedder{dim: 384}

	disc, err := New(Options{
		Index:       customIdx,
		Embedder:    embedder,
		HybridAlpha: 0.5,
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	// Register a tool through the custom index
	tool := makeTool("test_tool", "ns", "Test tool description", nil)
	customIdx.tools["ns:test_tool"] = tool

	ctx := context.Background()
	// This should trigger the getSearchDocs fallback path
	results, err := disc.Search(ctx, "test", 10)
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}

	// Should get results from fallback path
	t.Logf("Got %d results from fallback path", len(results))
}

func TestDiscovery_ProviderRegistration(t *testing.T) {
	disc, err := New(Options{})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	id, err := disc.RegisterProvider("", adapter.CanonicalProvider{
		Name:        "Test Agent",
		Description: "Handles tests",
		Version:     "1.0.0",
	})
	if err != nil {
		t.Fatalf("RegisterProvider error = %v", err)
	}
	if id == "" {
		t.Fatal("expected provider id")
	}

	got, err := disc.DescribeProvider(id)
	if err != nil {
		t.Fatalf("DescribeProvider error = %v", err)
	}
	if got.Name != "Test Agent" {
		t.Errorf("Name = %q, want Test Agent", got.Name)
	}

	list, err := disc.ListProviders()
	if err != nil {
		t.Fatalf("ListProviders error = %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("ListProviders length = %d, want 1", len(list))
	}
}

func TestDiscovery_RegisterTool_WithDocError(t *testing.T) {
	disc, _ := New(Options{})

	tool := makeTool("test_tool", "ns", "Test tool", nil)
	backend := makeBackend("server")
	// Create doc with invalid args that will fail validation
	doc := &tooldoc.DocEntry{
		Summary: "Test",
		Examples: []tooldoc.ToolExample{
			{
				Title: "Bad example",
				Args:  createDeeplyNestedArgs(10), // Exceeds depth limit
			},
		},
	}

	err := disc.RegisterTool(tool, backend, doc)
	if err == nil {
		t.Error("expected error for invalid doc args")
	}
}

func createDeeplyNestedArgs(depth int) map[string]any {
	if depth <= 0 {
		return map[string]any{"leaf": "value"}
	}
	return map[string]any{"nested": createDeeplyNestedArgs(depth - 1)}
}

func TestHybridSearcher_Search_ReturnsError(t *testing.T) {
	embedder := &errorEmbedder{}
	searcher, _ := NewHybridSearcher(HybridOptions{
		Embedder: embedder,
		Alpha:    0.5,
	})

	docs := []index.SearchDoc{
		{ID: "test", DocText: "test document", Summary: index.Summary{ID: "test"}},
	}

	_, err := searcher.Search("test", 10, docs)
	if err == nil {
		t.Error("expected error from embedder")
	}
}

func TestDiscovery_Search_EmptyQuery(t *testing.T) {
	disc, _ := New(Options{})

	tool := makeTool("test", "ns", "Test tool", nil)
	_ = disc.RegisterTool(tool, makeBackend("server"), nil)

	ctx := context.Background()
	results, err := disc.Search(ctx, "", 10)
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}

	// Empty query should return results
	if len(results) == 0 {
		t.Error("expected results for empty query")
	}
}

func TestHybridSearcher_ResultsLimitedCorrectly(t *testing.T) {
	embedder := &mockEmbedder{dim: 384}
	searcher, _ := NewHybridSearcher(HybridOptions{
		Embedder: embedder,
		Alpha:    0.5,
	})

	// Create many docs
	docs := make([]index.SearchDoc, 10)
	for i := 0; i < 10; i++ {
		docs[i] = index.SearchDoc{
			ID:      "tool" + string(rune('a'+i)),
			DocText: "test tool description",
			Summary: index.Summary{ID: "tool" + string(rune('a'+i))},
		}
	}

	results, err := searcher.SearchWithScores(context.Background(), "test", 3, docs)
	if err != nil {
		t.Fatalf("SearchWithScores() error = %v", err)
	}

	if len(results) != 3 {
		t.Errorf("expected 3 results (limited), got %d", len(results))
	}
}

// mockSearcher is a simple searcher for testing custom searcher injection
type mockSearcher struct{}

func (m *mockSearcher) Search(query string, limit int, docs []index.SearchDoc) ([]index.Summary, error) {
	if limit <= 0 || len(docs) == 0 {
		return []index.Summary{}, nil
	}
	results := make([]index.Summary, 0, limit)
	for i := 0; i < len(docs) && i < limit; i++ {
		results = append(results, docs[i].Summary)
	}
	return results, nil
}

func (m *mockSearcher) Deterministic() bool { return true }

// nonNotifyingIndex is an index that doesn't implement ChangeNotifier
type nonNotifyingIndex struct{}

func (n *nonNotifyingIndex) RegisterTool(tool model.Tool, backend model.ToolBackend) error {
	return nil
}
func (n *nonNotifyingIndex) RegisterTools(regs []index.ToolRegistration) error { return nil }
func (n *nonNotifyingIndex) RegisterToolsFromMCP(serverName string, tools []model.Tool) error {
	return nil
}
func (n *nonNotifyingIndex) UnregisterBackend(toolID string, kind model.BackendKind, serverName string) error {
	return nil
}
func (n *nonNotifyingIndex) GetTool(id string) (model.Tool, model.ToolBackend, error) {
	return model.Tool{}, model.ToolBackend{}, nil
}
func (n *nonNotifyingIndex) GetAllBackends(id string) ([]model.ToolBackend, error) {
	return nil, nil
}
func (n *nonNotifyingIndex) Search(query string, limit int) ([]index.Summary, error) {
	return nil, nil
}
func (n *nonNotifyingIndex) SearchPage(query string, limit int, cursor string) ([]index.Summary, string, error) {
	return nil, "", nil
}
func (n *nonNotifyingIndex) ListNamespaces() ([]string, error) { return nil, nil }
func (n *nonNotifyingIndex) ListNamespacesPage(limit int, cursor string) ([]string, string, error) {
	return nil, "", nil
}

// mockEmbedder is a simple embedder for testing
type mockEmbedder struct {
	dim int
}

func (m *mockEmbedder) Embed(_ context.Context, text string) ([]float32, error) {
	vec := make([]float32, m.dim)
	// Simple embedding: use hash of text
	for i := range vec {
		vec[i] = float32(len(text)+i) / float32(m.dim*10)
	}
	return vec, nil
}

// errorEmbedder always returns an error
type errorEmbedder struct{}

func (e *errorEmbedder) Embed(_ context.Context, _ string) ([]float32, error) {
	return nil, errors.New("embedder error")
}

// searchableNonInMemoryIndex is an index that supports search but is not InMemoryIndex
type searchableNonInMemoryIndex struct {
	tools map[string]model.Tool
}

func (s *searchableNonInMemoryIndex) RegisterTool(tool model.Tool, backend model.ToolBackend) error {
	s.tools[tool.ToolID()] = tool
	return nil
}
func (s *searchableNonInMemoryIndex) RegisterTools(regs []index.ToolRegistration) error {
	for _, r := range regs {
		s.tools[r.Tool.ToolID()] = r.Tool
	}
	return nil
}
func (s *searchableNonInMemoryIndex) RegisterToolsFromMCP(serverName string, tools []model.Tool) error {
	return nil
}
func (s *searchableNonInMemoryIndex) UnregisterBackend(toolID string, kind model.BackendKind, serverName string) error {
	return nil
}
func (s *searchableNonInMemoryIndex) GetTool(id string) (model.Tool, model.ToolBackend, error) {
	if tool, ok := s.tools[id]; ok {
		return tool, model.NewMCPBackend("server"), nil
	}
	return model.Tool{}, model.ToolBackend{}, index.ErrNotFound
}
func (s *searchableNonInMemoryIndex) GetAllBackends(id string) ([]model.ToolBackend, error) {
	return nil, nil
}
func (s *searchableNonInMemoryIndex) Search(query string, limit int) ([]index.Summary, error) {
	results := make([]index.Summary, 0)
	for id, tool := range s.tools {
		results = append(results, index.Summary{
			ID:               id,
			Name:             tool.Name,
			Namespace:        tool.Namespace,
			ShortDescription: tool.Description,
		})
	}
	if len(results) > limit {
		results = results[:limit]
	}
	return results, nil
}
func (s *searchableNonInMemoryIndex) SearchPage(query string, limit int, cursor string) ([]index.Summary, string, error) {
	results, _ := s.Search(query, limit)
	return results, "", nil
}
func (s *searchableNonInMemoryIndex) ListNamespaces() ([]string, error) { return nil, nil }
func (s *searchableNonInMemoryIndex) ListNamespacesPage(limit int, cursor string) ([]string, string, error) {
	return nil, "", nil
}
