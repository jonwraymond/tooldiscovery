// Package main demonstrates semantic search with embeddings.
//
// This example shows how to:
//   - Use the semantic package directly for embedding-based search
//   - Implement a custom Embedder interface
//   - Filter results by namespace or tags
//   - Score documents using different strategies
//
// Run with: go run ./examples/semantic
package main

import (
	"context"
	"fmt"
	"math"
	"strings"

	"github.com/jonwraymond/tooldiscovery/semantic"
)

func main() {
	ctx := context.Background()

	// 1. Create a semantic index
	idx := semantic.NewInMemoryIndex()

	// 2. Add tool documents
	docs := createSampleDocs()
	for _, doc := range docs {
		_ = idx.Add(ctx, doc)
	}
	fmt.Printf("Indexed %d documents\n\n", len(docs))

	// 3. Create search strategies
	bm25Strategy := semantic.NewBM25Strategy(nil) // Default scorer
	embedder := &simpleEmbedder{}
	embeddingStrategy := semantic.NewEmbeddingStrategy(embedder)

	// 4. Search with BM25 (lexical)
	fmt.Println("=== BM25 Search: 'kubernetes pods' ===")
	searcher := semantic.NewSearcher(idx, bm25Strategy)
	results, _ := searcher.Search(ctx, "kubernetes pods")
	for _, r := range results {
		fmt.Printf("  [%.2f] %s: %s\n", r.Score, r.Document.ID, r.Document.Description)
	}

	// 5. Search with embeddings (semantic)
	fmt.Println("\n=== Embedding Search: 'container management' ===")
	searcher = semantic.NewSearcher(idx, embeddingStrategy)
	results, _ = searcher.Search(ctx, "container management")
	for _, r := range results {
		fmt.Printf("  [%.2f] %s: %s\n", r.Score, r.Document.ID, r.Document.Description)
	}

	// 6. Use hybrid strategy
	fmt.Println("\n=== Hybrid Search (70% BM25, 30% Embedding): 'version control' ===")
	hybridStrategy, _ := semantic.NewHybridStrategy(bm25Strategy, embeddingStrategy, 0.7)
	searcher = semantic.NewSearcher(idx, hybridStrategy)
	results, _ = searcher.Search(ctx, "version control")
	for _, r := range results {
		fmt.Printf("  [%.2f] %s: %s\n", r.Score, r.Document.ID, r.Document.Description)
	}

	// 7. Filter by namespace
	fmt.Println("\n=== Filter by Namespace: 'docker' ===")
	allDocs := idx.List(ctx)
	dockerDocs := semantic.FilterByNamespace(allDocs, "docker")
	for _, doc := range dockerDocs {
		fmt.Printf("  - %s: %s\n", doc.ID, doc.Description)
	}

	// 8. Filter by tags
	fmt.Println("\n=== Filter by Tags: ['devops'] ===")
	devopsDocs := semantic.FilterByTags(allDocs, []string{"devops"})
	for _, doc := range devopsDocs {
		fmt.Printf("  - %s (tags: %v)\n", doc.ID, doc.Tags)
	}
}

func createSampleDocs() []semantic.Document {
	return []semantic.Document{
		{
			ID:          "git:status",
			Namespace:   "git",
			Name:        "status",
			Description: "Show the working tree status",
			Tags:        []string{"vcs", "version-control"},
			Category:    "vcs",
		},
		{
			ID:          "git:commit",
			Namespace:   "git",
			Name:        "commit",
			Description: "Record changes to the repository",
			Tags:        []string{"vcs", "version-control"},
			Category:    "vcs",
		},
		{
			ID:          "docker:ps",
			Namespace:   "docker",
			Name:        "ps",
			Description: "List running containers",
			Tags:        []string{"containers", "devops"},
			Category:    "containers",
		},
		{
			ID:          "docker:run",
			Namespace:   "docker",
			Name:        "run",
			Description: "Run a command in a new container",
			Tags:        []string{"containers", "devops"},
			Category:    "containers",
		},
		{
			ID:          "kubectl:get",
			Namespace:   "kubectl",
			Name:        "get",
			Description: "Display Kubernetes resources like pods and services",
			Tags:        []string{"kubernetes", "k8s", "devops"},
			Category:    "orchestration",
		},
		{
			ID:          "kubectl:apply",
			Namespace:   "kubectl",
			Name:        "apply",
			Description: "Apply configuration to Kubernetes cluster",
			Tags:        []string{"kubernetes", "k8s", "devops"},
			Category:    "orchestration",
		},
	}
}

// simpleEmbedder creates embeddings based on word presence.
// In production, use a real embedding model (e.g., OpenAI, Ollama, etc.)
type simpleEmbedder struct{}

func (e *simpleEmbedder) Embed(_ context.Context, text string) ([]float32, error) {
	// Simple keyword-based embedding for demonstration
	keywords := []string{
		"git", "commit", "status", "version", "control",
		"docker", "container", "run", "image",
		"kubernetes", "k8s", "pods", "deploy", "cluster",
		"devops", "management", "list", "show",
	}

	vec := make([]float32, len(keywords))
	textLower := strings.ToLower(text)

	for i, kw := range keywords {
		if strings.Contains(textLower, kw) {
			vec[i] = 1.0
		}
	}

	// Normalize
	var norm float64
	for _, v := range vec {
		norm += float64(v * v)
	}
	if norm > 0 {
		norm = math.Sqrt(norm)
		for i := range vec {
			vec[i] = float32(float64(vec[i]) / norm)
		}
	}

	return vec, nil
}
