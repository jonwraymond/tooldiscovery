# tooldiscovery

Discovery layer providing tool registry, search strategies, and progressive
documentation for the ApertureStack tool framework.

## Packages

| Package | Purpose |
|---------|---------|
| `discovery` | Unified facade combining index, search, semantic, and tooldoc |
| `index` | Global registry, tool lookup, and search interface |
| `search` | BM25-based full-text search strategy |
| `semantic` | Embedding-based semantic search (optional) |
| `tooldoc` | Progressive documentation with detail levels |
| `registry` | MCP server helper with local + backend execution |

## Installation

```bash
go get github.com/jonwraymond/tooldiscovery@latest
```

## Quick Start

### Use the Discovery Facade (Recommended)

```go
import (
  "context"
  "github.com/jonwraymond/tooldiscovery/discovery"
)

disc, _ := discovery.New(discovery.Options{})

// Register tools through the facade
_ = disc.RegisterTool(tool, backend)

// Search (hybrid-ready)
results, _ := disc.Search(context.Background(), "create issue", 5)
for _, r := range results {
  fmt.Printf("[%s] %s\n", r.ScoreType, r.Summary.ID)
}
```

### Build an MCP Server (registry)

```go
import (
  "context"
  "github.com/jonwraymond/tooldiscovery/registry"
)

reg := registry.New(registry.Config{
  ServerInfo: registry.ServerInfo{Name: "my-mcp", Version: "1.0.0"},
})

_ = reg.RegisterLocalFunc(
  "echo",
  "Echo input",
  map[string]any{"type": "object"},
  func(ctx context.Context, args map[string]any) (any, error) { return args, nil },
)

_ = reg.Start(context.Background())
defer reg.Stop()
```

### Register and Search Tools

```go
import (
  "github.com/jonwraymond/tooldiscovery/index"
  "github.com/jonwraymond/toolfoundation/model"
)

// Create an index
idx := index.NewInMemoryIndex()

// Register a tool
err := idx.RegisterTool(tool, backend)
if err != nil {
  log.Fatal(err)
}

// Search for tools
summaries, err := idx.Search("create issue", 5)
for _, s := range summaries {
  fmt.Printf("%s: %s\n", s.ID, s.ShortDescription)
}
```

### Enable BM25 Search

```go
import (
  "github.com/jonwraymond/tooldiscovery/index"
  "github.com/jonwraymond/tooldiscovery/search"
)

// Create BM25 searcher
searcher, err := search.NewBM25Searcher(search.DefaultConfig())
if err != nil {
  log.Fatal(err)
}
defer searcher.Close()

// Create index with BM25
idx := index.NewInMemoryIndex(index.WithSearchStrategy(searcher))
```

### Search Strategy Guidance

- **Lexical** (default): simple substring matching; best for small registries.
- **BM25** (`search`): higher quality ranking for larger registries.
- **Semantic** (`semantic`): intent-based matching when embeddings are available.
- **Hybrid** (`discovery`): combines BM25 + semantic with weighted scoring.

### Progressive Documentation

```go
import "github.com/jonwraymond/tooldiscovery/tooldoc"

store := tooldoc.NewInMemoryStore(tooldoc.StoreOptions{Index: idx})

// Get summary only (token-cheap)
doc, _ := store.GetDoc(toolID, tooldoc.DetailSummary)

// Get full schema (on-demand)
doc, _ = store.GetDoc(toolID, tooldoc.DetailSchema)
```

### Semantic Search (Optional)

```go
import "github.com/jonwraymond/tooldiscovery/semantic"

// Provide an Embedder + VectorStore implementation
searcher := semantic.NewSemanticSearcher(embedder, vectorStore)
idx := index.NewInMemoryIndex(index.WithSearchStrategy(searcher))
```

## Key Features

- **Token-efficient**: Summaries exclude schemas to reduce context usage
- **Pluggable search**: Swap between lexical, BM25, or semantic search
- **Progressive disclosure**: Request only the detail level needed
- **Namespace support**: List and filter tools by namespace

## Links

- [design notes](design-notes.md)
- [user journey](user-journey.md)
- [schemas and contracts](schemas.md)
- [architecture](architecture.md)
- [registry](registry.md)
- [concurrency](concurrency.md)
- [error handling](error-handling.md)
- [performance](performance.md)
- [migration](migration.md)
- [ai-tools-stack documentation](https://jonwraymond.github.io/ai-tools-stack/)
