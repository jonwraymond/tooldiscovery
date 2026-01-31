# tooldiscovery

Discovery layer providing tool registry, search strategies, and progressive
documentation for the ApertureStack tool framework.

## Packages

| Package | Purpose |
|---------|---------|
| `index` | Global registry, tool lookup, and search interface |
| `search` | BM25-based full-text search strategy |
| `semantic` | Embedding-based semantic search (optional) |
| `tooldoc` | Progressive documentation with detail levels |

## Installation

```bash
go get github.com/jonwraymond/tooldiscovery@latest
```

## Quick Start

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

### Progressive Documentation

```go
import "github.com/jonwraymond/tooldiscovery/tooldoc"

store := tooldoc.NewInMemoryStore(tooldoc.StoreOptions{Index: idx})

// Get summary only (token-cheap)
doc, _ := store.GetDoc(toolID, tooldoc.DetailSummary)

// Get full schema (on-demand)
doc, _ = store.GetDoc(toolID, tooldoc.DetailSchema)
```

## Key Features

- **Token-efficient**: Summaries exclude schemas to reduce context usage
- **Pluggable search**: Swap between lexical, BM25, or semantic search
- **Progressive disclosure**: Request only the detail level needed
- **Namespace support**: List and filter tools by namespace

## Links

- [design notes](design-notes.md)
- [user journey](user-journey.md)
- [ai-tools-stack documentation](https://jonwraymond.github.io/ai-tools-stack/)
