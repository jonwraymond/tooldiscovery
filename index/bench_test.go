package index

import (
	"fmt"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"

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
					"input": map[string]any{"type": "string"},
				},
			},
		},
		Namespace: fmt.Sprintf("ns_%d", i%10),
		Tags:      []string{"benchmark", "test", fmt.Sprintf("tag_%d", i%5)},
	}
}

func makeBenchBackend(i int) model.ToolBackend {
	return model.ToolBackend{
		Kind: model.BackendKindMCP,
		MCP:  &model.MCPBackend{ServerName: fmt.Sprintf("server_%d", i%3)},
	}
}

func setupIndexWithTools(n int) *InMemoryIndex {
	idx := NewInMemoryIndex()
	for i := range n {
		tool := makeBenchTool(i)
		backend := makeBenchBackend(i)
		_ = idx.RegisterTool(tool, backend)
	}
	return idx
}

func BenchmarkIndex_RegisterTool(b *testing.B) {
	tool := makeBenchTool(0)
	backend := makeBenchBackend(0)

	for b.Loop() {
		idx := NewInMemoryIndex()
		_ = idx.RegisterTool(tool, backend)
	}
}

func BenchmarkIndex_RegisterTool_Sequential(b *testing.B) {
	idx := NewInMemoryIndex()
	backend := makeBenchBackend(0)

	b.ResetTimer()
	for i := range b.N {
		tool := makeBenchTool(i)
		_ = idx.RegisterTool(tool, backend)
	}
}

func BenchmarkIndex_Search(b *testing.B) {
	idx := setupIndexWithTools(1000)

	b.ResetTimer()
	for b.Loop() {
		_, _ = idx.Search("git", 10)
	}
}

func BenchmarkIndex_Search_VaryingSize(b *testing.B) {
	sizes := []int{100, 500, 1000, 5000}

	for _, size := range sizes {
		b.Run(fmt.Sprintf("tools_%d", size), func(b *testing.B) {
			idx := setupIndexWithTools(size)

			b.ResetTimer()
			for b.Loop() {
				_, _ = idx.Search("git", 10)
			}
		})
	}
}

func BenchmarkIndex_Search_VaryingLimit(b *testing.B) {
	idx := setupIndexWithTools(1000)
	limits := []int{5, 10, 50, 100}

	for _, limit := range limits {
		b.Run(fmt.Sprintf("limit_%d", limit), func(b *testing.B) {
			b.ResetTimer()
			for b.Loop() {
				_, _ = idx.Search("tool", limit)
			}
		})
	}
}

func BenchmarkIndex_SearchPage(b *testing.B) {
	idx := setupIndexWithTools(1000)

	b.ResetTimer()
	for b.Loop() {
		_, _, _ = idx.SearchPage("", 10, "")
	}
}

func BenchmarkIndex_GetTool(b *testing.B) {
	idx := setupIndexWithTools(1000)

	b.ResetTimer()
	for b.Loop() {
		_, _, _ = idx.GetTool("ns_5:tool_500")
	}
}

func BenchmarkIndex_GetTool_Miss(b *testing.B) {
	idx := setupIndexWithTools(1000)

	b.ResetTimer()
	for b.Loop() {
		_, _, _ = idx.GetTool("nonexistent:tool")
	}
}

func BenchmarkIndex_ListNamespaces(b *testing.B) {
	idx := setupIndexWithTools(1000)

	b.ResetTimer()
	for b.Loop() {
		_, _ = idx.ListNamespaces()
	}
}

func BenchmarkIndex_Concurrent_Search(b *testing.B) {
	idx := setupIndexWithTools(1000)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _ = idx.Search("git", 10)
		}
	})
}

func BenchmarkIndex_Concurrent_Mixed(b *testing.B) {
	idx := setupIndexWithTools(1000)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			switch i % 3 {
			case 0:
				_, _ = idx.Search("git", 10)
			case 1:
				_, _, _ = idx.GetTool("ns_5:tool_500")
			case 2:
				_, _ = idx.ListNamespaces()
			}
			i++
		}
	})
}

func BenchmarkIndex_OnChange_WithListener(b *testing.B) {
	idx := NewInMemoryIndex()
	eventCount := 0
	unsubscribe := idx.OnChange(func(_ ChangeEvent) {
		eventCount++
	})
	defer unsubscribe()

	backend := makeBenchBackend(0)

	b.ResetTimer()
	for i := range b.N {
		tool := makeBenchTool(i)
		_ = idx.RegisterTool(tool, backend)
	}
}

func BenchmarkIndex_OnChange_NoListener(b *testing.B) {
	idx := NewInMemoryIndex()
	backend := makeBenchBackend(0)

	b.ResetTimer()
	for i := range b.N {
		tool := makeBenchTool(i)
		_ = idx.RegisterTool(tool, backend)
	}
}
