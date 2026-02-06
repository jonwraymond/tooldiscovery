# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.3.2](https://github.com/jonwraymond/tooldiscovery/compare/v0.3.1...v0.3.2) (2026-02-06)


### Documentation

* update version matrix ([f738b8b](https://github.com/jonwraymond/tooldiscovery/commit/f738b8b819368e7ee713a736613c0dcf784f1307))

## [0.3.1](https://github.com/jonwraymond/tooldiscovery/compare/v0.3.0...v0.3.1) (2026-02-03)


### Documentation

* update version matrix ([#9](https://github.com/jonwraymond/tooldiscovery/issues/9)) ([092792e](https://github.com/jonwraymond/tooldiscovery/commit/092792e833a1fa1a895f6a09a1e1aea8452a0e34))

## [Unreleased]

### Added

#### New `discovery` Package
- Unified facade combining index, search, semantic, and tooldoc packages
- `Discovery` type with simplified API for tool registration and search
- `HybridSearcher` combining BM25 and semantic search with configurable alpha
- `BM25OnlySearcher` for semantic-style BM25 with score tracking
- `Results` type with filtering helpers (`FilterByNamespace`, `FilterByMinScore`)
- Score type tracking (`ScoreBM25`, `ScoreEmbedding`, `ScoreHybrid`)

#### Semantic-Index Bridge
- `semantic/adapter.go` with conversion functions between packages
- `DocumentFromSearchDoc` / `SearchDocFromDocument` for type conversion
- `DocumentsFromSearchDocs` / `SearchDocsFromDocuments` for slice conversion
- Round-trip preservation of key fields

#### Example Tests (pkg.go.dev Documentation)
- `index/example_test.go` - 7 examples covering core index operations
- `semantic/example_test.go` - 8 examples covering semantic search
- `tooldoc/example_test.go` - 6 examples covering documentation store
- `discovery/example_test.go` - 6 examples covering unified facade

#### Benchmark Suite
- `index/bench_test.go` - Registration, search, lookup, and concurrent benchmarks
- `semantic/bench_test.go` - Strategy, indexer, and filter benchmarks
- `tooldoc/bench_test.go` - Documentation retrieval and validation benchmarks

#### Runnable Examples
- `examples/basic/main.go` - BM25 search with progressive disclosure
- `examples/semantic/main.go` - Semantic search with custom embedder
- `examples/hybrid/main.go` - Hybrid search with alpha comparison
- `examples/full/main.go` - Complete Discovery facade workflow

#### Documentation
- `docs/architecture.md` - Package hierarchy and data flow diagrams
- `docs/error-handling.md` - Error types and handling patterns
- `docs/concurrency.md` - Thread safety guarantees and patterns
- `docs/performance.md` - Tuning guide with benchmark results
- `docs/migration.md` - Migration guide from toolindex and custom registries
- Expanded `semantic/doc.go` with comprehensive package documentation

### Changed
- Updated `toolfoundation` dependency from v0.1.0 to v0.2.0

## [0.1.0] - 2026-01-31

### Added
- Initial release with four core packages

#### `index` Package
- `Index` interface for tool registry operations
- `InMemoryIndex` implementation with thread-safe storage
- `Searcher` interface for pluggable search strategies
- `DeterministicSearcher` for pagination support
- Change notification via `OnChange` listener pattern
- Cursor-based pagination with `SearchPage` and `ListNamespacesPage`
- Default lexical searcher for basic substring matching
- Tool ID format: `namespace:name:version`, `namespace:name`, or just `name`

#### `search` Package
- `BM25Searcher` using Bleve for full-text search
- Configurable field boosting (name, namespace, tags)
- Fingerprint-based caching for efficient repeated searches
- Deterministic ordering (score desc, ID asc) for stable pagination

#### `semantic` Package
- `Strategy` interface for pluggable scoring algorithms
- `Indexer` interface for document storage
- `Searcher` interface for semantic search operations
- `Embedder` interface for user-provided embedding generation
- Built-in strategies: BM25, Embedding, Hybrid
- `InMemoryIndex` for thread-safe document storage
- `InMemorySearcher` for deterministic search
- Filter functions for namespace and tag filtering

#### `tooldoc` Package
- `Store` interface for documentation storage
- `InMemoryStore` implementation
- Three detail levels: Summary, Schema, Full
- Progressive disclosure pattern for efficient documentation access
- Example validation with depth and size limits
- Schema information extraction from JSON Schema
- Integration with index package for tool lookup

### Dependencies
- `github.com/jonwraymond/toolfoundation` v0.1.0
- `github.com/blevesearch/bleve/v2` v2.5.7
- `github.com/modelcontextprotocol/go-sdk` v1.2.0
