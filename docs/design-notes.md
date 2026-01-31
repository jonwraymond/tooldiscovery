# tooldiscovery Design Notes

## Overview

tooldiscovery provides the discovery layer for the ApertureStack tool framework.
It handles tool registration, search, and progressive documentation.

## index Package

### Design Decisions

1. **Single Source of Truth**: The index is the authoritative registry for all
   registered tools. Tool IDs are derived from `toolfoundation/model.Tool.ToolID()`.

2. **Search Strategy Interface**: Search is pluggable via the `SearchStrategy`
   interface. Default is lexical substring matching.

3. **Token-Efficient Summaries**: `Search()` returns `Summary` objects that
   exclude schemas, keeping discovery cheap in terms of LLM context tokens.

4. **Namespace Isolation**: Tools are grouped by namespace, enabling filtered
   views and multi-tenant scenarios.

### Error Handling

- Duplicate tool registration returns an error
- Invalid tool IDs return descriptive errors
- Search errors are logged but don't fail the request

## search Package

### Design Decisions

1. **BM25 Algorithm**: Uses Okapi BM25 for relevance ranking, implemented via
   the Bleve search library.

2. **Field Boosting**: Configurable boosts for name (4x), namespace (2x), and
   tags (1x) fields.

3. **Build Tag Gating**: BM25 support requires the `toolsearch` build tag to
   keep the default binary minimal.

### Configuration

| Config | Default | Description |
|--------|---------|-------------|
| NameBoost | 4.0 | Boost for tool name matches |
| NamespaceBoost | 2.0 | Boost for namespace matches |
| TagBoost | 1.0 | Boost for tag matches |
| MaxDocs | 0 | Max docs to index (0=unlimited) |

## semantic Package

### Design Decisions

1. **Embedder Interface**: Abstracts the embedding model, allowing different
   providers (OpenAI, local models, etc.).

2. **Vector Store Interface**: Abstracts the vector storage, supporting
   in-memory, file-based, or external stores.

3. **Optional Dependency**: Semantic search is opt-in and requires additional
   setup (embeddings, vector store).

## tooldoc Package

### Design Decisions

1. **Detail Levels**: Four progressive levels:
   - `Summary`: Name, namespace, short description
   - `Description`: Full description, tags
   - `Schema`: Input/output JSON schemas
   - `Full`: Everything including examples

2. **On-Demand Loading**: Schemas are only loaded when requested at
   `DetailSchema` or `DetailFull` level.

3. **Index Integration**: DocStore can use an Index to derive documentation
   from registered tools.

## Dependencies

- `github.com/jonwraymond/toolfoundation/model` - Tool definitions
- `github.com/blevesearch/bleve/v2` - BM25 search (optional)

## Links

- [index](index.md)
- [user journey](user-journey.md)
