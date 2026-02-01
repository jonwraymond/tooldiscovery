// Package main demonstrates basic tool discovery with BM25 search.
//
// This example shows how to:
//   - Create an index with a BM25 searcher
//   - Register tools from an MCP server
//   - Perform text-based search
//   - Use progressive disclosure for documentation
//
// Run with: go run ./examples/basic
package main

import (
	"fmt"
	"log"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/jonwraymond/tooldiscovery/index"
	"github.com/jonwraymond/tooldiscovery/search"
	"github.com/jonwraymond/tooldiscovery/tooldoc"
	"github.com/jonwraymond/toolfoundation/model"
)

func main() {
	// 1. Create a BM25 searcher with custom configuration
	searcher := search.NewBM25Searcher(search.BM25Config{
		NameBoost:      3, // Name matches are 3x more important
		NamespaceBoost: 2, // Namespace matches are 2x
		TagsBoost:      2, // Tag matches are 2x
	})
	defer func() {
		_ = searcher.Close()
	}()

	// 2. Create an index with the BM25 searcher
	idx := index.NewInMemoryIndex(index.IndexOptions{
		Searcher: searcher,
	})

	// 3. Register tools (simulating MCP server response)
	tools := createSampleTools()
	backend := model.NewMCPBackend("dev-tools-mcp")

	for _, tool := range tools {
		if err := idx.RegisterTool(tool, backend); err != nil {
			log.Fatalf("Failed to register %s: %v", tool.Name, err)
		}
	}
	fmt.Printf("Registered %d tools\n\n", len(tools))

	// 4. Search for git-related tools
	fmt.Println("=== Search: 'git' ===")
	results, err := idx.Search("git", 5)
	if err != nil {
		log.Fatalf("Search failed: %v", err)
	}
	for _, r := range results {
		fmt.Printf("  - %s: %s\n", r.ID, r.ShortDescription)
	}

	// 5. Search for container tools
	fmt.Println("\n=== Search: 'containers' ===")
	results, err = idx.Search("containers", 5)
	if err != nil {
		log.Fatalf("Search failed: %v", err)
	}
	for _, r := range results {
		fmt.Printf("  - %s: %s\n", r.ID, r.ShortDescription)
	}

	// 6. List all namespaces
	fmt.Println("\n=== Namespaces ===")
	namespaces, _ := idx.ListNamespaces()
	for _, ns := range namespaces {
		fmt.Printf("  - %s\n", ns)
	}

	// 7. Use progressive disclosure for documentation
	fmt.Println("\n=== Progressive Disclosure ===")
	store := tooldoc.NewInMemoryStore(tooldoc.StoreOptions{Index: idx})

	// Register documentation for one tool
	_ = store.RegisterDoc("git:status", tooldoc.DocEntry{
		Summary: "Shows the state of your working directory",
		Notes:   "Use -s for short format, -b to show branch info",
		Examples: []tooldoc.ToolExample{
			{Title: "Basic status", Args: map[string]any{}},
			{Title: "Short format", Args: map[string]any{"short": true}},
		},
	})

	// Summary level (cheap)
	summary, _ := store.DescribeTool("git:status", tooldoc.DetailSummary)
	fmt.Printf("  Summary: %s\n", summary.Summary)

	// Full level (complete)
	full, _ := store.DescribeTool("git:status", tooldoc.DetailFull)
	fmt.Printf("  Notes: %s\n", full.Notes)
	fmt.Printf("  Examples: %d\n", len(full.Examples))
}

func createSampleTools() []model.Tool {
	return []model.Tool{
		{
			Tool: mcp.Tool{
				Name:        "status",
				Description: "Show the working tree status",
				InputSchema: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"short": map[string]any{"type": "boolean", "description": "Use short format"},
					},
				},
			},
			Namespace: "git",
			Tags:      []string{"vcs", "version-control"},
		},
		{
			Tool: mcp.Tool{
				Name:        "commit",
				Description: "Record changes to the repository",
				InputSchema: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"message": map[string]any{"type": "string", "description": "Commit message"},
						"all":     map[string]any{"type": "boolean", "description": "Commit all changed files"},
					},
					"required": []string{"message"},
				},
			},
			Namespace: "git",
			Tags:      []string{"vcs", "version-control"},
		},
		{
			Tool: mcp.Tool{
				Name:        "ps",
				Description: "List running containers",
				InputSchema: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"all": map[string]any{"type": "boolean", "description": "Show all containers"},
					},
				},
			},
			Namespace: "docker",
			Tags:      []string{"containers", "devops"},
		},
		{
			Tool: mcp.Tool{
				Name:        "run",
				Description: "Run a command in a new container",
				InputSchema: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"image":   map[string]any{"type": "string", "description": "Image name"},
						"command": map[string]any{"type": "string", "description": "Command to run"},
					},
					"required": []string{"image"},
				},
			},
			Namespace: "docker",
			Tags:      []string{"containers", "devops"},
		},
		{
			Tool: mcp.Tool{
				Name:        "get",
				Description: "Display one or many resources",
				InputSchema: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"resource": map[string]any{"type": "string", "description": "Resource type"},
						"name":     map[string]any{"type": "string", "description": "Resource name"},
					},
					"required": []string{"resource"},
				},
			},
			Namespace: "kubectl",
			Tags:      []string{"kubernetes", "k8s", "devops"},
		},
	}
}
