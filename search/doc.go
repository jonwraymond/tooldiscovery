// Package search provides BM25-based search implementations for the index package.
//
// It exists to:
//   - Keep index small and dependency-light
//   - Enable stronger ranking strategies without forcing heavier search
//     dependencies on every consumer
//
// # Usage
//
// The primary type is [BM25Searcher], which implements [index.Searcher]:
//
//	idx := index.NewInMemoryIndex(index.IndexOptions{
//	    Searcher: search.NewBM25Searcher(search.BM25Config{}),
//	})
//
// # Configuration
//
// [BM25Config] allows customization of field boosts and safety limits:
//
//	cfg := search.BM25Config{
//	    NameBoost:      3,   // Boost name matches (default: 3)
//	    NamespaceBoost: 2,   // Boost namespace matches (default: 2)
//	    TagsBoost:      2,   // Boost tag matches (default: 2)
//	    MaxDocs:        1000, // Limit documents to index (0 = unlimited)
//	    MaxDocTextLen:  5000, // Truncate long descriptions (0 = unlimited)
//	}
//
// # Thread Safety
//
// BM25Searcher is safe for concurrent use. It uses an internal RWMutex to
// protect index state and efficiently caches the Bleve index based on document
// fingerprints, only rebuilding when the document set changes.
//
// # Behavior
//
// Empty queries return the first N documents (matching index's default behavior).
// Non-empty queries use BM25 ranking with deterministic tie-breaking (score DESC,
// then ID ASC).
package search
