// Package discovery provides a unified facade for tool discovery operations.
//
// It combines the functionality of the index, search, semantic, and tooldoc
// packages into a single, easy-to-use API. This package is the recommended
// entry point for most tool discovery use cases.
//
// # Basic Usage
//
// Create a Discovery instance with default options:
//
//	disc, err := discovery.New(discovery.Options{})
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// Register a tool with documentation
//	err = disc.RegisterTool(tool, backend, &tooldoc.DocEntry{
//	    Summary:  "Creates GitHub issues",
//	    Notes:    "Requires GITHUB_TOKEN",
//	    Examples: []tooldoc.ToolExample{{Title: "Create bug", Args: map[string]any{"title": "Bug"}}},
//	})
//
//	// Search for tools
//	results, err := disc.Search(ctx, "create issue", 10)
//
//	// Get progressive documentation
//	doc, err := disc.DescribeTool("github:create-issue", tooldoc.DetailFull)
//
// # Hybrid Search
//
// Enable hybrid search by providing an embedder:
//
//	disc, err := discovery.New(discovery.Options{
//	    Embedder:    myEmbedder,  // implements semantic.Embedder
//	    HybridAlpha: 0.7,         // 70% BM25, 30% semantic
//	})
//
// # Components
//
// The Discovery facade integrates:
//   - index.Index: Tool registration and lookup
//   - index.Searcher: BM25-based text search
//   - semantic.Strategy: Embedding-based semantic search (optional)
//   - tooldoc.Store: Progressive documentation
//
// # Thread Safety
//
// All Discovery methods are safe for concurrent use.
package discovery
