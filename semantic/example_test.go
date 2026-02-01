package semantic_test

import (
	"context"
	"fmt"

	"github.com/jonwraymond/tooldiscovery/index"
	"github.com/jonwraymond/tooldiscovery/semantic"
)

func ExampleNewSearcher() {
	// Create an index and add documents
	idx := semantic.NewInMemoryIndex()
	ctx := context.Background()

	docs := []semantic.Document{
		{ID: "git:status", Name: "status", Namespace: "git", Description: "Show working tree status", Tags: []string{"vcs"}},
		{ID: "git:commit", Name: "commit", Namespace: "git", Description: "Record changes to repository", Tags: []string{"vcs"}},
		{ID: "docker:ps", Name: "ps", Namespace: "docker", Description: "List containers", Tags: []string{"containers"}},
	}

	for _, doc := range docs {
		_ = idx.Add(ctx, doc)
	}

	// Create a searcher with BM25 strategy
	strategy := semantic.NewBM25Strategy(nil) // nil uses default scorer
	searcher := semantic.NewSearcher(idx, strategy)

	// Search for git-related tools
	results, _ := searcher.Search(ctx, "git status")
	fmt.Println("Found:", len(results), "results")
	if len(results) > 0 {
		fmt.Println("Top result:", results[0].Document.ID)
	}
	// Output:
	// Found: 3 results
	// Top result: git:status
}

func ExampleNewHybridStrategy() {
	// Create mock embedder for demonstration
	embedder := &mockEmbedder{}

	// Create individual strategies
	bm25 := semantic.NewBM25Strategy(nil)
	embedding := semantic.NewEmbeddingStrategy(embedder)

	// Combine with hybrid strategy (0.7 = 70% BM25, 30% embedding)
	hybrid, err := semantic.NewHybridStrategy(bm25, embedding, 0.7)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	// Use the hybrid strategy
	doc := semantic.Document{
		ID:          "test",
		Name:        "search_files",
		Description: "Search for files in the filesystem",
	}

	ctx := context.Background()
	score, _ := hybrid.Score(ctx, "search files", doc.Normalized())
	fmt.Printf("Hybrid score: %.2f\n", score)
	// Output:
	// Hybrid score: 1.70
}

// mockEmbedder is a simple embedder for examples
type mockEmbedder struct{}

func (m *mockEmbedder) Embed(_ context.Context, _ string) ([]float32, error) {
	// Return a simple unit vector for demonstration
	return []float32{1.0, 0.0, 0.0}, nil
}

func ExampleInMemoryIndex() {
	idx := semantic.NewInMemoryIndex()
	ctx := context.Background()

	// Add documents
	doc := semantic.Document{
		ID:          "files:search",
		Name:        "search",
		Namespace:   "files",
		Description: "Search for files matching a pattern",
		Tags:        []string{"search", "filesystem"},
	}
	_ = idx.Add(ctx, doc)

	// Retrieve document
	retrieved, found := idx.Get(ctx, "files:search")
	fmt.Println("Found:", found)
	fmt.Println("Name:", retrieved.Name)

	// List all documents
	all := idx.List(ctx)
	fmt.Println("Total documents:", len(all))
	// Output:
	// Found: true
	// Name: search
	// Total documents: 1
}

func ExampleDocument_Normalized() {
	doc := semantic.Document{
		ID:          "example",
		Name:        "Search Files",
		Description: "Find files matching a pattern",
		Tags:        []string{"Search", "FILES", "  filesystem  "},
	}

	normalized := doc.Normalized()
	fmt.Println("Tags:", normalized.Tags)
	fmt.Println("Text:", normalized.Text)
	// Output:
	// Tags: [files filesystem search]
	// Text: Search Files Find files matching a pattern files filesystem search
}

func ExampleFilterByNamespace() {
	docs := []semantic.Document{
		{ID: "git:status", Namespace: "git"},
		{ID: "git:commit", Namespace: "git"},
		{ID: "docker:ps", Namespace: "docker"},
	}

	filtered := semantic.FilterByNamespace(docs, "git")
	fmt.Println("Git documents:", len(filtered))
	// Output:
	// Git documents: 2
}

func ExampleFilterByTags() {
	docs := []semantic.Document{
		{ID: "tool1", Tags: []string{"vcs", "git"}},
		{ID: "tool2", Tags: []string{"containers", "docker"}},
		{ID: "tool3", Tags: []string{"vcs", "svn"}},
	}

	// Find all documents with "vcs" tag
	filtered := semantic.FilterByTags(docs, []string{"vcs"})
	fmt.Println("VCS documents:", len(filtered))
	// Output:
	// VCS documents: 2
}

func ExampleDocumentFromSearchDoc() {
	// Convert from index.SearchDoc to semantic.Document
	searchDoc := index.SearchDoc{
		ID:      "github:create-issue",
		DocText: "create-issue github create issue tracker",
		Summary: index.Summary{
			ID:               "github:create-issue",
			Name:             "create-issue",
			Namespace:        "github",
			ShortDescription: "Create a new GitHub issue",
			Tags:             []string{"github", "issues"},
		},
	}

	doc := semantic.DocumentFromSearchDoc(searchDoc)
	fmt.Println("ID:", doc.ID)
	fmt.Println("Name:", doc.Name)
	fmt.Println("Namespace:", doc.Namespace)
	// Output:
	// ID: github:create-issue
	// Name: create-issue
	// Namespace: github
}

func ExampleSearchDocFromDocument() {
	// Convert from semantic.Document to index.SearchDoc
	doc := semantic.Document{
		ID:          "slack:send-message",
		Name:        "send-message",
		Namespace:   "slack",
		Description: "Send a message to a Slack channel",
		Tags:        []string{"slack", "messaging"},
		Text:        "send-message slack send a message slack messaging",
	}

	searchDoc := semantic.SearchDocFromDocument(doc)
	fmt.Println("ID:", searchDoc.ID)
	fmt.Println("Summary Name:", searchDoc.Summary.Name)
	fmt.Println("DocText:", searchDoc.DocText)
	// Output:
	// ID: slack:send-message
	// Summary Name: send-message
	// DocText: send-message slack send a message slack messaging
}
