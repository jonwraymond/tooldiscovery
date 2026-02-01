package tooldoc_test

import (
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/jonwraymond/tooldiscovery/index"
	"github.com/jonwraymond/tooldiscovery/tooldoc"
	"github.com/jonwraymond/toolfoundation/model"
)

func ExampleNewInMemoryStore() {
	// Create an index first
	idx := index.NewInMemoryIndex()

	tool := model.Tool{
		Tool: mcp.Tool{
			Name:        "create_issue",
			Description: "Create a new issue in a GitHub repository",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"title": map[string]any{"type": "string"},
					"body":  map[string]any{"type": "string"},
				},
				"required": []string{"title"},
			},
		},
		Namespace: "github",
	}
	_ = idx.RegisterTool(tool, model.NewMCPBackend("github-mcp"))

	// Create documentation store linked to the index
	store := tooldoc.NewInMemoryStore(tooldoc.StoreOptions{
		Index: idx,
	})

	// Register documentation for the tool
	_ = store.RegisterDoc("github:create_issue", tooldoc.DocEntry{
		Summary: "Creates GitHub issues with title and optional body",
		Notes:   "Requires GITHUB_TOKEN environment variable to be set.",
		Examples: []tooldoc.ToolExample{
			{
				Title:       "Create bug report",
				Description: "Create a simple bug report issue",
				Args:        map[string]any{"title": "Bug: Login fails", "body": "Steps to reproduce..."},
			},
		},
	})

	// Retrieve documentation at summary level
	doc, _ := store.DescribeTool("github:create_issue", tooldoc.DetailSummary)
	fmt.Println("Summary:", doc.Summary)
	// Output:
	// Summary: Creates GitHub issues with title and optional body
}

func ExampleInMemoryStore_DescribeTool() {
	idx := index.NewInMemoryIndex()

	tool := model.Tool{
		Tool: mcp.Tool{
			Name:        "search_code",
			Description: "Search for code across repositories",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"query":    map[string]any{"type": "string", "description": "Search query"},
					"language": map[string]any{"type": "string", "default": "any"},
				},
				"required": []string{"query"},
			},
		},
		Namespace: "github",
	}
	_ = idx.RegisterTool(tool, model.NewMCPBackend("github-mcp"))

	store := tooldoc.NewInMemoryStore(tooldoc.StoreOptions{Index: idx})
	_ = store.RegisterDoc("github:search_code", tooldoc.DocEntry{
		Summary: "Search code across GitHub repositories",
		Notes:   "Uses GitHub's code search API. Rate limited to 30 requests per minute.",
	})

	// Summary level - minimal info
	summary, _ := store.DescribeTool("github:search_code", tooldoc.DetailSummary)
	fmt.Println("Summary level:")
	fmt.Println("  Summary:", summary.Summary)
	fmt.Println("  Has Tool:", summary.Tool != nil)
	fmt.Println("  Has Notes:", summary.Notes != "")

	// Schema level - includes tool and schema info
	schema, _ := store.DescribeTool("github:search_code", tooldoc.DetailSchema)
	fmt.Println("Schema level:")
	fmt.Println("  Has Tool:", schema.Tool != nil)
	fmt.Println("  Has SchemaInfo:", schema.SchemaInfo != nil)

	// Full level - everything
	full, _ := store.DescribeTool("github:search_code", tooldoc.DetailFull)
	fmt.Println("Full level:")
	fmt.Println("  Has Notes:", full.Notes != "")
	// Output:
	// Summary level:
	//   Summary: Search code across GitHub repositories
	//   Has Tool: false
	//   Has Notes: false
	// Schema level:
	//   Has Tool: true
	//   Has SchemaInfo: true
	// Full level:
	//   Has Notes: true
}

func ExampleInMemoryStore_ListExamples() {
	idx := index.NewInMemoryIndex()

	tool := model.Tool{
		Tool: mcp.Tool{
			Name:        "run_query",
			Description: "Run a database query",
			InputSchema: map[string]any{"type": "object"},
		},
		Namespace: "db",
	}
	_ = idx.RegisterTool(tool, model.NewMCPBackend("db-server"))

	store := tooldoc.NewInMemoryStore(tooldoc.StoreOptions{Index: idx})

	// Register multiple examples
	examples := []tooldoc.ToolExample{
		{Title: "Select all", Description: "Get all records", Args: map[string]any{"sql": "SELECT * FROM users"}},
		{Title: "Filter by ID", Description: "Get specific record", Args: map[string]any{"sql": "SELECT * FROM users WHERE id = 1"}},
		{Title: "Count rows", Description: "Count total records", Args: map[string]any{"sql": "SELECT COUNT(*) FROM users"}},
	}
	_ = store.RegisterExamples("db:run_query", examples)

	// List examples with a limit
	listed, _ := store.ListExamples("db:run_query", 2)
	fmt.Println("Examples (limited to 2):")
	for _, ex := range listed {
		fmt.Printf("  - %s\n", ex.Title)
	}
	// Output:
	// Examples (limited to 2):
	//   - Select all
	//   - Filter by ID
}

func ExampleDocEntry_ValidateAndTruncate() {
	entry := tooldoc.DocEntry{
		Summary: "This is a very long summary that exceeds the maximum allowed length and will be truncated to fit within the 200 character limit that is enforced by the tooldoc package for summary fields to ensure consistent display",
		Notes:   "Some usage notes",
		Examples: []tooldoc.ToolExample{
			{
				Title:       "Example",
				Description: "A description that is also quite long and may need truncation if it exceeds the limit",
				Args:        map[string]any{"key": "value"},
			},
		},
	}

	validated := entry.ValidateAndTruncate()
	fmt.Printf("Summary length: %d (max 200)\n", len(validated.Summary))
	fmt.Println("Summary truncated:", len(validated.Summary) <= 200)
	// Output:
	// Summary length: 200 (max 200)
	// Summary truncated: true
}

func ExampleToolExample() {
	example := tooldoc.ToolExample{
		ID:          "ex-001",
		Title:       "Create user",
		Description: "Creates a new user account with the specified details",
		Args: map[string]any{
			"username": "johndoe",
			"email":    "john@example.com",
			"role":     "user",
		},
		ResultHint: `{"id": "user_123", "created": true}`,
	}

	fmt.Println("Title:", example.Title)
	fmt.Println("Has Args:", len(example.Args) > 0)
	fmt.Println("Has ResultHint:", example.ResultHint != "")
	// Output:
	// Title: Create user
	// Has Args: true
	// Has ResultHint: true
}

func ExampleValidateArgs() {
	// Valid args
	validArgs := map[string]any{
		"name": "test",
		"config": map[string]any{
			"enabled": true,
		},
	}
	stats, ok := tooldoc.ValidateArgs(validArgs)
	fmt.Printf("Valid: %v (depth=%d, keys=%d)\n", ok, stats.Depth, stats.Keys)

	// Args that are too deeply nested
	deepArgs := map[string]any{
		"l1": map[string]any{
			"l2": map[string]any{
				"l3": map[string]any{
					"l4": map[string]any{
						"l5": map[string]any{
							"l6": "too deep",
						},
					},
				},
			},
		},
	}
	stats, ok = tooldoc.ValidateArgs(deepArgs)
	fmt.Printf("Too deep: %v (depth=%d)\n", !ok, stats.Depth)
	// Output:
	// Valid: true (depth=2, keys=3)
	// Too deep: true (depth=6)
}
