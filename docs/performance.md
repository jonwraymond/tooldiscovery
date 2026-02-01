# Performance Tuning Guide

This guide covers performance characteristics and tuning options for tooldiscovery.

## Search Strategy Selection

Choose the right search strategy based on your needs:

| Strategy | Speed | Quality | Dependencies | Use When |
|----------|-------|---------|--------------|----------|
| Lexical (built-in) | Fastest | Basic | None | Simple substring matching |
| BM25 | Fast | Good | Bleve | Production text search |
| Embedding | Slow | Best | Embedder | Semantic similarity needed |
| Hybrid | Medium | Best | Embedder | Balance of speed and quality |

### Recommendation by Corpus Size

| Corpus Size | Recommended Strategy | Rationale |
|-------------|---------------------|-----------|
| < 100 tools | Lexical or BM25 | Low overhead, fast enough |
| 100-1000 tools | BM25 | Good balance of speed and quality |
| 1000+ tools | BM25 or Hybrid | May benefit from semantic ranking |

## BM25 Configuration

### Field Boost Values

The `BM25Config` controls how different fields affect ranking:

```go
searcher := search.NewBM25Searcher(search.BM25Config{
    NameBoost:      3,  // Name matches are 3x more important
    NamespaceBoost: 2,  // Namespace matches are 2x
    TagsBoost:      2,  // Tag matches are 2x
})
```

**Guidelines:**
- Higher `NameBoost` (3-5): Prefer exact tool name matches
- Higher `TagsBoost` (2-3): Prefer keyword/category matches
- Higher `NamespaceBoost` (2-3): Group related tools together

### Corpus Size Limits

For very large corpora, use limits to control memory:

```go
searcher := search.NewBM25Searcher(search.BM25Config{
    MaxDocs:       5000,  // Limit indexed documents
    MaxDocTextLen: 1000,  // Truncate long descriptions
})
```

**Trade-offs:**
- `MaxDocs`: Older documents may be excluded from search
- `MaxDocTextLen`: Long descriptions truncated (may miss relevant terms)

## Benchmark Results

Representative benchmarks on Apple M4 Max (run with `go test -bench=.`):

### Index Operations

| Operation | Corpus Size | Time/Op | Notes |
|-----------|-------------|---------|-------|
| RegisterTool | N/A | ~900ns | Single tool registration |
| RegisterTool (sequential) | Growing | ~1.5μs | With index growth |
| GetTool | 1000 | ~110ns | Hash map lookup |
| ListNamespaces | 1000 | ~210ns | Set iteration |

### Search Operations

| Operation | Corpus Size | Time/Op | Notes |
|-----------|-------------|---------|-------|
| Search (lexical) | 1000 | ~80μs | Built-in searcher |
| Search (BM25 cold) | 1000 | ~52ms | First search, builds index |
| Search (BM25 warm) | 1000 | ~580μs | Cached index |
| Search (BM25) | 100 | ~7μs | Small corpus |
| Search (BM25) | 500 | ~49μs | Medium corpus |
| Search (BM25) | 2000 | ~1.2ms | Large corpus |

### Semantic Operations

| Operation | Corpus Size | Time/Op | Notes |
|-----------|-------------|---------|-------|
| BM25 Strategy Score | 1 | ~700ns | Per document |
| Embedding Strategy Score | 1 | ~1μs | Per document (mock embedder) |
| Hybrid Search | 1000 | ~1.9ms | With mock embedder |

### Documentation Operations

| Operation | Detail Level | Time/Op | Notes |
|-----------|--------------|---------|-------|
| DescribeTool | Summary | ~250ns | Minimal data |
| DescribeTool | Schema | ~260ns | With schema info |
| DescribeTool | Full | ~250ns | All details |
| ListExamples | N/A | ~215ns | Example retrieval |

## Optimization Strategies

### 1. Warm Up BM25 Index

The BM25 searcher builds its Bleve index on first search. Warm it up at startup:

```go
func warmupSearcher(idx *index.InMemoryIndex) {
    // Trigger index build with empty query
    _, _ = idx.Search("", 1)
}
```

### 2. Batch Tool Registration

Use `RegisterTools` for batch registration instead of individual calls:

```go
// Slower: individual registrations
for _, tool := range tools {
    idx.RegisterTool(tool, backend)
}

// Faster: batch registration
regs := make([]index.ToolRegistration, len(tools))
for i, tool := range tools {
    regs[i] = index.ToolRegistration{Tool: tool, Backend: backend}
}
idx.RegisterTools(regs)
```

### 3. Cache Embeddings

For hybrid search, cache embeddings to avoid repeated API calls:

```go
type CachedEmbedder struct {
    embedder semantic.Embedder
    cache    sync.Map
}

func (e *CachedEmbedder) Embed(ctx context.Context, text string) ([]float32, error) {
    if vec, ok := e.cache.Load(text); ok {
        return vec.([]float32), nil
    }
    vec, err := e.embedder.Embed(ctx, text)
    if err != nil {
        return nil, err
    }
    e.cache.Store(text, vec)
    return vec, nil
}
```

### 4. Use Progressive Disclosure

Fetch only the detail level you need:

```go
// For search result display - use Summary (cheapest)
for _, r := range results {
    summary, _ := store.DescribeTool(r.ID, tooldoc.DetailSummary)
    displayResult(summary.Summary)
}

// Only fetch Full when user selects a tool
full, _ := store.DescribeTool(selectedID, tooldoc.DetailFull)
displayFullDoc(full)
```

### 5. Limit Search Results

Always specify a reasonable limit:

```go
// Good: limited results
results, _ := idx.Search(query, 10)

// Bad: potentially returns entire corpus
results, _ := idx.Search(query, 1000000)
```

### 6. Use Pagination for Large Result Sets

```go
var allResults []index.Summary
cursor := ""

for {
    page, nextCursor, err := idx.SearchPage(query, 50, cursor)
    if err != nil {
        break
    }
    allResults = append(allResults, page...)
    if nextCursor == "" {
        break
    }
    cursor = nextCursor
}
```

## Memory Considerations

### Index Memory Usage

Approximate memory per tool:
- `InMemoryIndex`: ~500 bytes (tool metadata + search doc)
- `BM25Searcher`: +200-500 bytes (Bleve index entry)
- `tooldoc.InMemoryStore`: +100-500 bytes (documentation)

**Estimate:** ~1KB per tool with full documentation

### Search Doc Caching

The index caches search documents. This is rebuilt when:
- New tools are registered
- Tools are updated
- Backends are removed
- `Refresh()` is called

Force a refresh if you need immediate consistency:

```go
idx.Refresh() // Rebuilds search doc cache
```

## Profiling

Use Go's built-in profiling to identify bottlenecks:

```go
import _ "net/http/pprof"

func main() {
    go func() {
        log.Println(http.ListenAndServe("localhost:6060", nil))
    }()
    // ... your code ...
}
```

Then profile with:

```bash
# CPU profile
go tool pprof http://localhost:6060/debug/pprof/profile?seconds=30

# Memory profile
go tool pprof http://localhost:6060/debug/pprof/heap

# Goroutine profile
go tool pprof http://localhost:6060/debug/pprof/goroutine
```

## Running Benchmarks

```bash
# Run all benchmarks
go test ./... -bench=. -benchmem

# Run specific package benchmarks
go test ./index -bench=. -benchmem
go test ./search -bench=. -benchmem
go test ./semantic -bench=. -benchmem

# Run with longer duration for stable results
go test ./... -bench=. -benchtime=5s

# Save baseline for comparison
go test ./... -bench=. > benchmark_baseline.txt

# Compare after changes
go test ./... -bench=. > benchmark_new.txt
benchstat benchmark_baseline.txt benchmark_new.txt
```
