package discovery_test

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/jonwraymond/tooldiscovery/discovery"
	"github.com/jonwraymond/tooldiscovery/tooldoc"
	"github.com/jonwraymond/toolfoundation/model"
)

func ExampleNew() {
	// Create a Discovery instance with default options (BM25 search)
	disc, err := discovery.New(discovery.Options{})
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	// Register a tool
	tool := model.Tool{
		Tool: mcp.Tool{
			Name:        "create_issue",
			Description: "Create a new GitHub issue",
			InputSchema: map[string]any{"type": "object"},
		},
		Namespace: "github",
	}
	_ = disc.RegisterTool(tool, model.NewMCPBackend("github-server"), nil)

	fmt.Println("Discovery initialized")
	// Output:
	// Discovery initialized
}

func ExampleDiscovery_RegisterTool() {
	disc, _ := discovery.New(discovery.Options{})

	tool := model.Tool{
		Tool: mcp.Tool{
			Name:        "search_code",
			Description: "Search for code across repositories",
			InputSchema: map[string]any{"type": "object"},
		},
		Namespace: "github",
		Tags:      []string{"search", "code"},
	}
	backend := model.NewMCPBackend("github-mcp")

	// Register with documentation
	doc := &tooldoc.DocEntry{
		Summary: "Search code across GitHub repositories",
		Notes:   "Requires GITHUB_TOKEN. Rate limited to 30 req/min.",
		Examples: []tooldoc.ToolExample{
			{Title: "Search for function", Args: map[string]any{"query": "func main"}},
		},
	}

	err := disc.RegisterTool(tool, backend, doc)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	fmt.Println("Tool registered with documentation")
	// Output:
	// Tool registered with documentation
}

func ExampleDiscovery_Search() {
	disc, _ := discovery.New(discovery.Options{})

	// Register some tools
	tools := []model.Tool{
		{
			Tool:      mcp.Tool{Name: "git_status", Description: "Show working tree status", InputSchema: map[string]any{"type": "object"}},
			Namespace: "git",
			Tags:      []string{"vcs"},
		},
		{
			Tool:      mcp.Tool{Name: "git_commit", Description: "Record changes to repository", InputSchema: map[string]any{"type": "object"}},
			Namespace: "git",
			Tags:      []string{"vcs"},
		},
		{
			Tool:      mcp.Tool{Name: "docker_ps", Description: "List containers", InputSchema: map[string]any{"type": "object"}},
			Namespace: "docker",
			Tags:      []string{"containers"},
		},
	}
	for _, t := range tools {
		_ = disc.RegisterTool(t, model.NewMCPBackend("server"), nil)
	}

	// Search for git tools
	ctx := context.Background()
	results, _ := disc.Search(ctx, "git", 10)

	fmt.Println("Found:", len(results), "git-related tools")
	for _, r := range results {
		fmt.Printf("  - %s (%s)\n", r.Summary.Name, r.Summary.Namespace)
	}
	// Output:
	// Found: 2 git-related tools
	//   - git_commit (git)
	//   - git_status (git)
}

func ExampleDiscovery_DescribeTool() {
	disc, _ := discovery.New(discovery.Options{})

	tool := model.Tool{
		Tool: mcp.Tool{
			Name:        "run_query",
			Description: "Execute a database query",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"sql": map[string]any{"type": "string"},
				},
				"required": []string{"sql"},
			},
		},
		Namespace: "db",
	}
	_ = disc.RegisterTool(tool, model.NewMCPBackend("db-server"), &tooldoc.DocEntry{
		Summary: "Run SQL queries against the database",
		Notes:   "Use parameterized queries to prevent SQL injection.",
	})

	// Get summary (minimal)
	summary, _ := disc.DescribeTool("db:run_query", tooldoc.DetailSummary)
	fmt.Println("Summary:", summary.Summary)

	// Get full documentation
	full, _ := disc.DescribeTool("db:run_query", tooldoc.DetailFull)
	fmt.Println("Notes:", full.Notes)
	// Output:
	// Summary: Run SQL queries against the database
	// Notes: Use parameterized queries to prevent SQL injection.
}

func ExampleResults_FilterByNamespace() {
	disc, _ := discovery.New(discovery.Options{})

	tools := []model.Tool{
		{Tool: mcp.Tool{Name: "status", Description: "Git status", InputSchema: map[string]any{"type": "object"}}, Namespace: "git"},
		{Tool: mcp.Tool{Name: "commit", Description: "Git commit", InputSchema: map[string]any{"type": "object"}}, Namespace: "git"},
		{Tool: mcp.Tool{Name: "ps", Description: "Docker ps", InputSchema: map[string]any{"type": "object"}}, Namespace: "docker"},
	}
	for _, t := range tools {
		_ = disc.RegisterTool(t, model.NewMCPBackend("server"), nil)
	}

	ctx := context.Background()
	results, _ := disc.Search(ctx, "", 10) // Get all

	// Filter to just git tools
	gitTools := results.FilterByNamespace("git")
	fmt.Println("Git tools:", len(gitTools))
	// Output:
	// Git tools: 2
}

func ExampleResults_IDs() {
	disc, _ := discovery.New(discovery.Options{})

	tools := []model.Tool{
		{Tool: mcp.Tool{Name: "create_issue", Description: "Create issue", InputSchema: map[string]any{"type": "object"}}, Namespace: "github"},
		{Tool: mcp.Tool{Name: "list_repos", Description: "List repos", InputSchema: map[string]any{"type": "object"}}, Namespace: "github"},
	}
	for _, t := range tools {
		_ = disc.RegisterTool(t, model.NewMCPBackend("server"), nil)
	}

	ctx := context.Background()
	results, _ := disc.Search(ctx, "github", 10)

	ids := results.IDs()
	fmt.Println("Found tool IDs:", len(ids))
	// Output:
	// Found tool IDs: 2
}
