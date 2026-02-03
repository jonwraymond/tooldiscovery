// Package index provides a global registry and search layer for tools.
//
// This package implements tool registration, storage, retrieval, and search
// capabilities. It supports multiple index backends and pluggable search strategies.
//
// # Index Types
//
// The package provides a built-in index implementation:
//
//   - InMemoryIndex: Fast, in-memory storage with optional custom searcher
//
// # Usage
//
// Create and populate an index:
//
//	idx := index.NewInMemoryIndex()
//
//	tool := model.Tool{
//	    Tool: mcp.Tool{
//	        Name:        "calculator",
//	        Description: "Performs arithmetic operations",
//	    },
//	    Namespace: "math",
//	    Tags:      []string{"arithmetic", "math"},
//	}
//	backend := model.ToolBackend{
//	    Kind: model.BackendKindMCP,
//	    MCP:  &model.MCPBackend{ServerName: "math-server"},
//	}
//	err := idx.RegisterTool(tool, backend)
//
// Search for tools:
//
//	results, err := idx.Search("arithmetic", 10)
//
// # Pluggable Search
//
// The index accepts a custom Searcher for advanced search capabilities:
//
//	type MySearcher struct{}
//	func (s *MySearcher) Search(query string, limit int, docs []index.SearchDoc) ([]index.Summary, error) {
//	    // Custom search implementation
//	}
//
//	idx := index.NewInMemoryIndex(index.WithSearcher(&MySearcher{}))
//
// # Progressive Disclosure
//
// Tools support progressive disclosure through Summary objects that contain
// only essential information for display and discovery:
//
//   - ID: Canonical tool identifier (namespace:name)
//   - Name: Tool name
//   - Namespace: Optional namespace for grouping
//   - ShortDescription: Truncated description (max 120 chars)
//   - Summary: Short summary (mirrors ShortDescription)
//   - Category: Optional category label
//   - InputModes: Supported input media types
//   - OutputModes: Supported output media types
//   - SecuritySummary: Short auth summary
//   - Tags: Associated tags for filtering
//
// # Pagination
//
// Search and list operations support cursor-based pagination:
//
//	results, nextCursor, err := idx.SearchPage("query", 10, "")
//	if nextCursor != "" {
//	    moreResults, nextCursor, err = idx.SearchPage("query", 10, nextCursor)
//	}
//
// # Change Notifications
//
// The index supports change notifications for reactive updates:
//
//	unsub := idx.OnChange(func(event index.ChangeEvent) {
//	    // Handle tool added/removed/updated
//	})
//	defer unsub()
//
// # Migration Note
//
// This package was migrated from github.com/jonwraymond/toolindex as part of
// the ApertureStack consolidation (PRD-130).
package index
