// Package semantic provides semantic indexing and search for tool discovery.
//
// It defines pluggable search strategies (BM25, embeddings, hybrid) without
// enforcing any specific vector backend or network dependency. This allows
// users to bring their own embedding provider (OpenAI, Ollama, local models).
//
// # Core Interfaces
//
// The package defines four key interfaces:
//
//   - [Strategy]: Scores a document against a query (BM25, embedding, or hybrid)
//   - [Searcher]: Performs search over indexed documents using a strategy
//   - [Indexer]: Stores and retrieves documents for search
//   - [Embedder]: Generates vector embeddings from text (user-provided)
//
// # Search Strategies
//
// Three built-in strategies are provided:
//
//   - BM25 (lexical): Token overlap scoring, no external dependencies
//   - Embedding (semantic): Cosine similarity of vector embeddings
//   - Hybrid: Weighted combination of BM25 and embedding scores
//
// Create strategies using the constructor functions:
//
//	bm25 := semantic.NewBM25Strategy(nil)           // nil uses default scorer
//	emb := semantic.NewEmbeddingStrategy(embedder)  // requires Embedder impl
//	hybrid, _ := semantic.NewHybridStrategy(bm25, emb, 0.7)  // 70% BM25
//
// # Document Model
//
// [Document] represents a tool for semantic indexing:
//
//	doc := semantic.Document{
//	    ID:          "github:create-issue",
//	    Name:        "create-issue",
//	    Namespace:   "github",
//	    Description: "Create a new GitHub issue",
//	    Tags:        []string{"github", "issues"},
//	    Category:    "vcs",
//	}
//
// Use [Document.Normalized] to prepare documents for indexing, which
// lowercases tags, sorts them, and builds the combined Text field.
//
// # Basic Usage
//
//	// Create index and add documents
//	idx := semantic.NewInMemoryIndex()
//	idx.Add(ctx, doc1)
//	idx.Add(ctx, doc2)
//
//	// Create searcher with BM25 strategy
//	strategy := semantic.NewBM25Strategy(nil)
//	searcher := semantic.NewSearcher(idx, strategy)
//
//	// Search
//	results, err := searcher.Search(ctx, "create issue")
//	for _, r := range results {
//	    fmt.Printf("[%.2f] %s\n", r.Score, r.Document.ID)
//	}
//
// # Implementing Custom Embedder
//
// To use embedding-based or hybrid search, implement the [Embedder] interface:
//
//	type MyEmbedder struct {
//	    client *openai.Client
//	}
//
//	func (e *MyEmbedder) Embed(ctx context.Context, text string) ([]float32, error) {
//	    resp, err := e.client.CreateEmbedding(ctx, openai.EmbeddingRequest{
//	        Model: "text-embedding-3-small",
//	        Input: []string{text},
//	    })
//	    if err != nil {
//	        return nil, err
//	    }
//	    return resp.Data[0].Embedding, nil
//	}
//
// # Filtering Results
//
// Use the filter functions to narrow results:
//
//	docs := idx.List(ctx)
//	gitDocs := semantic.FilterByNamespace(docs, "git")
//	vcsDocs := semantic.FilterByTags(docs, []string{"vcs"})
//
// # Integration with index Package
//
// The [adapter.go] file provides conversion between index.SearchDoc and
// semantic.Document, enabling seamless integration:
//
//	// Convert index docs to semantic docs
//	semDocs := semantic.DocumentsFromSearchDocs(searchDocs)
//
//	// Convert back after processing
//	searchDocs := semantic.SearchDocsFromDocuments(semDocs)
//
// # Thread Safety
//
// All types in this package are safe for concurrent use:
//   - [InMemoryIndex] uses sync.RWMutex for thread-safe document storage
//   - [InMemorySearcher] is stateless and safe for concurrent Search calls
//   - All Strategy implementations are safe for concurrent Score calls
//
// # Error Handling
//
// The package defines these sentinel errors:
//   - [ErrInvalidSearcher]: Searcher missing index or strategy
//   - [ErrInvalidDocumentID]: Document ID is empty
//   - [ErrInvalidEmbedder]: Embedder is nil when required
//   - [ErrInvalidHybridConfig]: Invalid hybrid strategy configuration
//
// Use errors.Is for error checking:
//
//	if errors.Is(err, semantic.ErrInvalidEmbedder) {
//	    // handle missing embedder
//	}
package semantic
