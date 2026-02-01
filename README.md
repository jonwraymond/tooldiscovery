# tooldiscovery

Discovery layer providing tool registry, search strategies, and progressive
documentation for the ApertureStack tool framework.

## Installation

```bash
go get github.com/jonwraymond/tooldiscovery@latest
```

## Packages

| Package | Purpose |
|---------|---------|
| `discovery` | Unified facade combining index, search, semantic, and tooldoc |
| `index` | Global registry, tool lookup, and search interface |
| `search` | BM25-based full-text search strategy |
| `semantic` | Embedding-based semantic search (optional) |
| `tooldoc` | Progressive documentation with detail levels |

## Quick Start (Discovery Facade)

```go
import (
  "context"
  "github.com/jonwraymond/tooldiscovery/discovery"
)

disc, _ := discovery.New(discovery.Options{})

_ = disc.RegisterTool(tool, backend)

results, _ := disc.Search(context.Background(), "create issue", 5)
for _, r := range results {
  fmt.Printf("[%s] %s\n", r.ScoreType, r.Summary.ID)
}
```

## Documentation

- **MkDocs site**: https://jonwraymond.github.io/tooldiscovery/
- **Schemas and contracts**: `docs/schemas.md`
- **Architecture**: `docs/architecture.md`
- **Examples**: `docs/examples.md`

## Examples

Run the examples from the repo root:

```bash
go run ./examples/basic
go run ./examples/semantic
go run ./examples/hybrid
go run ./examples/full
```

## License

MIT License - see [LICENSE](./LICENSE)
