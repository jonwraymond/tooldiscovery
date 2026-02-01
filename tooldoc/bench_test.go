package tooldoc

import (
	"fmt"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/jonwraymond/tooldiscovery/index"
	"github.com/jonwraymond/toolfoundation/model"
)

func makeBenchTool(i int) model.Tool {
	return model.Tool{
		Tool: mcp.Tool{
			Name:        fmt.Sprintf("tool_%d", i),
			Description: fmt.Sprintf("Description for tool %d with various keywords like git docker kubernetes", i),
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"input":    map[string]any{"type": "string", "description": "Input parameter"},
					"optional": map[string]any{"type": "integer", "default": 10},
				},
				"required": []string{"input"},
			},
		},
		Namespace: fmt.Sprintf("ns_%d", i%10),
		Tags:      []string{"benchmark", "test"},
	}
}

func makeBenchDocEntry(i int) DocEntry {
	return DocEntry{
		Summary: fmt.Sprintf("Summary for tool %d with usage information", i),
		Notes:   fmt.Sprintf("Detailed notes for tool %d. This includes usage guidance, constraints, and best practices.", i),
		Examples: []ToolExample{
			{
				Title:       "Basic usage",
				Description: "Shows basic usage of the tool",
				Args:        map[string]any{"input": "test"},
			},
			{
				Title:       "Advanced usage",
				Description: "Shows advanced usage with optional parameters",
				Args:        map[string]any{"input": "test", "optional": 20},
				ResultHint:  `{"result": "success"}`,
			},
		},
		ExternalRefs: []string{"https://example.com/docs"},
	}
}

const benchDocCount = 1000

func setupStoreWithDocs() *InMemoryStore {
	idx := index.NewInMemoryIndex()
	backend := model.NewMCPBackend("bench-server")

	for i := range benchDocCount {
		tool := makeBenchTool(i)
		_ = idx.RegisterTool(tool, backend)
	}

	store := NewInMemoryStore(StoreOptions{Index: idx})

	for i := range benchDocCount {
		toolID := fmt.Sprintf("ns_%d:tool_%d", i%10, i)
		entry := makeBenchDocEntry(i)
		_ = store.RegisterDoc(toolID, entry)
	}

	return store
}

func BenchmarkStore_RegisterDoc(b *testing.B) {
	idx := index.NewInMemoryIndex()
	backend := model.NewMCPBackend("bench-server")
	tool := makeBenchTool(0)
	_ = idx.RegisterTool(tool, backend)

	entry := makeBenchDocEntry(0)

	b.ResetTimer()
	for b.Loop() {
		store := NewInMemoryStore(StoreOptions{Index: idx})
		_ = store.RegisterDoc("ns_0:tool_0", entry)
	}
}

func BenchmarkStore_RegisterDoc_Sequential(b *testing.B) {
	idx := index.NewInMemoryIndex()
	backend := model.NewMCPBackend("bench-server")

	for i := range b.N {
		tool := makeBenchTool(i)
		_ = idx.RegisterTool(tool, backend)
	}

	store := NewInMemoryStore(StoreOptions{Index: idx})

	b.ResetTimer()
	for i := range b.N {
		toolID := fmt.Sprintf("ns_%d:tool_%d", i%10, i)
		entry := makeBenchDocEntry(i)
		_ = store.RegisterDoc(toolID, entry)
	}
}

func BenchmarkStore_DescribeTool_Summary(b *testing.B) {
	store := setupStoreWithDocs()

	b.ResetTimer()
	for b.Loop() {
		_, _ = store.DescribeTool("ns_5:tool_500", DetailSummary)
	}
}

func BenchmarkStore_DescribeTool_Schema(b *testing.B) {
	store := setupStoreWithDocs()

	b.ResetTimer()
	for b.Loop() {
		_, _ = store.DescribeTool("ns_5:tool_500", DetailSchema)
	}
}

func BenchmarkStore_DescribeTool_Full(b *testing.B) {
	store := setupStoreWithDocs()

	b.ResetTimer()
	for b.Loop() {
		_, _ = store.DescribeTool("ns_5:tool_500", DetailFull)
	}
}

func BenchmarkStore_DescribeTool_VaryingLevel(b *testing.B) {
	store := setupStoreWithDocs()
	levels := []DetailLevel{DetailSummary, DetailSchema, DetailFull}

	for _, level := range levels {
		b.Run(string(level), func(b *testing.B) {
			b.ResetTimer()
			for b.Loop() {
				_, _ = store.DescribeTool("ns_5:tool_500", level)
			}
		})
	}
}

func BenchmarkStore_ListExamples(b *testing.B) {
	store := setupStoreWithDocs()

	b.ResetTimer()
	for b.Loop() {
		_, _ = store.ListExamples("ns_5:tool_500", 10)
	}
}

func BenchmarkStore_ListExamples_VaryingLimit(b *testing.B) {
	store := setupStoreWithDocs()
	limits := []int{1, 5, 10, 50}

	for _, limit := range limits {
		b.Run(fmt.Sprintf("limit_%d", limit), func(b *testing.B) {
			b.ResetTimer()
			for b.Loop() {
				_, _ = store.ListExamples("ns_5:tool_500", limit)
			}
		})
	}
}

func BenchmarkDocEntry_ValidateAndTruncate(b *testing.B) {
	entry := DocEntry{
		Summary: "A moderately long summary that tests the validation and truncation logic",
		Notes:   "Detailed notes that may or may not need truncation depending on their length",
		Examples: []ToolExample{
			{
				Title:       "Example",
				Description: "Description of the example",
				Args:        map[string]any{"key": "value", "nested": map[string]any{"a": 1}},
			},
		},
	}

	b.ResetTimer()
	for b.Loop() {
		_ = entry.ValidateAndTruncate()
	}
}

func BenchmarkValidateArgs(b *testing.B) {
	args := map[string]any{
		"string": "value",
		"number": 42,
		"nested": map[string]any{
			"level2": map[string]any{
				"level3": "deep",
			},
		},
		"array": []any{1, 2, 3},
	}

	b.ResetTimer()
	for b.Loop() {
		_, _ = ValidateArgs(args)
	}
}

func BenchmarkValidateArgs_DeepNesting(b *testing.B) {
	// Create args with maximum allowed nesting
	args := map[string]any{
		"l1": map[string]any{
			"l2": map[string]any{
				"l3": map[string]any{
					"l4": map[string]any{
						"l5": "value",
					},
				},
			},
		},
	}

	b.ResetTimer()
	for b.Loop() {
		_, _ = ValidateArgs(args)
	}
}

func BenchmarkConcurrent_DescribeTool(b *testing.B) {
	store := setupStoreWithDocs()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			toolID := fmt.Sprintf("ns_%d:tool_%d", i%10, i%1000)
			_, _ = store.DescribeTool(toolID, DetailFull)
			i++
		}
	})
}

func BenchmarkConcurrent_Mixed(b *testing.B) {
	store := setupStoreWithDocs()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			toolID := fmt.Sprintf("ns_%d:tool_%d", i%10, i%1000)
			switch i % 3 {
			case 0:
				_, _ = store.DescribeTool(toolID, DetailSummary)
			case 1:
				_, _ = store.DescribeTool(toolID, DetailFull)
			case 2:
				_, _ = store.ListExamples(toolID, 5)
			}
			i++
		}
	})
}
