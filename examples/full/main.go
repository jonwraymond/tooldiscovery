// Package main demonstrates the complete Discovery facade workflow.
//
// This example shows the recommended way to use tooldiscovery:
//   - Create a Discovery instance (optionally with hybrid search)
//   - Register tools with their documentation in one call
//   - Search using the configured strategy
//   - Use progressive disclosure for documentation
//   - Listen for index changes
//
// Run with: go run ./examples/full
package main

import (
	"context"
	"fmt"
	"log"
	"math"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/jonwraymond/tooldiscovery/discovery"
	"github.com/jonwraymond/tooldiscovery/index"
	"github.com/jonwraymond/tooldiscovery/tooldoc"
	"github.com/jonwraymond/toolfoundation/model"
)

func main() {
	ctx := context.Background()

	// 1. Create Discovery with hybrid search
	fmt.Println("=== Initializing Discovery ===")
	disc, err := discovery.New(discovery.Options{
		Embedder:    &simpleEmbedder{},
		HybridAlpha: 0.7, // 70% BM25, 30% semantic
		MaxExamples: 5,
	})
	if err != nil {
		log.Fatalf("Failed to create discovery: %v", err)
	}
	fmt.Println("Created Discovery with hybrid search (alpha=0.7)")

	// 2. Subscribe to changes
	unsubscribe := disc.OnChange(func(event index.ChangeEvent) {
		fmt.Printf("  [event] %s: %s\n", event.Type, event.ToolID)
	})
	defer unsubscribe()

	// 3. Register tools with documentation
	fmt.Println("\n=== Registering Tools ===")
	registerTools(disc)

	// 4. Search using hybrid strategy
	fmt.Println("\n=== Hybrid Search ===")
	queries := []string{"git status", "run container", "kubernetes"}

	for _, query := range queries {
		fmt.Printf("\nQuery: %q\n", query)
		results, err := disc.Search(ctx, query, 3)
		if err != nil {
			fmt.Printf("  Error: %v\n", err)
			continue
		}

		for _, r := range results {
			fmt.Printf("  [%s %.3f] %s: %s\n",
				r.ScoreType, r.Score, r.Summary.ID, r.Summary.ShortDescription)
		}
	}

	// 5. Filter results
	fmt.Println("\n=== Filtering Results ===")
	allResults, _ := disc.Search(ctx, "devops", 10)
	fmt.Printf("All devops results: %d\n", len(allResults))

	dockerResults := allResults.FilterByNamespace("docker")
	fmt.Printf("Docker namespace only: %d\n", len(dockerResults))

	highScoreResults := allResults.FilterByMinScore(0.5)
	fmt.Printf("Score >= 0.5: %d\n", len(highScoreResults))

	// 6. Progressive disclosure
	fmt.Println("\n=== Progressive Disclosure ===")
	toolID := "git:commit"

	// Summary level (cheap - for search results)
	summary, err := disc.DescribeTool(toolID, tooldoc.DetailSummary)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	} else {
		fmt.Printf("Summary: %s\n", summary.Summary)
	}

	// Schema level (medium - for tool selection)
	schema, _ := disc.DescribeTool(toolID, tooldoc.DetailSchema)
	if schema.SchemaInfo != nil {
		fmt.Printf("Required params: %v\n", schema.SchemaInfo.Required)
	}

	// Full level (expensive - for tool usage)
	full, _ := disc.DescribeTool(toolID, tooldoc.DetailFull)
	fmt.Printf("Notes: %s\n", full.Notes)
	fmt.Printf("Examples: %d\n", len(full.Examples))

	// 7. List examples separately
	fmt.Println("\n=== Examples ===")
	examples, _ := disc.ListExamples(toolID, 3)
	for _, ex := range examples {
		fmt.Printf("  - %s: %v\n", ex.Title, ex.Args)
	}

	// 8. List namespaces
	fmt.Println("\n=== Namespaces ===")
	namespaces, _ := disc.ListNamespaces()
	fmt.Printf("Available: %v\n", namespaces)

	// 9. Direct tool lookup
	fmt.Println("\n=== Direct Lookup ===")
	tool, backend, err := disc.GetTool("docker:run")
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	} else {
		fmt.Printf("Tool: %s (backend: %s)\n", tool.Name, backend.MCP.ServerName)
	}
}

func registerTools(disc *discovery.Discovery) {
	tools := []struct {
		tool    model.Tool
		backend model.ToolBackend
		doc     *tooldoc.DocEntry
	}{
		{
			tool: model.Tool{
				Tool: mcp.Tool{
					Name:        "status",
					Description: "Show the working tree status",
					InputSchema: map[string]any{
						"type": "object",
						"properties": map[string]any{
							"short": map[string]any{"type": "boolean", "default": false},
						},
					},
				},
				Namespace: "git",
				Tags:      []string{"vcs", "version-control"},
			},
			backend: model.NewMCPBackend("git-mcp"),
			doc: &tooldoc.DocEntry{
				Summary: "Displays paths with differences between index and HEAD",
				Notes:   "Use -s for short format output",
			},
		},
		{
			tool: model.Tool{
				Tool: mcp.Tool{
					Name:        "commit",
					Description: "Record changes to the repository",
					InputSchema: map[string]any{
						"type": "object",
						"properties": map[string]any{
							"message": map[string]any{"type": "string", "description": "Commit message"},
							"all":     map[string]any{"type": "boolean", "default": false},
						},
						"required": []string{"message"},
					},
				},
				Namespace: "git",
				Tags:      []string{"vcs", "version-control"},
			},
			backend: model.NewMCPBackend("git-mcp"),
			doc: &tooldoc.DocEntry{
				Summary: "Create a new commit with staged changes",
				Notes:   "Always write meaningful commit messages. Use -a to auto-stage modified files.",
				Examples: []tooldoc.ToolExample{
					{Title: "Simple commit", Args: map[string]any{"message": "Fix bug"}},
					{Title: "Commit all", Args: map[string]any{"message": "Update docs", "all": true}},
				},
			},
		},
		{
			tool: model.Tool{
				Tool: mcp.Tool{
					Name:        "ps",
					Description: "List running containers",
					InputSchema: map[string]any{
						"type": "object",
						"properties": map[string]any{
							"all": map[string]any{"type": "boolean", "default": false},
						},
					},
				},
				Namespace: "docker",
				Tags:      []string{"containers", "devops"},
			},
			backend: model.NewMCPBackend("docker-mcp"),
			doc: &tooldoc.DocEntry{
				Summary: "Show running Docker containers",
				Notes:   "Use -a to show all containers including stopped ones",
			},
		},
		{
			tool: model.Tool{
				Tool: mcp.Tool{
					Name:        "run",
					Description: "Run a command in a new container",
					InputSchema: map[string]any{
						"type": "object",
						"properties": map[string]any{
							"image":   map[string]any{"type": "string"},
							"command": map[string]any{"type": "string"},
							"detach":  map[string]any{"type": "boolean", "default": false},
						},
						"required": []string{"image"},
					},
				},
				Namespace: "docker",
				Tags:      []string{"containers", "devops"},
			},
			backend: model.NewMCPBackend("docker-mcp"),
			doc: &tooldoc.DocEntry{
				Summary: "Create and start a new container from an image",
				Notes:   "Use -d for detached mode. The container will be removed after exit unless --rm is used.",
				Examples: []tooldoc.ToolExample{
					{Title: "Run nginx", Args: map[string]any{"image": "nginx:latest", "detach": true}},
					{Title: "Run command", Args: map[string]any{"image": "alpine", "command": "echo hello"}},
				},
			},
		},
		{
			tool: model.Tool{
				Tool: mcp.Tool{
					Name:        "get",
					Description: "Display Kubernetes resources",
					InputSchema: map[string]any{
						"type": "object",
						"properties": map[string]any{
							"resource":  map[string]any{"type": "string"},
							"name":      map[string]any{"type": "string"},
							"namespace": map[string]any{"type": "string", "default": "default"},
						},
						"required": []string{"resource"},
					},
				},
				Namespace: "kubectl",
				Tags:      []string{"kubernetes", "k8s", "devops"},
			},
			backend: model.NewMCPBackend("k8s-mcp"),
			doc: &tooldoc.DocEntry{
				Summary: "Fetch and display Kubernetes resources",
				Notes:   "Common resources: pods, services, deployments, configmaps",
				Examples: []tooldoc.ToolExample{
					{Title: "List pods", Args: map[string]any{"resource": "pods"}},
					{Title: "Get service", Args: map[string]any{"resource": "service", "name": "my-svc"}},
				},
			},
		},
	}

	for _, t := range tools {
		if err := disc.RegisterTool(t.tool, t.backend, t.doc); err != nil {
			log.Printf("Failed to register %s: %v", t.tool.Name, err)
		}
	}
}

// simpleEmbedder creates embeddings based on word presence.
// In production, use a real embedding model.
type simpleEmbedder struct{}

func (e *simpleEmbedder) Embed(_ context.Context, text string) ([]float32, error) {
	keywords := []string{
		"git", "commit", "status", "diff", "changes", "working", "tree", "show", "record",
		"docker", "container", "run", "build", "image", "list", "ps",
		"kubernetes", "k8s", "kubectl", "cluster", "apply", "pods", "resources", "get",
		"devops", "management", "display", "version", "control",
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
