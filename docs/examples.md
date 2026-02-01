# Examples

tooldiscovery ships with runnable examples that demonstrate each search mode
and documentation level.

## Basic discovery (BM25 + progressive docs)

```bash
go run ./examples/basic
```

Shows:
- Registering tools in the index
- BM25 search results
- Summary vs schema disclosure

## Semantic search

```bash
go run ./examples/semantic
```

Shows:
- Custom embedder stub
- Semantic index + searcher
- Score ordering

## Hybrid search

```bash
go run ./examples/hybrid
```

Shows:
- BM25 + semantic weighted scoring
- Score type tracking

## Full discovery facade

```bash
go run ./examples/full
```

Shows:
- `discovery.Discovery` unified API
- Registration + search + describe flow
