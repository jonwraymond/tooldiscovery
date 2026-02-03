# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## 1.0.0 (2026-02-03)


### Features

* expand discovery facade and docs ([9cf784f](https://github.com/jonwraymond/tooldiscovery/commit/9cf784f9cff784c0c53094017f13eb3ff1190837))
* **index:** migrate toolindex package ([e27371a](https://github.com/jonwraymond/tooldiscovery/commit/e27371aab76e755093f4587694a20c67c1b09ed6))
* initial repository structure ([2e010c5](https://github.com/jonwraymond/tooldiscovery/commit/2e010c51fb1773c7913c1dfaf442ea0dacd91668))
* provider discovery + canonical summaries ([#8](https://github.com/jonwraymond/tooldiscovery/issues/8)) ([73b1d55](https://github.com/jonwraymond/tooldiscovery/commit/73b1d552efe367acb9627cdcfd7cfa2435d24c9a))
* **registry:** implement MCP backend support ([a540fcb](https://github.com/jonwraymond/tooldiscovery/commit/a540fcb2592df4bb6643a6568911a7645ce05289))
* **search:** migrate toolsearch package ([34c742a](https://github.com/jonwraymond/tooldiscovery/commit/34c742a1d93b4be74403a67ef6e672db9f3cf257))
* **semantic:** migrate toolsemantic package ([10961ef](https://github.com/jonwraymond/tooldiscovery/commit/10961efb89e14c57e55518770e4ae28e7ca524c9))
* **tooldoc:** migrate tooldocs package ([0fa5b33](https://github.com/jonwraymond/tooldiscovery/commit/0fa5b337fa9973e6fcb26d13b3d1b718e1c4f651))


### Bug Fixes

* **deps:** use toolfoundation v0.1.0, remove local replace directive ([da18a6d](https://github.com/jonwraymond/tooldiscovery/commit/da18a6d00b96d61c71651234e9c1b9a32069136b))


### Documentation

* add mkdocs config ([4d47d65](https://github.com/jonwraymond/tooldiscovery/commit/4d47d650314cae15c2a0ee53795c62b6ef43bcd0))
* align discovery docs and README ([b7f622c](https://github.com/jonwraymond/tooldiscovery/commit/b7f622ca575e32da885480214b213f02ce9a5362))
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
