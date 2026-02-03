# tooldiscovery User Journey

## Overview

This guide walks through the progressive disclosure pattern that tooldiscovery
enables: discover cheaply, inspect on-demand, then execute.

## 1. Installation

```bash
go get github.com/jonwraymond/tooldiscovery@latest
```

## 2. Set Up the Index

```go
import "github.com/jonwraymond/tooldiscovery/index"

// Create an in-memory index
idx := index.NewInMemoryIndex()
```

## 3. Register Tools

```go
import "github.com/jonwraymond/toolfoundation/model"

// Create and validate a tool
tool := model.Tool{
  Namespace: "github",
  Tool: mcp.Tool{
    Name:        "create_issue",
    Description: "Create a new GitHub issue",
    InputSchema: map[string]any{...},
  },
}

// Define the backend
backend := model.ToolBackend{
  Kind:       model.BackendKindMCP,
  ServerName: "github-mcp",
}

// Register
err := idx.RegisterTool(tool, backend)
```

## 4. Search for Tools (Token-Cheap)

```go
// Search returns summaries without schemas
summaries, err := idx.Search("create issue", 5)
if err != nil {
  log.Fatal(err)
}

for _, s := range summaries {
  fmt.Printf("Found: %s - %s\n", s.ID, s.Summary)
}
```

## 5. Get Full Documentation (On-Demand)

```go
import "github.com/jonwraymond/tooldiscovery/tooldoc"

store := tooldoc.NewInMemoryStore(tooldoc.StoreOptions{Index: idx})

// Progressive detail levels
doc, _ := store.GetDoc("github:create_issue", tooldoc.DetailSummary)
fmt.Println(doc.Summary)

doc, _ = store.GetDoc("github:create_issue", tooldoc.DetailSchema)
fmt.Printf("Input Schema: %v\n", doc.Tool.InputSchema)
```

### Detail Level Guidance

- **Summary**: listing/search results (token-cheap)
- **Schema**: just-in-time execution
- **Full**: documentation view or export

## 6. Enable BM25 Search (Optional)

```go
import "github.com/jonwraymond/tooldiscovery/search"

// Create BM25 searcher with custom config
config := search.Config{
  NameBoost:      4.0,
  NamespaceBoost: 2.0,
  TagBoost:       1.0,
}

searcher, err := search.NewBM25Searcher(config)
if err != nil {
  log.Fatal(err)
}
defer searcher.Close()

// Create index with BM25 strategy
idx := index.NewInMemoryIndex(index.WithSearchStrategy(searcher))
```

## 7. List Namespaces

```go
namespaces := idx.ListNamespaces()
// ["github", "slack", "jira", ...]

// Filter tools by namespace
tools := idx.ListToolsInNamespace("github")
```

## Progressive Disclosure Flow

```
Agent                    MCP Server              tooldiscovery
  |                          |                        |
  |-- search_tools("issue") -|                        |
  |                          |-- idx.Search() --------|
  |                          |<-- []Summary ----------|
  |<- summaries (no schema) -|                        |
  |                          |                        |
  |-- describe_tool(id) -----|                        |
  |                          |-- store.GetDoc() -----|
  |                          |<-- ToolDoc w/schema ---|
  |<-- full schema ----------|                        |
  |                          |                        |
  |-- run_tool(id, args) ----|                        |
  |                          |      (to toolexec)     |
```

## Next Steps

- Execute tools with [toolexec/run](https://github.com/jonwraymond/toolexec)
- Expose via MCP with [metatools-mcp](https://github.com/jonwraymond/metatools-mcp)
