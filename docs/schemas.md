# Schemas and Data Contracts

tooldiscovery does **not** define new JSON Schemas for tool input/output.
Those come from **toolfoundation/model.Tool** and are treated as opaque,
validated payloads. What tooldiscovery *does* define are the **data
contracts** that shape discovery, documentation, and search outputs.

This page documents those contracts, their constraints, and how they relate
to the underlying tool schemas.

## Canonical tool schema dependency

- `model.Tool` is the canonical tool record (from **toolfoundation**).
- `InputSchema` is required; `OutputSchema` is optional.
- tooldiscovery never mutates schemas — it passes them through for
  **describe** or **execution** flows.

If you need the JSON Schema contracts, see the
**toolfoundation** schema docs.

## Summary schema (index.Summary)

`Summary` is the minimal discovery payload returned from search and list calls.

Fields:

| Field | Type | Notes |
|-------|------|-------|
| `id` | string | Canonical tool ID (`namespace:name:version`, `namespace:name`, or `name`) |
| `name` | string | Tool name |
| `namespace` | string | Optional namespace |
| `shortDescription` | string | Truncated to 120 chars |
| `summary` | string | Short summary (mirrors shortDescription) |
| `category` | string | Optional category label |
| `inputModes` | []string | Supported input media types |
| `outputModes` | []string | Supported output media types |
| `securitySummary` | string | Short auth scheme summary |
| `tags` | []string | Normalized tags |

Constraints:

- `shortDescription` is capped by `index.MaxShortDescriptionLen` (120).
- `summary` mirrors the shortDescription payload for search results.
- `inputModes`, `outputModes`, and `securitySummary` are derived from tool metadata.
- `tags` are normalized and deduplicated by the index.
- `Summary` never includes schemas.

## SearchDoc schema (index.SearchDoc)

`SearchDoc` is the internal/searcher payload used to score results.

Fields:

| Field | Type | Notes |
|-------|------|-------|
| `ID` | string | Canonical tool ID |
| `DocText` | string | Lowercased concatenation of name, namespace, description, summary, category, modes, tags |
| `Summary` | Summary | Prebuilt summary returned to callers |

Contracts:

- `DocText` must be deterministic for the same tool.
- `SearchDoc` is read-only for searchers; mutating it is forbidden.

## Documentation schema (tooldoc.ToolDoc)

`ToolDoc` is the progressive documentation payload returned by `tooldoc.Store`.

Fields:

| Field | Type | Notes |
|-------|------|-------|
| `tool` | *model.Tool | Present for `schema`/`full` levels |
| `summary` | string | Capped at 200 chars |
| `inputModes` | []string | Supported input media types |
| `outputModes` | []string | Supported output media types |
| `securitySummary` | string | Short auth scheme summary |
| `annotations` | map[string]any | Tool annotations for UI hints |
| `schemaInfo` | *SchemaInfo | Derived from input schema |
| `notes` | string | Capped at 2000 chars |
| `examples` | []ToolExample | Optional usage examples |
| `externalRefs` | []string | URLs or resource IDs |

### SchemaInfo

Derived from a tool’s **InputSchema** (best effort):

| Field | Type | Notes |
|-------|------|-------|
| `required` | []string | Required parameter names |
| `defaults` | map[string]any | Default values |
| `types` | map[string][]string | Allowed types by param |

### ToolExample

Usage examples are bounded to prevent context bloat:

- `Description` max 300 chars
- `ResultHint` max 200 chars
- `Args` capped at depth **5** and size **50** (keys + items)

## Detail levels (tooldoc.DetailLevel)

| Level | Contents |
|-------|----------|
| `summary` | Summary only |
| `schema` | Summary + tool + schema info |
| `full` | Schema + notes + examples + external refs |

## Discovery results (discovery.Result)

The discovery facade wraps summaries with scoring metadata:

| Field | Type | Notes |
|-------|------|-------|
| `summary` | Summary | Tool metadata |
| `score` | float64 | Relevance score |
| `scoreType` | string | `bm25`, `embedding`, or `hybrid` |

## Semantic document contract (semantic.Document)

Semantic search operates on normalized `Document` payloads:

| Field | Type | Notes |
|-------|------|-------|
| `id` | string | Canonical tool ID |
| `namespace` | string | Optional |
| `name` | string | Tool name |
| `description` | string | Short description |
| `tags` | []string | Lowercased + sorted |
| `category` | string | Optional category |
| `text` | string | Normalized search text |

`Document.Normalized()` lowercases and sorts tags and builds `text`.

## JSON Schema guidance

For JSON Schema input/output contract details, reference:

- **toolfoundation** schema docs
- `model.Tool` in **toolfoundation/model**
