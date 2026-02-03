# Architecture Overview

This document describes the component architecture of tooldiscovery and how the packages interact.

## Package Hierarchy

```
┌─────────────────────────────────────────────────────────────────┐
│                         discovery                                │
│              (Unified Facade - Recommended Entry Point)          │
└──────────────────────────┬──────────────────────────────────────┘
                           │
         ┌─────────────────┼─────────────────┐
         │                 │                 │
         v                 v                 v
┌─────────────┐    ┌──────────────┐   ┌──────────┐
│    index    │◄───│   semantic   │   │  tooldoc │
│  (Registry) │    │  (Scoring)   │   │  (Docs)  │
└──────┬──────┘    └──────────────┘   └────┬─────┘
       │                                    │
       v                                    │
┌──────────────┐                           │
│    search    │◄──────────────────────────┘
│   (BM25)     │
└──────┬───────┘
       │
       v
┌──────────────────┐
│  toolfoundation  │
│     (model)      │
└──────────────────┘
```

## Package Responsibilities

### `discovery` - Unified Facade

The recommended entry point for most use cases. Combines all packages into a simple API.

**Provides:**
- Single `Discovery` type with unified operations
- Built-in hybrid search (BM25 + semantic)
- Integrated documentation management
- Result filtering helpers

**Key Types:**
- `Discovery` - Main facade
- `Options` - Configuration
- `Result` / `Results` - Search results with scores
- `HybridSearcher` - Composite searcher

### `registry` - MCP Server Helper

High-level helper for building MCP servers. It composes `index` + `search` with
local execution handlers and MCP backend aggregation.

**Provides:**
- Local tool registration with handlers
- MCP backend connections and tool aggregation
- MCP protocol handlers (`initialize`, `tools/list`, `tools/call`)
- Transports (`ServeStdio`, `ServeHTTP`, `ServeSSE`)

**Key Types:**
- `Registry` - Core registry + lifecycle
- `ToolHandler` - Local execution handler
- `BackendConfig` - MCP backend connection config

### `index` - Tool Registry

Core registry for tool storage, lookup, and search orchestration.

**Provides:**
- Tool registration with backends
- Canonical ID generation (`namespace:name:version` when version is set)
- Pluggable search via `Searcher` interface
- Change notifications
- Pagination support

**Key Types:**
- `Index` - Registry interface
- `InMemoryIndex` - Default implementation
- `Searcher` - Search strategy interface
- `Summary` / `SearchDoc` - Lightweight search results

### `search` - BM25 Implementation

Production-ready BM25 search using Bleve.

**Provides:**
- Full-text search with field boosting
- Configurable term weighting
- Deterministic ordering for pagination
- Efficient index caching

**Key Types:**
- `BM25Searcher` - Main searcher (implements `index.Searcher`)
- `BM25Config` - Boost configuration

### `semantic` - Embedding Search

Pluggable semantic search with no external dependencies.

**Provides:**
- Strategy pattern for scoring (BM25, embedding, hybrid)
- Document indexing for semantic operations
- Bring-your-own-embedder support
- Namespace/tag filtering

**Key Types:**
- `Strategy` - Scoring interface
- `Embedder` - User-provided embedding generator
- `Indexer` - Document storage interface
- `Document` - Semantic document model

### `tooldoc` - Documentation Store

Progressive disclosure documentation system.

**Provides:**
- Three detail levels (summary, schema, full)
- Example storage and validation
- Schema information extraction
- Integration with index for tool lookup

**Key Types:**
- `Store` - Documentation interface
- `InMemoryStore` - Default implementation
- `DetailLevel` - Disclosure granularity
- `ToolDoc` / `DocEntry` - Documentation types

## Data Flow

### Tool Registration

```
model.Tool + ToolBackend
        │
        v
┌───────────────┐
│ index.Index   │──────► Stores tool + backend
│ RegisterTool  │──────► Normalizes tags
└───────┬───────┘──────► Builds SearchDoc
        │               ──────► Notifies listeners
        v
┌───────────────┐
│ tooldoc.Store │──────► Stores documentation
│ RegisterDoc   │──────► Validates examples
└───────────────┘
```

### Search Flow

```
Query String
     │
     v
┌────────────────┐
│ index.Search   │◄─── Gets SearchDoc snapshot
└───────┬────────┘
        │
        v
┌────────────────┐
│ Searcher.Search│◄─── BM25Searcher or HybridSearcher
└───────┬────────┘
        │
        v
┌────────────────┐
│ Strategy.Score │◄─── BM25, Embedding, or Hybrid
└───────┬────────┘
        │
        v
   []Summary (sorted by score, then ID)
```

### Progressive Disclosure

```
Tool ID
   │
   ├──► DetailSummary ──► Summary only (cheap)
   │
   ├──► DetailSchema  ──► Summary + Tool + SchemaInfo
   │
   └──► DetailFull    ──► Everything + Notes + Examples
```

## Interface Contracts

### index.Searcher

```go
type Searcher interface {
    Search(query string, limit int, docs []SearchDoc) ([]Summary, error)
}
```

**Contract:**
- Must be safe for concurrent use
- Must return deterministic ordering (score desc, ID asc)
- Must handle empty query (return first N docs)
- Must respect limit parameter

**Implementations:**
- `search.BM25Searcher` - Bleve-based full-text search
- `discovery.HybridSearcher` - BM25 + semantic combination
- `discovery.BM25OnlySearcher` - Semantic BM25 with scores

### index.DeterministicSearcher

```go
type DeterministicSearcher interface {
    Searcher
    Deterministic() bool
}
```

Required for pagination support. Implementations that return `true` guarantee stable ordering across calls with identical inputs.

### semantic.Strategy

```go
type Strategy interface {
    Score(ctx context.Context, query string, doc Document) (float64, error)
}
```

**Contract:**
- Must honor context cancellation
- Must return deterministic scores
- Must be safe for concurrent use

**Implementations:**
- `bm25Strategy` - Token overlap scoring
- `embeddingStrategy` - Cosine similarity of embeddings
- `hybridStrategy` - Weighted combination

### semantic.Embedder

```go
type Embedder interface {
    Embed(ctx context.Context, text string) ([]float32, error)
}
```

**Contract:**
- Must honor context cancellation
- Must return consistent-length vectors
- Must be safe for concurrent use

User-provided implementations connect to embedding services (OpenAI, Ollama, etc.).

## Type Mapping

### index.SearchDoc ↔ semantic.Document

The `semantic/adapter.go` provides conversion between package types:

| index.SearchDoc | semantic.Document |
|-----------------|-------------------|
| ID | ID |
| DocText | Text |
| Summary.Name | Name |
| Summary.Namespace | Namespace |
| Summary.Summary (fallback ShortDescription) | Description |
| Summary.Tags | Tags |
| Summary.Category | Category |

## Configuration Patterns

### Minimal Setup (BM25 only)

```go
idx := index.NewInMemoryIndex()  // Uses built-in lexical searcher
```

### BM25 with Custom Config

```go
searcher := search.NewBM25Searcher(search.BM25Config{
    NameBoost: 3,
    TagsBoost: 2,
})
idx := index.NewInMemoryIndex(index.IndexOptions{
    Searcher: searcher,
})
```

### Hybrid Search via Discovery

```go
disc, _ := discovery.New(discovery.Options{
    Embedder:    myEmbedder,
    HybridAlpha: 0.7,  // 70% BM25, 30% semantic
})
```

## Extension Points

1. **Custom Searcher**: Implement `index.Searcher` for alternative search backends
2. **Custom Embedder**: Implement `semantic.Embedder` for any embedding provider
3. **Custom Strategy**: Implement `semantic.Strategy` for custom scoring logic
4. **Custom Backend Selector**: Provide `BackendSelector` function to `IndexOptions`
5. **Change Listeners**: Subscribe via `OnChange` for reactive integrations
