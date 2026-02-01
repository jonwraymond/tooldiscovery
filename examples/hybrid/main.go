// Package main demonstrates hybrid search combining BM25 and semantic search.
//
// This example shows how to:
//   - Use the discovery package's HybridSearcher
//   - Configure the BM25/embedding weight ratio (alpha)
//   - Get search results with detailed score information
//   - Compare results across different alpha values
//
// Run with: go run ./examples/hybrid
package main

import (
	"context"
	"fmt"
	"math"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/jonwraymond/tooldiscovery/discovery"
	"github.com/jonwraymond/tooldiscovery/index"
	"github.com/jonwraymond/toolfoundation/model"
)

func main() {
	ctx := context.Background()

	// Create sample search documents
	docs := createSearchDocs()
	fmt.Printf("Loaded %d documents\n\n", len(docs))

	// Test different alpha values
	alphas := []float64{0.0, 0.3, 0.5, 0.7, 1.0}
	query := "show working status"

	fmt.Printf("Query: %q\n\n", query)

	for _, alpha := range alphas {
		fmt.Printf("=== Alpha: %.1f (%.0f%% BM25, %.0f%% Embedding) ===\n",
			alpha, alpha*100, (1-alpha)*100)

		embedder := &simpleEmbedder{}
		searcher, err := discovery.NewHybridSearcher(discovery.HybridOptions{
			Embedder: embedder,
			Alpha:    alpha,
		})
		if err != nil {
			fmt.Printf("Error creating searcher: %v\n", err)
			continue
		}

		results, err := searcher.SearchWithScores(ctx, query, 5, docs)
		if err != nil {
			fmt.Printf("Error searching: %v\n", err)
			continue
		}

		for i, r := range results {
			fmt.Printf("  %d. [%.3f] %s\n", i+1, r.Score, r.Summary.ID)
		}
		fmt.Println()
	}

	// Demonstrate the facade's built-in hybrid search
	fmt.Println("=== Using Discovery Facade with Hybrid Search ===")
	disc, err := discovery.New(discovery.Options{
		Embedder:    &simpleEmbedder{},
		HybridAlpha: 0.6, // 60% BM25, 40% semantic
	})
	if err != nil {
		fmt.Printf("Error creating discovery: %v\n", err)
		return
	}

	// Register tools
	for _, doc := range docs {
		tool := docToTool(doc)
		_ = disc.RegisterTool(tool, mcpBackend("example-server"), nil)
	}

	// Search
	results, _ := disc.Search(ctx, "container management", 5)
	for i, r := range results {
		fmt.Printf("  %d. [%s %.3f] %s: %s\n",
			i+1, r.ScoreType, r.Score, r.Summary.ID, r.Summary.ShortDescription)
	}
}

func createSearchDocs() []index.SearchDoc {
	items := []struct {
		id, name, ns, desc string
		tags               []string
	}{
		{"git:status", "status", "git", "Show the working tree status", []string{"vcs"}},
		{"git:commit", "commit", "git", "Record changes to the repository", []string{"vcs"}},
		{"git:diff", "diff", "git", "Show changes between commits", []string{"vcs"}},
		{"docker:ps", "ps", "docker", "List running containers", []string{"containers", "devops"}},
		{"docker:run", "run", "docker", "Run a command in a new container", []string{"containers", "devops"}},
		{"docker:build", "build", "docker", "Build an image from Dockerfile", []string{"containers", "devops"}},
		{"kubectl:get", "get", "kubectl", "Display Kubernetes resources", []string{"k8s", "devops"}},
		{"kubectl:apply", "apply", "kubectl", "Apply configuration to cluster", []string{"k8s", "devops"}},
	}

	docs := make([]index.SearchDoc, len(items))
	for i, item := range items {
		docs[i] = index.SearchDoc{
			ID:      item.id,
			DocText: fmt.Sprintf("%s %s %s %s", item.name, item.ns, item.desc, strings.Join(item.tags, " ")),
			Summary: index.Summary{
				ID:               item.id,
				Name:             item.name,
				Namespace:        item.ns,
				ShortDescription: item.desc,
				Tags:             item.tags,
			},
		}
	}
	return docs
}

// simpleEmbedder creates embeddings based on word presence.
type simpleEmbedder struct{}

func (e *simpleEmbedder) Embed(_ context.Context, text string) ([]float32, error) {
	keywords := []string{
		"git", "commit", "status", "diff", "changes", "working", "tree", "show",
		"docker", "container", "run", "build", "image", "list",
		"kubernetes", "k8s", "kubectl", "cluster", "apply", "resources",
		"devops", "management", "display",
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

// Helper functions for Discovery facade example

func docToTool(doc index.SearchDoc) model.Tool {
	return model.Tool{
		Tool: mcp.Tool{
			Name:        doc.Summary.Name,
			Description: doc.Summary.ShortDescription,
			InputSchema: map[string]any{"type": "object"},
		},
		Namespace: doc.Summary.Namespace,
		Tags:      doc.Summary.Tags,
	}
}

func mcpBackend(serverName string) model.ToolBackend {
	return model.NewMCPBackend(serverName)
}
