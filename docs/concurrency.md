# Concurrency Guide

All tooldiscovery types are safe for concurrent use. This document explains the concurrency model and provides examples of safe concurrent patterns.

## Thread Safety Guarantees

### index Package

| Type | Thread Safety | Implementation |
|------|---------------|----------------|
| `InMemoryIndex` | Safe | `sync.RWMutex` |
| `Summary` | Immutable | Value type |
| `SearchDoc` | Immutable | Value type |
| `ChangeEvent` | Immutable | Value type |

**Operations:**
- `RegisterTool`: Write lock (exclusive)
- `GetTool`: Read lock (shared)
- `Search`: Read lock (shared)
- `OnChange`: Write lock for subscription, callbacks run outside lock

### search Package

| Type | Thread Safety | Implementation |
|------|---------------|----------------|
| `BM25Searcher` | Safe | `sync.RWMutex` + fingerprint caching |

**Operations:**
- `Search`: Read lock for cache check, write lock only on cache miss
- `Close`: Write lock (exclusive)

### semantic Package

| Type | Thread Safety | Implementation |
|------|---------------|----------------|
| `InMemoryIndex` | Safe | `sync.RWMutex`, point-in-time snapshots |
| `InMemorySearcher` | Safe | Stateless |
| All strategies | Safe | Stateless |

**Note:** `InMemoryIndex.List()` holds the read lock for the entire operation to ensure point-in-time snapshot consistency.

### tooldoc Package

| Type | Thread Safety | Implementation |
|------|---------------|----------------|
| `InMemoryStore` | Safe | `sync.RWMutex` |
| `ToolDoc` | Immutable | Value type (deep copied) |
| `DocEntry` | Immutable | Value type |

### discovery Package

| Type | Thread Safety | Implementation |
|------|---------------|----------------|
| `Discovery` | Safe | Delegates to thread-safe components |
| `HybridSearcher` | Safe | Stateless |
| `Results` | Immutable | Value type |

## Concurrent Usage Patterns

### Concurrent Search

Multiple goroutines can search simultaneously:

```go
idx := index.NewInMemoryIndex()
// ... register tools ...

var wg sync.WaitGroup
queries := []string{"git", "docker", "kubernetes", "python", "javascript"}

for _, query := range queries {
    wg.Add(1)
    go func(q string) {
        defer wg.Done()

        results, err := idx.Search(q, 10)
        if err != nil {
            log.Printf("Search failed for %q: %v", q, err)
            return
        }
        log.Printf("Found %d results for %q", len(results), q)
    }(query)
}

wg.Wait()
```

### Concurrent Registration and Search

Searches can run while registrations happen:

```go
idx := index.NewInMemoryIndex()
ctx, cancel := context.WithCancel(context.Background())
defer cancel()

// Background registration
go func() {
    for i := 0; i < 1000; i++ {
        tool := createTool(i)
        if err := idx.RegisterTool(tool, backend); err != nil {
            log.Printf("Registration failed: %v", err)
        }
        time.Sleep(10 * time.Millisecond)
    }
}()

// Concurrent searches
for i := 0; i < 10; i++ {
    go func() {
        for {
            select {
            case <-ctx.Done():
                return
            default:
                results, _ := idx.Search("tool", 10)
                log.Printf("Found %d tools", len(results))
                time.Sleep(50 * time.Millisecond)
            }
        }
    }()
}
```

### Concurrent Documentation Access

```go
store := tooldoc.NewInMemoryStore(tooldoc.StoreOptions{Index: idx})

var wg sync.WaitGroup
toolIDs := []string{"git:status", "docker:ps", "kubectl:get"}

for _, id := range toolIDs {
    wg.Add(1)
    go func(toolID string) {
        defer wg.Done()

        // These can all run concurrently
        summary, _ := store.DescribeTool(toolID, tooldoc.DetailSummary)
        schema, _ := store.DescribeTool(toolID, tooldoc.DetailSchema)
        full, _ := store.DescribeTool(toolID, tooldoc.DetailFull)

        log.Printf("%s: summary=%q, hasSchema=%v, notes=%q",
            toolID, summary.Summary, schema.SchemaInfo != nil, full.Notes)
    }(id)
}

wg.Wait()
```

### Using Discovery Facade Concurrently

```go
disc, _ := discovery.New(discovery.Options{})
// ... register tools ...

// Concurrent hybrid search
results := make(chan discovery.Results, 10)
queries := []string{"create issue", "list containers", "deploy app"}

for _, q := range queries {
    go func(query string) {
        ctx := context.Background()
        r, err := disc.Search(ctx, query, 10)
        if err != nil {
            log.Printf("Search error: %v", err)
            results <- nil
            return
        }
        results <- r
    }(q)
}

for range queries {
    r := <-results
    if r != nil {
        log.Printf("Got %d results", len(r))
    }
}
```

## Change Notification Safety

Change listeners are called outside the index lock to prevent deadlocks:

```go
idx := index.NewInMemoryIndex()

// Safe: listener doesn't hold locks
unsubscribe := idx.OnChange(func(event index.ChangeEvent) {
    // This runs outside the index lock
    log.Printf("Tool %s: %s", event.ToolID, event.Type)

    // Safe to call index methods (they acquire their own locks)
    if event.Type == index.ChangeRegistered {
        tool, _, _ := idx.GetTool(event.ToolID)
        log.Printf("Registered: %s", tool.Description)
    }
})
defer unsubscribe()

// Registrations trigger callbacks
idx.RegisterTool(tool, backend)
```

**Warning:** Don't hold your own locks when calling index methods from listeners, as this can cause deadlocks.

## Embedder Concurrency

Custom embedders must be thread-safe:

```go
type CachedEmbedder struct {
    client *openai.Client
    cache  sync.Map // Thread-safe cache
}

func (e *CachedEmbedder) Embed(ctx context.Context, text string) ([]float32, error) {
    // Check cache (thread-safe)
    if cached, ok := e.cache.Load(text); ok {
        return cached.([]float32), nil
    }

    // Make API call
    resp, err := e.client.CreateEmbedding(ctx, openai.EmbeddingRequest{
        Model: "text-embedding-3-small",
        Input: []string{text},
    })
    if err != nil {
        return nil, err
    }

    vec := resp.Data[0].Embedding

    // Store in cache (thread-safe)
    e.cache.Store(text, vec)

    return vec, nil
}
```

## Avoiding Common Pitfalls

### Don't Hold Locks Across Calls

```go
// BAD: Holding your own lock while calling index methods
func (s *MyService) badPattern() {
    s.mu.Lock()
    defer s.mu.Unlock()

    // This acquires the index lock while holding s.mu
    // If another goroutine holds index lock and tries to acquire s.mu,
    // you have a deadlock
    results, _ := s.idx.Search("query", 10)
    s.cache = results
}

// GOOD: Release your lock before calling index methods
func (s *MyService) goodPattern() {
    results, _ := s.idx.Search("query", 10)

    s.mu.Lock()
    s.cache = results
    s.mu.Unlock()
}
```

### Don't Modify Returned Slices

```go
// BAD: Modifying returned slice
results, _ := idx.Search("query", 10)
results[0].Name = "modified" // Don't do this!

// GOOD: Copy if you need to modify
results, _ := idx.Search("query", 10)
myResults := make([]index.Summary, len(results))
copy(myResults, results)
myResults[0].Name = "modified" // Safe
```

### Use Context for Timeouts

```go
// GOOD: Use context to prevent hanging on slow embedders
func (s *MyService) SearchWithTimeout(query string) ([]Result, error) {
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    return s.searcher.Search(ctx, query)
}
```

## Benchmarking Concurrent Performance

Use the provided benchmarks to measure concurrent performance:

```bash
# Run concurrent benchmarks
go test ./index -bench=Concurrent -benchtime=5s

# Example output:
# BenchmarkIndex_Concurrent_Search-8     50000    25000 ns/op
# BenchmarkIndex_Concurrent_Mixed-8      30000    40000 ns/op
```

The benchmarks use `b.RunParallel` to test concurrent access patterns.
