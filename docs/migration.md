# Migration Guide

This guide helps you migrate to tooldiscovery from other tool discovery systems.

## Migrating from toolindex

The `tooldiscovery/index` package is the successor to `github.com/jonwraymond/toolindex`. The migration is straightforward as the API is largely compatible.

### Import Changes

```go
// Before
import "github.com/jonwraymond/toolindex"

// After
import "github.com/jonwraymond/tooldiscovery/index"
```

### Type Mapping

| toolindex | tooldiscovery/index | Notes |
|-----------|---------------------|-------|
| `Index` | `Index` | Same interface |
| `InMemoryIndex` | `InMemoryIndex` | Same implementation |
| `Summary` | `Summary` | Same fields |
| `SearchDoc` | `SearchDoc` | Same fields |
| `Searcher` | `Searcher` | Same interface |
| `ToolRegistration` | `ToolRegistration` | Same struct |

### New Features

tooldiscovery adds:
- `discovery` package for unified facade
- `semantic` package for embedding-based search
- `tooldoc` package for progressive documentation
- `search` package with production BM25

### Code Changes

Most code works unchanged:

```go
// Before
idx := toolindex.NewInMemoryIndex()
idx.RegisterTool(tool, backend)
results, _ := idx.Search("query", 10)

// After (identical)
idx := index.NewInMemoryIndex()
idx.RegisterTool(tool, backend)
results, _ := idx.Search("query", 10)
```

### Using Discovery Facade (Recommended)

For new code, use the `discovery` package:

```go
// New recommended approach
disc, _ := discovery.New(discovery.Options{})
disc.RegisterTool(tool, backend, &tooldoc.DocEntry{
    Summary: "Tool description",
})
results, _ := disc.Search(ctx, "query", 10)
```

## Migrating from Custom Tool Registries

If you have a custom tool registry, here's how to migrate:

### Step 1: Map Your Tool Type

Create a function to convert your tools to `model.Tool`:

```go
func convertTool(myTool MyTool) model.Tool {
    return model.Tool{
        Tool: mcp.Tool{
            Name:        myTool.Name,
            Description: myTool.Description,
            InputSchema: myTool.Schema,
        },
        Namespace: myTool.Category, // Map to namespace
        Tags:      myTool.Keywords, // Map to tags
    }
}
```

### Step 2: Map Your Backend Type

```go
func convertBackend(myBackend MyBackend) model.ToolBackend {
    switch myBackend.Type {
    case "mcp":
        return model.NewMCPBackend(myBackend.ServerName)
    case "local":
        return model.NewLocalBackend(myBackend.HandlerName)
    case "external":
        return model.NewProviderBackend(myBackend.Provider, myBackend.ID)
    default:
        return model.NewMCPBackend("default")
    }
}
```

### Step 3: Migrate Registration

```go
// Create index
idx := index.NewInMemoryIndex()

// Migrate existing tools
for _, myTool := range myRegistry.GetAllTools() {
    tool := convertTool(myTool)
    backend := convertBackend(myTool.Backend)

    if err := idx.RegisterTool(tool, backend); err != nil {
        log.Printf("Failed to migrate %s: %v", myTool.Name, err)
        continue
    }
}
```

### Step 4: Migrate Search

```go
// Before (custom registry)
results := myRegistry.Search(query)

// After
summaries, _ := idx.Search(query, 100)
```

### Step 5: Add Documentation (Optional)

```go
store := tooldoc.NewInMemoryStore(tooldoc.StoreOptions{Index: idx})

for _, myTool := range myRegistry.GetAllTools() {
    toolID := fmt.Sprintf("%s:%s", myTool.Category, myTool.Name)

    store.RegisterDoc(toolID, tooldoc.DocEntry{
        Summary:  myTool.ShortHelp,
        Notes:    myTool.LongHelp,
        Examples: convertExamples(myTool.Examples),
    })
}
```

## Integrating with Existing MCP Servers

### Receiving Tools from MCP Server

```go
// When you receive tools from an MCP server
func onToolsReceived(serverName string, mcpTools []mcp.Tool) {
    // Convert to model.Tool (which embeds mcp.Tool)
    tools := make([]model.Tool, len(mcpTools))
    for i, t := range mcpTools {
        tools[i] = model.Tool{Tool: t}
    }

    // Register all tools from this server
    if err := idx.RegisterToolsFromMCP(serverName, tools); err != nil {
        log.Printf("Failed to register tools from %s: %v", serverName, err)
    }
}
```

### Handling Server Disconnect

```go
// When an MCP server disconnects, remove its tools
func onServerDisconnect(serverName string) {
    // Get all tools and find ones from this server
    // (You'd need to track this mapping separately)
    for _, toolID := range toolsFromServer[serverName] {
        idx.UnregisterBackend(toolID, model.BackendKindMCP, serverName)
    }
}
```

## Using with toolfoundation v0.2.0

tooldiscovery requires toolfoundation v0.2.0+:

```bash
go get github.com/jonwraymond/toolfoundation@v0.2.0
```

Key features from toolfoundation used by tooldiscovery:
- `model.Tool` with embedded `mcp.Tool`
- `model.ToolBackend` with MCP/Provider/Local variants
- `model.NormalizeTags` for consistent tag handling
- Backend factory functions (`NewMCPBackend`, etc.)

## Gradual Migration Strategy

For large codebases, migrate gradually:

### Phase 1: Add tooldiscovery Alongside Existing

```go
// Keep existing registry
oldRegistry := myRegistry.New()

// Add tooldiscovery
idx := index.NewInMemoryIndex()

// Sync tools to both
func registerTool(tool MyTool) {
    oldRegistry.Register(tool)
    idx.RegisterTool(convertTool(tool), convertBackend(tool.Backend))
}
```

### Phase 2: Shadow Read

```go
// Search both, compare results
func search(query string) []MyTool {
    oldResults := oldRegistry.Search(query)

    // Shadow: also search new index
    newResults, _ := idx.Search(query, 100)
    logComparison(oldResults, newResults)

    return oldResults // Still use old results
}
```

### Phase 3: Switch Read Path

```go
func search(query string) []MyTool {
    summaries, _ := idx.Search(query, 100)
    return convertToMyTools(summaries)
}
```

### Phase 4: Remove Old Registry

```go
// Only use tooldiscovery
idx := index.NewInMemoryIndex()
// ... registration and search ...
```

## Common Migration Issues

### Tool ID Format

tooldiscovery uses `namespace:name:version` format for tool IDs when version is set (otherwise `namespace:name`):

```go
// Tool with namespace "git" and name "status"
toolID := "git:status"

// Tool without namespace
toolID := "simple_tool"
```

Ensure your code handles both formats.

### Search Result Differences

tooldiscovery's BM25 search may return different results than simple substring matching. Tune the `BM25Config` to match your expected behavior:

```go
// For behavior closer to substring matching
searcher := search.NewBM25Searcher(search.BM25Config{
    NameBoost:      5,  // Strongly prefer name matches
    NamespaceBoost: 1,
    TagsBoost:      1,
})
```

### Backend Selection

If you have multiple backends per tool, customize the selector:

```go
idx := index.NewInMemoryIndex(index.IndexOptions{
    BackendSelector: func(backends []model.ToolBackend) model.ToolBackend {
        // Your custom logic
        for _, b := range backends {
            if b.Kind == model.BackendKindLocal {
                return b // Prefer local backends
            }
        }
        return backends[0]
    },
})
```
