package index_test

import (
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/jonwraymond/tooldiscovery/index"
	"github.com/jonwraymond/toolfoundation/model"
)

func ExampleNewInMemoryIndex() {
	idx := index.NewInMemoryIndex()

	tool := model.Tool{
		Tool: mcp.Tool{
			Name:        "search",
			Description: "Search for files in the filesystem",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"query": map[string]any{"type": "string"},
				},
			},
		},
		Namespace: "files",
		Tags:      []string{"search", "filesystem"},
	}
	backend := model.NewMCPBackend("files-server")

	_ = idx.RegisterTool(tool, backend)

	results, _ := idx.Search("search", 10)
	fmt.Println("Found:", len(results))
	// Output:
	// Found: 1
}

func ExampleInMemoryIndex_Search() {
	idx := index.NewInMemoryIndex()

	// Register multiple tools
	tools := []model.Tool{
		{
			Tool: mcp.Tool{
				Name:        "git_status",
				Description: "Show the working tree status",
				InputSchema: map[string]any{"type": "object"},
			},
			Namespace: "git",
			Tags:      []string{"vcs"},
		},
		{
			Tool: mcp.Tool{
				Name:        "git_commit",
				Description: "Record changes to the repository",
				InputSchema: map[string]any{"type": "object"},
			},
			Namespace: "git",
			Tags:      []string{"vcs"},
		},
		{
			Tool: mcp.Tool{
				Name:        "docker_ps",
				Description: "List running containers",
				InputSchema: map[string]any{"type": "object"},
			},
			Namespace: "docker",
			Tags:      []string{"containers"},
		},
	}

	backend := model.NewMCPBackend("dev-tools")
	for _, tool := range tools {
		_ = idx.RegisterTool(tool, backend)
	}

	// Search for git-related tools
	results, _ := idx.Search("git", 10)
	fmt.Println("Git tools found:", len(results))

	// Search for containers
	results, _ = idx.Search("containers", 10)
	fmt.Println("Container tools found:", len(results))
	// Output:
	// Git tools found: 2
	// Container tools found: 1
}

func ExampleInMemoryIndex_SearchPage() {
	idx := index.NewInMemoryIndex()

	// Register several tools
	for i := 0; i < 5; i++ {
		tool := model.Tool{
			Tool: mcp.Tool{
				Name:        fmt.Sprintf("tool_%d", i),
				Description: "A sample tool",
				InputSchema: map[string]any{"type": "object"},
			},
		}
		_ = idx.RegisterTool(tool, model.NewMCPBackend("server"))
	}

	// Paginate through results
	var cursor string
	page := 1
	for {
		results, nextCursor, _ := idx.SearchPage("", 2, cursor)
		fmt.Printf("Page %d: %d results\n", page, len(results))

		if nextCursor == "" {
			break
		}
		cursor = nextCursor
		page++
	}
	// Output:
	// Page 1: 2 results
	// Page 2: 2 results
	// Page 3: 1 results
}

func ExampleInMemoryIndex_GetTool() {
	idx := index.NewInMemoryIndex()

	tool := model.Tool{
		Tool: mcp.Tool{
			Name:        "read_file",
			Description: "Read contents of a file",
			InputSchema: map[string]any{"type": "object"},
		},
		Namespace: "files",
	}
	backend := model.NewMCPBackend("files-server")
	_ = idx.RegisterTool(tool, backend)

	// Retrieve the tool by its canonical ID
	retrieved, retrievedBackend, err := idx.GetTool("files:read_file")
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	fmt.Println("Name:", retrieved.Name)
	fmt.Println("Backend:", retrievedBackend.MCP.ServerName)
	// Output:
	// Name: read_file
	// Backend: files-server
}

func ExampleInMemoryIndex_ListNamespaces() {
	idx := index.NewInMemoryIndex()

	// Register tools in different namespaces
	namespaces := []string{"git", "docker", "kubectl"}
	for _, ns := range namespaces {
		tool := model.Tool{
			Tool: mcp.Tool{
				Name:        "tool",
				Description: "A tool",
				InputSchema: map[string]any{"type": "object"},
			},
			Namespace: ns,
		}
		_ = idx.RegisterTool(tool, model.NewMCPBackend("server"))
	}

	// List all namespaces (alphabetically sorted)
	nsList, _ := idx.ListNamespaces()
	for _, ns := range nsList {
		fmt.Println(ns)
	}
	// Output:
	// docker
	// git
	// kubectl
}

func ExampleInMemoryIndex_OnChange() {
	idx := index.NewInMemoryIndex()

	// Subscribe to changes
	unsubscribe := idx.OnChange(func(event index.ChangeEvent) {
		fmt.Printf("Event: %s for %s\n", event.Type, event.ToolID)
	})
	defer unsubscribe()

	tool := model.Tool{
		Tool: mcp.Tool{
			Name:        "my_tool",
			Description: "A tool",
			InputSchema: map[string]any{"type": "object"},
		},
	}

	// This triggers a change event
	_ = idx.RegisterTool(tool, model.NewMCPBackend("server"))
	// Output:
	// Event: registered for my_tool
}

func ExampleInMemoryIndex_RegisterToolsFromMCP() {
	idx := index.NewInMemoryIndex()

	// Simulate receiving tools from an MCP server
	mcpTools := []model.Tool{
		{
			Tool: mcp.Tool{
				Name:        "list_files",
				Description: "List files in directory",
				InputSchema: map[string]any{"type": "object"},
			},
		},
		{
			Tool: mcp.Tool{
				Name:        "read_file",
				Description: "Read file contents",
				InputSchema: map[string]any{"type": "object"},
			},
		},
	}

	// Register all tools from the MCP server
	_ = idx.RegisterToolsFromMCP("filesystem-server", mcpTools)

	results, _ := idx.Search("file", 10)
	fmt.Println("File tools:", len(results))
	// Output:
	// File tools: 2
}
