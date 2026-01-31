// Package tooldoc provides progressive, rich documentation for tools defined
// in the model package and indexed by the index package. It delivers tiered detail (summary,
// schema, full) without pulling long content into context until explicitly
// requested. It is transport-agnostic and designed to back MCP metatools
// (describe_tool, list_tool_examples) aligned with MCP spec 2025-11-25.
//
// # Documentation Tiers
//
// Summary: Returns a short description (1-2 lines) derived from Tool.Description
// or a custom doc override. Does not include schemas or examples. Works with
// docs-only registration (no tool in index required).
//
// Schema: Includes the full model.Tool (InputSchema/OutputSchema/Annotations).
// Adds derived schema info (required fields, defaults, allowed types) when
// available (best-effort). Requires tool to be resolved via index or
// StoreOptions.ToolResolver.
//
// Full: Includes everything in Schema plus human-authored Notes (constraints,
// pagination/auth hints, error semantics), optional small set of examples (1-3),
// and external references (URLs or resource IDs). Requires tool via index
// or StoreOptions.ToolResolver.
//
// # Error Handling
//
// The package defines four error values:
//   - ErrNotFound: Tool ID not found in index or docs
//   - ErrNoTool: Schema/full requested but tool not in index (docs may exist)
//   - ErrInvalidDetail: Invalid DetailLevel value
//   - ErrArgsTooLarge: Example Args exceeds depth (MaxArgsDepth) or size (MaxArgsKeys) caps
//
// Use errors.Is() to check error types.
//
// # Args Caps
//
// Example Args are validated at registration to prevent context pollution
// when examples are included in LLM context:
//   - MaxArgsDepth (5): Maximum nesting depth for maps/slices
//   - MaxArgsKeys (50): Maximum total size (map keys + slice items) across all levels
//
// RegisterDoc and RegisterExamples return ErrArgsTooLarge if any example
// violates these caps.
//
// # Usage
//
// Create an InMemoryStore with a index.Index reference:
//
//	idx := index.NewInMemoryIndex()
//	// ... register tools with idx ...
//
//	store := tooldoc.NewInMemoryStore(tooldoc.StoreOptions{
//		Index:       idx,
//		MaxExamples: 3, // Optional cap on examples returned
//	})
//
// Or inject tools directly without a index:
//
//	store := tooldoc.NewInMemoryStore(tooldoc.StoreOptions{
//		ToolResolver: func(id string) (*model.Tool, error) {
//			if id != "ns:tool" {
//				return nil, fmt.Errorf("not found: %s", id)
//			}
//			t := &model.Tool{}
//			t.Name = "tool"
//			t.Namespace = "ns"
//			t.Description = "Injected tool"
//			t.InputSchema = map[string]any{"type": "object"}
//			return t, nil
//		},
//	})
//
//	// Register documentation (returns error if Args exceeds caps)
//	err := store.RegisterDoc("my-tool", tooldoc.DocEntry{
//		Summary: "Creates a new widget",
//		Notes:   "Requires authentication. Supports pagination via cursor.",
//	})
//	if err != nil {
//		// Handle registration error (e.g., Args too large)
//	}
//
//	// Retrieve documentation at different levels
//	doc, err := store.DescribeTool("my-tool", tooldoc.DetailSummary)
//	doc, err := store.DescribeTool("my-tool", tooldoc.DetailSchema)
//	doc, err := store.DescribeTool("my-tool", tooldoc.DetailFull)
//
//	// Get examples (effective limit is min(max, MaxExamples))
//	examples, err := store.ListExamples("my-tool", 3)
//
// # Thread Safety
//
// InMemoryStore is safe for concurrent use. All reads and writes are
// properly synchronized. Example Args are deep-copied to prevent races.
package tooldoc
