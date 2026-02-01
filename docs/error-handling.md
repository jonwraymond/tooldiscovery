# Error Handling Guide

This document describes the error types and error handling patterns used throughout tooldiscovery.

## Error Types by Package

### index Package

| Error | When Returned | Example Cause |
|-------|---------------|---------------|
| `ErrNotFound` | Tool/backend lookup fails | `GetTool("nonexistent:tool")` |
| `ErrInvalidTool` | Tool validation fails | Empty name, invalid schema |
| `ErrInvalidBackend` | Backend validation fails | MCP backend missing ServerName |
| `ErrInvalidCursor` | Pagination cursor invalid | Malformed or expired cursor |
| `ErrNonDeterministicSearcher` | SearchPage with non-deterministic searcher | Custom searcher without stable ordering |

### search Package

The search package returns errors from the underlying Bleve index but doesn't define custom error types. Errors are typically related to index operations.

### semantic Package

| Error | When Returned | Example Cause |
|-------|---------------|---------------|
| `ErrInvalidSearcher` | Searcher missing components | `NewSearcher(nil, nil)` |
| `ErrInvalidDocumentID` | Document ID is empty | `idx.Add(ctx, Document{})` |
| `ErrInvalidEmbedder` | Embedder is nil | `NewEmbeddingStrategy(nil)` |
| `ErrInvalidHybridConfig` | Invalid hybrid config | Alpha outside [0,1] range |

### tooldoc Package

| Error | When Returned | Example Cause |
|-------|---------------|---------------|
| `ErrNotFound` | Tool not in index/resolver | Unknown tool ID |
| `ErrInvalidDetail` | Invalid detail level | Unrecognized DetailLevel value |
| `ErrNoTool` | No tool source configured | Store without Index or ToolResolver |
| `ErrArgsTooLarge` | Example args exceed limits | Nesting > 5 or keys > 50 |

### discovery Package

| Error | When Returned | Example Cause |
|-------|---------------|---------------|
| `ErrNotFound` | Tool lookup fails | Forwarded from index package |

## Error Checking Patterns

### Using errors.Is

Always use `errors.Is` for error checking, as errors may be wrapped:

```go
tool, backend, err := idx.GetTool("github:create-issue")
if errors.Is(err, index.ErrNotFound) {
    // Tool doesn't exist
    log.Printf("Tool not found: %s", toolID)
    return
}
if err != nil {
    // Other error
    return fmt.Errorf("failed to get tool: %w", err)
}
```

### Handling Validation Errors

Validation errors often wrap the sentinel error with additional context:

```go
err := idx.RegisterTool(tool, backend)
if errors.Is(err, index.ErrInvalidTool) {
    // Tool validation failed - check the error message for details
    log.Printf("Invalid tool: %v", err)
    return
}
if errors.Is(err, index.ErrInvalidBackend) {
    // Backend validation failed
    log.Printf("Invalid backend: %v", err)
    return
}
```

### Handling Search Errors

```go
results, err := searcher.Search(ctx, query)
if errors.Is(err, semantic.ErrInvalidSearcher) {
    // Searcher not properly configured
    log.Fatal("Searcher missing index or strategy")
}
if errors.Is(err, semantic.ErrInvalidEmbedder) {
    // Embedder is nil (for embedding/hybrid strategies)
    log.Fatal("Embedder required for semantic search")
}
if err != nil {
    // May be context cancellation or embedder error
    return fmt.Errorf("search failed: %w", err)
}
```

### Handling Documentation Errors

```go
doc, err := store.DescribeTool(toolID, tooldoc.DetailFull)
if errors.Is(err, tooldoc.ErrNotFound) {
    // Tool not found in index
    return nil, fmt.Errorf("unknown tool: %s", toolID)
}
if errors.Is(err, tooldoc.ErrNoTool) {
    // Store has no way to look up tools
    log.Fatal("Store not configured with Index or ToolResolver")
}
if errors.Is(err, tooldoc.ErrInvalidDetail) {
    // Invalid detail level (shouldn't happen with constants)
    return nil, fmt.Errorf("invalid detail level")
}
```

## Context Errors

Many operations accept a context and honor cancellation:

```go
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

results, err := searcher.Search(ctx, query)
if errors.Is(err, context.DeadlineExceeded) {
    log.Printf("Search timed out")
    return nil, err
}
if errors.Is(err, context.Canceled) {
    log.Printf("Search was canceled")
    return nil, err
}
```

## Wrapping Errors

When propagating errors, wrap them with context:

```go
func (s *MyService) FindTools(query string) ([]Tool, error) {
    results, err := s.discovery.Search(ctx, query, 10)
    if err != nil {
        return nil, fmt.Errorf("tool search failed: %w", err)
    }

    var tools []Tool
    for _, r := range results {
        tool, _, err := s.discovery.GetTool(r.Summary.ID)
        if err != nil {
            // Log but continue - tool may have been removed
            log.Printf("Failed to get tool %s: %v", r.Summary.ID, err)
            continue
        }
        tools = append(tools, tool)
    }
    return tools, nil
}
```

## Validation Before Operations

Validate inputs before expensive operations:

```go
// Validate tool before registration
if tool.Name == "" {
    return fmt.Errorf("tool name is required")
}
if tool.InputSchema == nil {
    return fmt.Errorf("tool InputSchema is required")
}

// Now safe to register
if err := idx.RegisterTool(tool, backend); err != nil {
    return fmt.Errorf("registration failed: %w", err)
}
```

## Error Recovery Patterns

### Graceful Degradation

```go
func (s *MyService) Search(query string) ([]Result, error) {
    // Try hybrid search first
    if s.embedder != nil {
        results, err := s.hybridSearch(query)
        if err == nil {
            return results, nil
        }
        // Log and fall back to BM25
        log.Printf("Hybrid search failed, falling back to BM25: %v", err)
    }

    // Fall back to BM25-only search
    return s.bm25Search(query)
}
```

### Retry with Backoff

```go
func (e *MyEmbedder) EmbedWithRetry(ctx context.Context, text string) ([]float32, error) {
    var lastErr error
    for i := 0; i < 3; i++ {
        vec, err := e.Embed(ctx, text)
        if err == nil {
            return vec, nil
        }
        lastErr = err

        // Don't retry on context errors
        if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
            return nil, err
        }

        // Exponential backoff
        time.Sleep(time.Duration(1<<i) * 100 * time.Millisecond)
    }
    return nil, fmt.Errorf("embedding failed after 3 retries: %w", lastErr)
}
```

## Testing Error Conditions

```go
func TestGetTool_NotFound(t *testing.T) {
    idx := index.NewInMemoryIndex()

    _, _, err := idx.GetTool("nonexistent:tool")

    if !errors.Is(err, index.ErrNotFound) {
        t.Errorf("expected ErrNotFound, got %v", err)
    }
}

func TestRegisterTool_InvalidTool(t *testing.T) {
    idx := index.NewInMemoryIndex()

    // Tool with empty name
    tool := model.Tool{
        Tool: mcp.Tool{
            Name: "", // Invalid
            InputSchema: map[string]any{"type": "object"},
        },
    }

    err := idx.RegisterTool(tool, model.NewMCPBackend("server"))

    if !errors.Is(err, index.ErrInvalidTool) {
        t.Errorf("expected ErrInvalidTool, got %v", err)
    }
}
```
