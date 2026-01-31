package tooldoc

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/jonwraymond/tooldiscovery/index"
	"github.com/jonwraymond/toolfoundation/model"
)

// Helper to create a tool with InputSchema
func makeToolWithSchema(name, namespace, description string, schema map[string]any) model.Tool {
	t := model.Tool{
		Namespace: namespace,
	}
	t.Name = name
	t.Description = description
	t.InputSchema = schema
	return t
}

func mustRegisterDoc(t *testing.T, store *InMemoryStore, id string, entry DocEntry) {
	t.Helper()
	if err := store.RegisterDoc(id, entry); err != nil {
		t.Fatalf("RegisterDoc failed: %v", err)
	}
}

func mustRegisterExamples(t *testing.T, store *InMemoryStore, id string, examples []ToolExample) {
	t.Helper()
	if err := store.RegisterExamples(id, examples); err != nil {
		t.Fatalf("RegisterExamples failed: %v", err)
	}
}

func TestDetailLevelConstants(t *testing.T) {
	// Verify constant values match PRD
	if DetailSummary != "summary" {
		t.Errorf("DetailSummary = %q, want %q", DetailSummary, "summary")
	}
	if DetailSchema != "schema" {
		t.Errorf("DetailSchema = %q, want %q", DetailSchema, "schema")
	}
	if DetailFull != "full" {
		t.Errorf("DetailFull = %q, want %q", DetailFull, "full")
	}
}

func TestNewInMemoryStore(t *testing.T) {
	store := NewInMemoryStore(StoreOptions{})
	if store == nil {
		t.Fatal("NewInMemoryStore returned nil")
	}
	if store.docs == nil {
		t.Error("docs map not initialized")
	}
}

func TestRegisterDoc(t *testing.T) {
	store := NewInMemoryStore(StoreOptions{})

	entry := DocEntry{
		Summary:      "Test summary",
		Notes:        "Test notes",
		ExternalRefs: []string{"https://example.com"},
		Examples: []ToolExample{
			{Title: "Example 1", Description: "First example", Args: map[string]any{"key": "value"}},
		},
	}

	mustRegisterDoc(t, store, "test-tool", entry)

	// Verify registration
	store.mu.RLock()
	record := store.docs["test-tool"]
	store.mu.RUnlock()

	if record == nil {
		t.Fatal("doc record not created")
	}
	if record.summary != entry.Summary {
		t.Errorf("summary = %q, want %q", record.summary, entry.Summary)
	}
	if record.notes != entry.Notes {
		t.Errorf("notes = %q, want %q", record.notes, entry.Notes)
	}
	if len(record.examples) != 1 {
		t.Errorf("len(examples) = %d, want 1", len(record.examples))
	}
}

func TestRegisterDoc_Truncation(t *testing.T) {
	store := NewInMemoryStore(StoreOptions{})

	// Create strings longer than caps
	longSummary := strings.Repeat("a", MaxSummaryLen+100)
	longNotes := strings.Repeat("b", MaxNotesLen+100)
	longDesc := strings.Repeat("c", MaxDescriptionLen+100)
	longHint := strings.Repeat("d", MaxResultHintLen+100)

	entry := DocEntry{
		Summary: longSummary,
		Notes:   longNotes,
		Examples: []ToolExample{
			{Title: "Ex", Description: longDesc, ResultHint: longHint},
		},
	}

	mustRegisterDoc(t, store, "test-tool", entry)

	store.mu.RLock()
	record := store.docs["test-tool"]
	store.mu.RUnlock()

	if len(record.summary) != MaxSummaryLen {
		t.Errorf("summary len = %d, want %d", len(record.summary), MaxSummaryLen)
	}
	if len(record.notes) != MaxNotesLen {
		t.Errorf("notes len = %d, want %d", len(record.notes), MaxNotesLen)
	}
	if len(record.examples[0].Description) != MaxDescriptionLen {
		t.Errorf("description len = %d, want %d", len(record.examples[0].Description), MaxDescriptionLen)
	}
	if len(record.examples[0].ResultHint) != MaxResultHintLen {
		t.Errorf("resultHint len = %d, want %d", len(record.examples[0].ResultHint), MaxResultHintLen)
	}
}

func TestDescribeTool_Summary_WithDocOnly(t *testing.T) {
	store := NewInMemoryStore(StoreOptions{})
	mustRegisterDoc(t, store, "my-tool", DocEntry{Summary: "My tool does things"})

	doc, err := store.DescribeTool("my-tool", DetailSummary)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if doc.Summary != "My tool does things" {
		t.Errorf("Summary = %q, want %q", doc.Summary, "My tool does things")
	}
	if doc.Tool != nil {
		t.Error("Tool should be nil for summary level")
	}
	if doc.SchemaInfo != nil {
		t.Error("SchemaInfo should be nil for summary level")
	}
	if doc.Notes != "" {
		t.Error("Notes should be empty for summary level")
	}
	if len(doc.Examples) != 0 {
		t.Error("Examples should be empty for summary level")
	}
}

func TestDescribeTool_Summary_FromToolDescription(t *testing.T) {
	idx := index.NewInMemoryIndex()
	tool := makeToolWithSchema("my-tool", "test", "Tool description from metadata", map[string]any{
		"type": "object",
	})
	backend := model.ToolBackend{
		Kind:  model.BackendKindLocal,
		Local: &model.LocalBackend{Name: "handler"},
	}
	if err := idx.RegisterTool(tool, backend); err != nil {
		t.Fatalf("failed to register tool: %v", err)
	}

	store := NewInMemoryStore(StoreOptions{Index: idx})

	// No doc registered, should use tool description
	doc, err := store.DescribeTool("test:my-tool", DetailSummary)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if doc.Summary != "Tool description from metadata" {
		t.Errorf("Summary = %q, want %q", doc.Summary, "Tool description from metadata")
	}
}

func TestDescribeTool_ResolverProvidesTool(t *testing.T) {
	tool := makeToolWithSchema("resolve", "ns", "Resolved tool", map[string]any{
		"type": "object",
	})

	store := NewInMemoryStore(StoreOptions{
		ToolResolver: func(id string) (*model.Tool, error) {
			if id != "ns:resolve" {
				return nil, nil
			}
			t := tool
			return &t, nil
		},
	})

	// Summary should use resolver-provided description
	doc, err := store.DescribeTool("ns:resolve", DetailSummary)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if doc.Summary != "Resolved tool" {
		t.Errorf("Summary = %q, want %q", doc.Summary, "Resolved tool")
	}

	// Schema should succeed without index
	doc, err = store.DescribeTool("ns:resolve", DetailSchema)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if doc.Tool == nil || doc.Tool.Name != "resolve" {
		t.Fatalf("Tool = %+v, want resolve tool", doc.Tool)
	}
}

func TestDescribeTool_ResolverErrorPropagation(t *testing.T) {
	resolverErr := errors.New("resolver failure")

	store := NewInMemoryStore(StoreOptions{
		ToolResolver: func(_ string) (*model.Tool, error) {
			return nil, resolverErr
		},
	})

	// Register doc so we can test schema level
	if err := store.RegisterDoc("test:tool", DocEntry{Summary: "test"}); err != nil {
		t.Fatalf("failed to register: %v", err)
	}

	// Schema level should propagate resolver error
	_, err := store.DescribeTool("test:tool", DetailSchema)
	if err == nil {
		t.Error("expected resolver error to propagate")
	}
	if !errors.Is(err, resolverErr) {
		t.Errorf("error = %v, want resolver error", err)
	}

	// Full level should also propagate resolver error
	_, err = store.DescribeTool("test:tool", DetailFull)
	if err == nil {
		t.Error("expected resolver error to propagate")
	}
	if !errors.Is(err, resolverErr) {
		t.Errorf("error = %v, want resolver error", err)
	}
}

func TestDescribeTool_SummaryResolverErrorPropagation(t *testing.T) {
	resolverErr := errors.New("resolver failure")

	store := NewInMemoryStore(StoreOptions{
		ToolResolver: func(_ string) (*model.Tool, error) {
			return nil, resolverErr
		},
	})

	_, err := store.DescribeTool("missing:tool", DetailSummary)
	if err == nil {
		t.Fatal("expected resolver error to propagate at summary level")
	}
	if !errors.Is(err, resolverErr) {
		t.Errorf("error = %v, want resolver error", err)
	}
}

func TestDescribeTool_Schema(t *testing.T) {
	idx := index.NewInMemoryIndex()
	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"query": map[string]any{
				"type":    "string",
				"default": "test",
			},
			"limit": map[string]any{
				"type": "integer",
			},
		},
		"required": []any{"query"},
	}
	tool := makeToolWithSchema("search", "api", "Search for items", schema)
	backend := model.ToolBackend{
		Kind:  model.BackendKindLocal,
		Local: &model.LocalBackend{Name: "handler"},
	}
	if err := idx.RegisterTool(tool, backend); err != nil {
		t.Fatalf("failed to register tool: %v", err)
	}

	store := NewInMemoryStore(StoreOptions{Index: idx})
	mustRegisterDoc(t, store, "api:search", DocEntry{
		Summary: "Custom summary",
		Notes:   "These notes should not appear at schema level",
	})

	doc, err := store.DescribeTool("api:search", DetailSchema)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should have tool
	if doc.Tool == nil {
		t.Fatal("Tool should not be nil for schema level")
	}
	if doc.Tool.Name != "search" {
		t.Errorf("Tool.Name = %q, want %q", doc.Tool.Name, "search")
	}

	// Should use custom summary
	if doc.Summary != "Custom summary" {
		t.Errorf("Summary = %q, want %q", doc.Summary, "Custom summary")
	}

	// Should have schema info
	if doc.SchemaInfo == nil {
		t.Fatal("SchemaInfo should not be nil")
	}
	if len(doc.SchemaInfo.Required) != 1 || doc.SchemaInfo.Required[0] != "query" {
		t.Errorf("Required = %v, want [query]", doc.SchemaInfo.Required)
	}
	if doc.SchemaInfo.Types["query"][0] != "string" {
		t.Errorf("Types[query] = %v, want [string]", doc.SchemaInfo.Types["query"])
	}
	if doc.SchemaInfo.Defaults["query"] != "test" {
		t.Errorf("Defaults[query] = %v, want test", doc.SchemaInfo.Defaults["query"])
	}

	// Notes should be empty at schema level
	if doc.Notes != "" {
		t.Errorf("Notes = %q, want empty", doc.Notes)
	}
}

func TestDescribeTool_Full(t *testing.T) {
	idx := index.NewInMemoryIndex()
	tool := makeToolWithSchema("create", "tickets", "Create a ticket", map[string]any{
		"type": "object",
		"properties": map[string]any{
			"title": map[string]any{"type": "string"},
		},
		"required": []any{"title"},
	})
	backend := model.ToolBackend{
		Kind:  model.BackendKindLocal,
		Local: &model.LocalBackend{Name: "handler"},
	}
	if err := idx.RegisterTool(tool, backend); err != nil {
		t.Fatalf("failed to register tool: %v", err)
	}

	store := NewInMemoryStore(StoreOptions{Index: idx})
	mustRegisterDoc(t, store, "tickets:create", DocEntry{
		Summary:      "Create a new ticket",
		Notes:        "Requires authentication. Rate limited to 100/min.",
		ExternalRefs: []string{"https://docs.example.com/tickets"},
		Examples: []ToolExample{
			{
				Title:       "Basic ticket",
				Description: "Create a simple ticket",
				Args:        map[string]any{"title": "Bug report"},
				ResultHint:  "Returns ticket ID",
			},
		},
	})

	doc, err := store.DescribeTool("tickets:create", DetailFull)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should have everything
	if doc.Tool == nil {
		t.Error("Tool should not be nil")
	}
	if doc.Summary != "Create a new ticket" {
		t.Errorf("Summary = %q, want %q", doc.Summary, "Create a new ticket")
	}
	if doc.SchemaInfo == nil {
		t.Error("SchemaInfo should not be nil")
	}
	if doc.Notes != "Requires authentication. Rate limited to 100/min." {
		t.Errorf("Notes = %q, want auth/rate limit notes", doc.Notes)
	}
	if len(doc.Examples) != 1 {
		t.Errorf("len(Examples) = %d, want 1", len(doc.Examples))
	}
	if len(doc.ExternalRefs) != 1 {
		t.Errorf("len(ExternalRefs) = %d, want 1", len(doc.ExternalRefs))
	}
}

func TestDescribeTool_NotFound(t *testing.T) {
	store := NewInMemoryStore(StoreOptions{})

	_, err := store.DescribeTool("nonexistent", DetailSummary)
	if err == nil {
		t.Error("expected error for nonexistent tool")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error = %v, want 'not found'", err)
	}
}

func TestDescribeTool_InvalidLevel(t *testing.T) {
	store := NewInMemoryStore(StoreOptions{})
	mustRegisterDoc(t, store, "test", DocEntry{Summary: "test"})

	_, err := store.DescribeTool("test", "invalid")
	if err == nil {
		t.Error("expected error for invalid detail level")
	}
	if !strings.Contains(err.Error(), "invalid detail level") {
		t.Errorf("error = %v, want 'invalid detail level'", err)
	}
}

func TestListExamples(t *testing.T) {
	store := NewInMemoryStore(StoreOptions{})
	mustRegisterDoc(t, store, "my-tool", DocEntry{
		Summary: "test",
		Examples: []ToolExample{
			{Title: "Ex1", Description: "First"},
			{Title: "Ex2", Description: "Second"},
			{Title: "Ex3", Description: "Third"},
		},
	})

	// Get all
	examples, err := store.ListExamples("my-tool", 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(examples) != 3 {
		t.Errorf("len(examples) = %d, want 3", len(examples))
	}

	// Get limited
	examples, err = store.ListExamples("my-tool", 2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(examples) != 2 {
		t.Errorf("len(examples) = %d, want 2", len(examples))
	}
	if examples[0].Title != "Ex1" || examples[1].Title != "Ex2" {
		t.Errorf("examples = %v, want Ex1, Ex2", examples)
	}
}

func TestListExamples_NotFound(t *testing.T) {
	store := NewInMemoryStore(StoreOptions{})

	_, err := store.ListExamples("nonexistent", 10)
	if err == nil {
		t.Error("expected error for nonexistent tool")
	}
}

func TestListExamples_WithIndex(t *testing.T) {
	idx := index.NewInMemoryIndex()
	tool := makeToolWithSchema("my-tool", "ns", "desc", map[string]any{"type": "object"})
	backend := model.ToolBackend{
		Kind:  model.BackendKindLocal,
		Local: &model.LocalBackend{Name: "handler"},
	}
	if err := idx.RegisterTool(tool, backend); err != nil {
		t.Fatalf("failed to register: %v", err)
	}

	store := NewInMemoryStore(StoreOptions{Index: idx})
	mustRegisterExamples(t, store, "ns:my-tool", []ToolExample{
		{Title: "Example", Description: "Test"},
	})

	examples, err := store.ListExamples("ns:my-tool", 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(examples) != 1 {
		t.Errorf("len(examples) = %d, want 1", len(examples))
	}
}

func TestListExamples_WithResolver(t *testing.T) {
	tool := makeToolWithSchema("my-tool", "ns", "desc", map[string]any{"type": "object"})

	store := NewInMemoryStore(StoreOptions{
		ToolResolver: func(id string) (*model.Tool, error) {
			if id == "ns:my-tool" {
				t := tool
				return &t, nil
			}
			return nil, nil
		},
	})

	// Tool exists via resolver but has no docs - should return empty, not error
	examples, err := store.ListExamples("ns:my-tool", 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(examples) != 0 {
		t.Errorf("len(examples) = %d, want 0", len(examples))
	}

	// Register examples for resolver-provided tool
	if err := store.RegisterExamples("ns:my-tool", []ToolExample{
		{Title: "Example", Description: "Test"},
	}); err != nil {
		t.Fatalf("failed to register: %v", err)
	}

	examples, err = store.ListExamples("ns:my-tool", 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(examples) != 1 {
		t.Errorf("len(examples) = %d, want 1", len(examples))
	}

	// Tool not found by resolver - should return ErrNotFound
	_, err = store.ListExamples("unknown:tool", 10)
	if err == nil {
		t.Error("expected error for unknown tool")
	}
}

func TestListExamples_ResolverErrorPropagation(t *testing.T) {
	resolverErr := errors.New("resolver failure")

	store := NewInMemoryStore(StoreOptions{
		ToolResolver: func(_ string) (*model.Tool, error) {
			return nil, resolverErr
		},
	})

	_, err := store.ListExamples("any:tool", 10)
	if err == nil {
		t.Error("expected resolver error to propagate")
	}
	if !errors.Is(err, resolverErr) {
		t.Errorf("error = %v, want resolver error", err)
	}
}

func TestDeriveSchemaInfo(t *testing.T) {
	tests := []struct {
		name     string
		schema   any
		wantNil  bool
		wantReq  []string
		wantType map[string][]string
		wantDef  map[string]any
	}{
		{
			name:    "nil schema",
			schema:  nil,
			wantNil: true,
		},
		{
			name: "basic schema",
			schema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"name": map[string]any{"type": "string"},
					"age":  map[string]any{"type": "integer", "default": 0},
				},
				"required": []any{"name"},
			},
			wantReq:  []string{"name"},
			wantType: map[string][]string{"name": {"string"}, "age": {"integer"}},
			wantDef:  map[string]any{"age": float64(0)}, // Normalized to float64
		},
		{
			name: "array type",
			schema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"value": map[string]any{"type": []any{"string", "null"}},
				},
			},
			wantType: map[string][]string{"value": {"string", "null"}},
		},
		{
			name: "json.RawMessage schema",
			schema: json.RawMessage(`{
				"type": "object",
				"properties": {"x": {"type": "number"}},
				"required": ["x"]
			}`),
			wantReq:  []string{"x"},
			wantType: map[string][]string{"x": {"number"}},
		},
		{
			name: "typed []string required (Go literal)",
			schema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"foo": map[string]any{"type": "string"},
					"bar": map[string]any{"type": "integer"},
				},
				"required": []string{"foo", "bar"}, // []string instead of []any
			},
			wantReq:  []string{"foo", "bar"},
			wantType: map[string][]string{"foo": {"string"}, "bar": {"integer"}},
		},
		{
			name: "typed []string union type (Go literal)",
			schema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"nullable": map[string]any{"type": []string{"string", "null"}}, // []string
				},
			},
			wantType: map[string][]string{"nullable": {"string", "null"}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info := deriveSchemaInfo(tt.schema)

			if tt.wantNil {
				if info != nil {
					t.Errorf("expected nil, got %+v", info)
				}
				return
			}

			if info == nil {
				t.Fatal("expected non-nil SchemaInfo")
			}

			// Check required
			if len(tt.wantReq) > 0 {
				if len(info.Required) != len(tt.wantReq) {
					t.Errorf("Required = %v, want %v", info.Required, tt.wantReq)
				}
			}

			// Check types
			for k, v := range tt.wantType {
				if got := info.Types[k]; len(got) != len(v) {
					t.Errorf("Types[%s] = %v, want %v", k, got, v)
				}
			}

			// Check defaults
			for k, v := range tt.wantDef {
				if got := info.Defaults[k]; got != v {
					t.Errorf("Defaults[%s] = %v, want %v", k, got, v)
				}
			}
		})
	}
}

func TestToolDoc_MCPShapeMapping(t *testing.T) {
	// Test that ToolDoc fields map cleanly to MCP metatool outputs
	idx := index.NewInMemoryIndex()
	tool := makeToolWithSchema("search", "api", "Search API", map[string]any{
		"type": "object",
		"properties": map[string]any{
			"q": map[string]any{"type": "string"},
		},
	})
	backend := model.ToolBackend{
		Kind:  model.BackendKindLocal,
		Local: &model.LocalBackend{Name: "h"},
	}
	if err := idx.RegisterTool(tool, backend); err != nil {
		t.Fatalf("failed to register tool: %v", err)
	}

	store := NewInMemoryStore(StoreOptions{Index: idx})
	mustRegisterDoc(t, store, "api:search", DocEntry{
		Summary:      "Search for results",
		Notes:        "Pagination via cursor",
		ExternalRefs: []string{"https://api.example.com/docs/search"},
		Examples: []ToolExample{
			{
				Title:       "Simple search",
				Description: "Search for widgets",
				Args:        map[string]any{"q": "widgets"},
				ResultHint:  "Array of results",
			},
		},
	})

	doc, err := store.DescribeTool("api:search", DetailFull)
	if err != nil {
		t.Fatalf("DescribeTool failed: %v", err)
	}

	// Verify MCP describe_tool shape:
	// - Tool object (InputSchema, OutputSchema, Annotations)
	// - notes, examples, externalRefs
	if doc.Tool == nil {
		t.Error("MCP shape requires Tool")
	}
	if doc.Tool.Name != "search" {
		t.Error("Tool.Name missing")
	}
	if doc.Notes == "" {
		t.Error("notes field empty")
	}
	if len(doc.Examples) == 0 {
		t.Error("examples field empty")
	}
	if len(doc.ExternalRefs) == 0 {
		t.Error("externalRefs field empty")
	}

	// Verify MCP list_tool_examples shape:
	// - examples: [ { title, description, args, resultHint } ]
	examples, err := store.ListExamples("api:search", 10)
	if err != nil {
		t.Fatalf("ListExamples failed: %v", err)
	}
	if len(examples) == 0 {
		t.Fatal("no examples returned")
	}
	ex := examples[0]
	if ex.Title == "" || ex.Description == "" || ex.Args == nil {
		t.Error("example missing required MCP fields")
	}
}

func TestDescribeTool_SchemaRequiresTool(t *testing.T) {
	// Schema/full levels require Tool from index, even if docs exist
	store := NewInMemoryStore(StoreOptions{})
	mustRegisterDoc(t, store, "doc-only", DocEntry{
		Summary: "Has summary",
		Notes:   "Has notes",
	})

	// Summary should work
	_, err := store.DescribeTool("doc-only", DetailSummary)
	if err != nil {
		t.Errorf("summary should work without tool: %v", err)
	}

	// Schema should fail with ErrNoTool
	_, err = store.DescribeTool("doc-only", DetailSchema)
	if err == nil {
		t.Error("schema should require tool")
	}
	if !strings.Contains(err.Error(), "tool required") {
		t.Errorf("error = %v, want 'tool required'", err)
	}

	// Full should fail with ErrNoTool
	_, err = store.DescribeTool("doc-only", DetailFull)
	if err == nil {
		t.Error("full should require tool")
	}
	if !strings.Contains(err.Error(), "tool required") {
		t.Errorf("error = %v, want 'tool required'", err)
	}
}

func TestMaxExamples_ListExamples(t *testing.T) {
	store := NewInMemoryStore(StoreOptions{MaxExamples: 2})
	mustRegisterDoc(t, store, "test", DocEntry{
		Summary: "test",
		Examples: []ToolExample{
			{Title: "Ex1"}, {Title: "Ex2"}, {Title: "Ex3"}, {Title: "Ex4"},
		},
	})

	// max=0 should use MaxExamples (2)
	examples, err := store.ListExamples("test", 0)
	if err != nil {
		t.Fatalf("ListExamples failed: %v", err)
	}
	if len(examples) != 2 {
		t.Errorf("max=0: len=%d, want 2 (MaxExamples)", len(examples))
	}

	// max=1 should use 1 (lower than MaxExamples)
	examples, err = store.ListExamples("test", 1)
	if err != nil {
		t.Fatalf("ListExamples failed: %v", err)
	}
	if len(examples) != 1 {
		t.Errorf("max=1: len=%d, want 1", len(examples))
	}

	// max=10 should use MaxExamples (2, lower than 10)
	examples, err = store.ListExamples("test", 10)
	if err != nil {
		t.Fatalf("ListExamples failed: %v", err)
	}
	if len(examples) != 2 {
		t.Errorf("max=10: len=%d, want 2 (MaxExamples)", len(examples))
	}
}

func TestMaxExamples_RegisterExamples(t *testing.T) {
	store := NewInMemoryStore(StoreOptions{MaxExamples: 2})

	err := store.RegisterExamples("test", []ToolExample{
		{Title: "Ex1"}, {Title: "Ex2"}, {Title: "Ex3"},
	})
	if err != nil {
		t.Fatalf("RegisterExamples failed: %v", err)
	}

	examples, err := store.ListExamples("test", 0)
	if err != nil {
		t.Fatalf("ListExamples failed: %v", err)
	}
	if len(examples) != 2 {
		t.Errorf("len=%d, want 2 (MaxExamples)", len(examples))
	}
}

func TestMaxExamples_DescribeTool(t *testing.T) {
	idx := index.NewInMemoryIndex()
	tool := makeToolWithSchema("test", "ns", "desc", map[string]any{"type": "object"})
	backend := model.ToolBackend{
		Kind:  model.BackendKindLocal,
		Local: &model.LocalBackend{Name: "h"},
	}
	if err := idx.RegisterTool(tool, backend); err != nil {
		t.Fatalf("failed to register tool: %v", err)
	}

	store := NewInMemoryStore(StoreOptions{Index: idx, MaxExamples: 2})
	mustRegisterDoc(t, store, "ns:test", DocEntry{
		Summary: "test",
		Examples: []ToolExample{
			{Title: "Ex1"}, {Title: "Ex2"}, {Title: "Ex3"},
		},
	})

	doc, err := store.DescribeTool("ns:test", DetailFull)
	if err != nil {
		t.Fatalf("DescribeTool failed: %v", err)
	}
	if len(doc.Examples) != 2 {
		t.Errorf("full examples len=%d, want 2 (MaxExamples)", len(doc.Examples))
	}
}

func TestArgsDeepCopy_Isolation(t *testing.T) {
	store := NewInMemoryStore(StoreOptions{})

	originalArgs := map[string]any{"key": "original"}
	mustRegisterDoc(t, store, "test", DocEntry{
		Summary:  "test",
		Examples: []ToolExample{{Title: "Ex", Args: originalArgs}},
	})

	// Mutate original after registration
	originalArgs["key"] = "mutated"

	// Stored value should be unaffected
	examples, err := store.ListExamples("test", 10)
	if err != nil {
		t.Fatalf("ListExamples failed: %v", err)
	}
	if examples[0].Args["key"] != "original" {
		t.Errorf("stored args mutated: got %v, want 'original'", examples[0].Args["key"])
	}

	// Mutate returned value
	examples[0].Args["key"] = "returned-mutated"

	// Subsequent retrieval should be unaffected
	examples2, err := store.ListExamples("test", 10)
	if err != nil {
		t.Fatalf("ListExamples failed: %v", err)
	}
	if examples2[0].Args["key"] != "original" {
		t.Errorf("returned args not isolated: got %v, want 'original'", examples2[0].Args["key"])
	}
}

func TestArgsDeepCopy_NestedStructures(t *testing.T) {
	store := NewInMemoryStore(StoreOptions{})

	nestedMap := map[string]any{"inner": "value"}
	nestedSlice := []any{"a", "b", "c"}
	originalArgs := map[string]any{
		"nested_map":   nestedMap,
		"nested_slice": nestedSlice,
		"number":       42,
	}

	mustRegisterDoc(t, store, "test", DocEntry{
		Summary:  "test",
		Examples: []ToolExample{{Title: "Ex", Args: originalArgs}},
	})

	// Mutate nested structures after registration
	nestedMap["inner"] = "mutated"
	nestedSlice[0] = "mutated"

	// Stored nested values should be unaffected
	examples, err := store.ListExamples("test", 10)
	if err != nil {
		t.Fatalf("ListExamples failed: %v", err)
	}
	innerMap := examples[0].Args["nested_map"].(map[string]any)
	if innerMap["inner"] != "value" {
		t.Errorf("nested map mutated: got %v, want 'value'", innerMap["inner"])
	}
	innerSlice := examples[0].Args["nested_slice"].([]any)
	if innerSlice[0] != "a" {
		t.Errorf("nested slice mutated: got %v, want 'a'", innerSlice[0])
	}

	// Mutate returned nested structures
	innerMap["inner"] = "returned-mutated"
	innerSlice[0] = "returned-mutated"

	// Subsequent retrieval should be unaffected
	examples2, err := store.ListExamples("test", 10)
	if err != nil {
		t.Fatalf("ListExamples failed: %v", err)
	}
	innerMap2 := examples2[0].Args["nested_map"].(map[string]any)
	if innerMap2["inner"] != "value" {
		t.Errorf("returned nested map not isolated: got %v, want 'value'", innerMap2["inner"])
	}
	innerSlice2 := examples2[0].Args["nested_slice"].([]any)
	if innerSlice2[0] != "a" {
		t.Errorf("returned nested slice not isolated: got %v, want 'a'", innerSlice2[0])
	}
}

func TestArgsDeepCopy_TypedSlicesAndMaps(t *testing.T) {
	store := NewInMemoryStore(StoreOptions{})

	// Test typed slices and maps (common in Go literals)
	// These should be normalized to MCP-native shapes ([]any, map[string]any)
	typedStringSlice := []string{"a", "b", "c"}
	typedIntSlice := []int{1, 2, 3}
	typedStringMap := map[string]string{"key": "value"}

	originalArgs := map[string]any{
		"string_slice": typedStringSlice,
		"int_slice":    typedIntSlice,
		"string_map":   typedStringMap,
	}

	mustRegisterDoc(t, store, "test", DocEntry{
		Summary:  "test",
		Examples: []ToolExample{{Title: "Ex", Args: originalArgs}},
	})

	// Mutate original typed structures after registration
	typedStringSlice[0] = "mutated"
	typedIntSlice[0] = 999
	typedStringMap["key"] = "mutated"

	// Stored values should be unaffected and normalized to MCP shapes
	examples, err := store.ListExamples("test", 10)
	if err != nil {
		t.Fatalf("ListExamples failed: %v", err)
	}

	// Check string slice - normalized to []any
	if ss, ok := examples[0].Args["string_slice"].([]any); ok {
		if ss[0] != "a" {
			t.Errorf("string slice mutated: got %v, want 'a'", ss[0])
		}
	} else {
		t.Errorf("string_slice not normalized to []any, got %T", examples[0].Args["string_slice"])
	}

	// Check int slice - normalized to []any
	if is, ok := examples[0].Args["int_slice"].([]any); ok {
		if is[0] != 1 {
			t.Errorf("int slice mutated: got %v, want 1", is[0])
		}
	} else {
		t.Errorf("int_slice not normalized to []any, got %T", examples[0].Args["int_slice"])
	}

	// Check string map - normalized to map[string]any
	if sm, ok := examples[0].Args["string_map"].(map[string]any); ok {
		if sm["key"] != "value" {
			t.Errorf("string map mutated: got %v, want 'value'", sm["key"])
		}
	} else {
		t.Errorf("string_map not normalized to map[string]any, got %T", examples[0].Args["string_map"])
	}
}

func TestConcurrentAccess(t *testing.T) {
	store := NewInMemoryStore(StoreOptions{})

	// Register initial doc
	if err := store.RegisterDoc("test", DocEntry{Summary: "initial"}); err != nil {
		t.Fatalf("failed to register doc: %v", err)
	}

	done := make(chan bool)
	errCh := make(chan error, 100)

	// Concurrent reads
	for i := 0; i < 10; i++ {
		go func() {
			for j := 0; j < 100; j++ {
				if _, err := store.DescribeTool("test", DetailSummary); err != nil {
					errCh <- err
				}
				if _, err := store.ListExamples("test", 10); err != nil {
					errCh <- err
				}
			}
			done <- true
		}()
	}

	// Concurrent writes
	for i := 0; i < 5; i++ {
		go func(_ int) {
			for j := 0; j < 50; j++ {
				if err := store.RegisterDoc("test", DocEntry{Summary: "updated"}); err != nil {
					errCh <- err
				}
				if err := store.RegisterExamples("test", []ToolExample{{Title: "ex"}}); err != nil {
					errCh <- err
				}
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 15; i++ {
		<-done
	}

	close(errCh)
	for err := range errCh {
		t.Errorf("concurrent access error: %v", err)
	}
}

func TestValidateArgs(t *testing.T) {
	tests := []struct {
		name      string
		args      map[string]any
		wantValid bool
		wantDepth int
		wantKeys  int
	}{
		{
			name:      "nil args",
			args:      nil,
			wantValid: true,
			wantDepth: 0,
			wantKeys:  0,
		},
		{
			name:      "empty args",
			args:      map[string]any{},
			wantValid: true,
			wantDepth: 1,
			wantKeys:  0,
		},
		{
			name:      "flat args",
			args:      map[string]any{"a": 1, "b": "two", "c": true},
			wantValid: true,
			wantDepth: 1,
			wantKeys:  3,
		},
		{
			name: "nested map",
			args: map[string]any{
				"level1": map[string]any{
					"level2": map[string]any{
						"value": "deep",
					},
				},
			},
			wantValid: true,
			wantDepth: 3,
			wantKeys:  3, // level1, level2, value
		},
		{
			name: "nested slice with maps",
			args: map[string]any{
				"items": []any{
					map[string]any{"id": 1},
					map[string]any{"id": 2},
				},
			},
			wantValid: true,
			wantDepth: 3, // root -> items -> slice element -> map
			wantKeys:  5, // items=1, 2 slice items=2, id+id=2
		},
		{
			name: "at depth limit",
			args: func() map[string]any {
				// Build structure at exactly MaxArgsDepth
				result := map[string]any{"value": "leaf"}
				for i := 1; i < MaxArgsDepth; i++ {
					result = map[string]any{"nested": result}
				}
				return result
			}(),
			wantValid: true,
			wantDepth: MaxArgsDepth,
		},
		{
			name: "exceeds depth limit",
			args: func() map[string]any {
				// Build structure exceeding MaxArgsDepth
				result := map[string]any{"value": "leaf"}
				for i := 0; i < MaxArgsDepth; i++ {
					result = map[string]any{"nested": result}
				}
				return result
			}(),
			wantValid: false,
			wantDepth: MaxArgsDepth + 1,
		},
		{
			name: "at keys limit",
			args: func() map[string]any {
				result := make(map[string]any, MaxArgsKeys)
				for i := 0; i < MaxArgsKeys; i++ {
					result[string(rune('a'+i%26))+string(rune('0'+i/26))] = i
				}
				return result
			}(),
			wantValid: true,
			wantKeys:  MaxArgsKeys,
		},
		{
			name: "exceeds keys limit",
			args: func() map[string]any {
				result := make(map[string]any, MaxArgsKeys+1)
				for i := 0; i <= MaxArgsKeys; i++ {
					result[string(rune('a'+i%26))+string(rune('0'+i/26))] = i
				}
				return result
			}(),
			wantValid: false,
			wantKeys:  MaxArgsKeys + 1,
		},
		{
			name: "large slice exceeds size limit",
			args: func() map[string]any {
				// Large slice should count items toward size
				largeSlice := make([]any, MaxArgsKeys+1)
				for i := range largeSlice {
					largeSlice[i] = i
				}
				return map[string]any{"items": largeSlice}
			}(),
			wantValid: false,
			wantKeys:  MaxArgsKeys + 2, // 1 key + 51 items = 52
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stats, valid := ValidateArgs(tt.args)

			if valid != tt.wantValid {
				t.Errorf("valid = %v, want %v", valid, tt.wantValid)
			}

			if tt.wantDepth > 0 && stats.Depth != tt.wantDepth {
				t.Errorf("depth = %d, want %d", stats.Depth, tt.wantDepth)
			}

			if tt.wantKeys > 0 && stats.Keys != tt.wantKeys {
				t.Errorf("keys = %d, want %d", stats.Keys, tt.wantKeys)
			}
		})
	}
}

func TestRegisterDoc_ArgsCapsEnforced(t *testing.T) {
	store := NewInMemoryStore(StoreOptions{})

	// Build args that exceed depth limit
	deepArgs := map[string]any{"value": "leaf"}
	for i := 0; i < MaxArgsDepth; i++ {
		deepArgs = map[string]any{"nested": deepArgs}
	}

	err := store.RegisterDoc("test", DocEntry{
		Summary: "test",
		Examples: []ToolExample{
			{Title: "Too Deep", Args: deepArgs},
		},
	})

	if err == nil {
		t.Error("expected error for args exceeding depth limit")
	}
	if !strings.Contains(err.Error(), "args exceeds caps") {
		t.Errorf("error = %v, want 'args exceeds caps'", err)
	}

	// Valid args should succeed
	err = store.RegisterDoc("test2", DocEntry{
		Summary: "test",
		Examples: []ToolExample{
			{Title: "Valid", Args: map[string]any{"key": "value"}},
		},
	})
	if err != nil {
		t.Errorf("unexpected error for valid args: %v", err)
	}
}

func TestRegisterExamples_ArgsCapsEnforced(t *testing.T) {
	store := NewInMemoryStore(StoreOptions{})

	// Build args that exceed keys limit
	manyKeys := make(map[string]any, MaxArgsKeys+1)
	for i := 0; i <= MaxArgsKeys; i++ {
		manyKeys[string(rune('a'+i%26))+string(rune('0'+i/26))] = i
	}

	err := store.RegisterExamples("test", []ToolExample{
		{Title: "Too Many Keys", Args: manyKeys},
	})

	if err == nil {
		t.Error("expected error for args exceeding keys limit")
	}
	if !strings.Contains(err.Error(), "args exceeds caps") {
		t.Errorf("error = %v, want 'args exceeds caps'", err)
	}

	// Valid args should succeed
	err = store.RegisterExamples("test2", []ToolExample{
		{Title: "Valid", Args: map[string]any{"key": "value"}},
	})
	if err != nil {
		t.Errorf("unexpected error for valid args: %v", err)
	}
}
