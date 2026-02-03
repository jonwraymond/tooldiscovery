package discovery

import (
	"context"
	"errors"

	"github.com/jonwraymond/tooldiscovery/index"
	"github.com/jonwraymond/tooldiscovery/provider"
	"github.com/jonwraymond/tooldiscovery/search"
	"github.com/jonwraymond/tooldiscovery/semantic"
	"github.com/jonwraymond/tooldiscovery/tooldoc"
	"github.com/jonwraymond/toolfoundation/adapter"
	"github.com/jonwraymond/toolfoundation/model"
)

// Error values for discovery operations.
var (
	ErrNotFound = errors.New("tool not found")
)

// Options configures a Discovery instance.
type Options struct {
	// Index is the tool registry. If nil, creates a new InMemoryIndex.
	Index index.Index

	// Searcher is the search implementation. If nil, uses BM25Searcher.
	// This is ignored if Embedder is provided (uses HybridSearcher instead).
	Searcher index.Searcher

	// DocStore is the documentation store. If nil, creates a new InMemoryStore.
	DocStore tooldoc.Store

	// ProviderStore is the provider registry. If nil, creates a new InMemoryStore.
	ProviderStore provider.Store

	// Embedder enables hybrid search when provided.
	// If nil, uses BM25-only search.
	Embedder semantic.Embedder

	// HybridAlpha is the BM25 weight for hybrid search (0.0 to 1.0).
	// Semantic weight is 1-HybridAlpha.
	// Default: 0.5 (equal weighting). Only used when Embedder is provided.
	HybridAlpha float64

	// BM25Config configures the BM25 searcher.
	// Only used when Searcher is nil and Embedder is nil.
	BM25Config search.BM25Config

	// MaxExamples is the default maximum number of examples to return.
	// Default: 10
	MaxExamples int
}

// Discovery is the unified facade for tool discovery operations.
// It combines index, search, and documentation functionality.
type Discovery struct {
	idx        index.Index
	searcher   index.Searcher
	compositeS CompositeSearcher // nil if using standard searcher
	docs       *tooldoc.InMemoryStore
	providers  provider.Store
	scoreType  ScoreType
	searchDocs func() []index.SearchDoc
}

// New creates a new Discovery instance with the given options.
func New(opts Options) (*Discovery, error) {
	d := &Discovery{}

	// Setup index
	if opts.Index != nil {
		d.idx = opts.Index
	} else {
		d.idx = index.NewInMemoryIndex()
	}

	// Setup searcher
	if opts.Embedder != nil {
		// Use hybrid search
		alpha := opts.HybridAlpha
		if alpha == 0 {
			alpha = 0.5 // Default to equal weighting
		}
		hybrid, err := NewHybridSearcher(HybridOptions{
			Embedder: opts.Embedder,
			Alpha:    alpha,
		})
		if err != nil {
			return nil, err
		}
		d.searcher = hybrid
		d.compositeS = hybrid
		d.scoreType = ScoreHybrid
	} else if opts.Searcher != nil {
		d.searcher = opts.Searcher
		d.scoreType = ScoreBM25
	} else {
		d.searcher = search.NewBM25Searcher(opts.BM25Config)
		d.scoreType = ScoreBM25
	}

	// Setup doc store
	maxExamples := opts.MaxExamples
	if maxExamples == 0 {
		maxExamples = 10
	}

	if inMemIdx, ok := d.idx.(*index.InMemoryIndex); ok {
		d.docs = tooldoc.NewInMemoryStore(tooldoc.StoreOptions{
			Index:       inMemIdx,
			MaxExamples: maxExamples,
		})
	} else {
		// Create store without index linkage - will need resolver
		d.docs = tooldoc.NewInMemoryStore(tooldoc.StoreOptions{
			MaxExamples: maxExamples,
			ToolResolver: func(id string) (*model.Tool, error) {
				tool, _, err := d.idx.GetTool(id)
				if err != nil {
					return nil, err
				}
				return &tool, nil
			},
		})
	}

	// Setup provider store
	if opts.ProviderStore != nil {
		d.providers = opts.ProviderStore
	} else {
		d.providers = provider.NewInMemoryStore()
	}

	// Setup search doc accessor
	if inMemIdx, ok := d.idx.(*index.InMemoryIndex); ok {
		d.searchDocs = func() []index.SearchDoc {
			docs, _ := inMemIdx.Search("", 1000000) // Get all
			searchDocs := make([]index.SearchDoc, len(docs))
			for i, s := range docs {
				searchDocs[i] = index.SearchDoc{
					ID:      s.ID,
					Summary: s,
				}
			}
			return searchDocs
		}
	}

	return d, nil
}

// RegisterTool registers a tool with its backend and optional documentation.
// If doc is nil, the tool is registered without additional documentation.
func (d *Discovery) RegisterTool(tool model.Tool, backend model.ToolBackend, doc *tooldoc.DocEntry) error {
	if err := d.idx.RegisterTool(tool, backend); err != nil {
		return err
	}

	if doc != nil {
		return d.docs.RegisterDoc(tool.ToolID(), *doc)
	}

	return nil
}

// RegisterTools registers multiple tools with their backends.
func (d *Discovery) RegisterTools(regs []index.ToolRegistration) error {
	return d.idx.RegisterTools(regs)
}

// RegisterDoc registers or updates documentation for a tool.
func (d *Discovery) RegisterDoc(toolID string, doc tooldoc.DocEntry) error {
	return d.docs.RegisterDoc(toolID, doc)
}

// RegisterExamples adds examples to a tool's documentation.
func (d *Discovery) RegisterExamples(toolID string, examples []tooldoc.ToolExample) error {
	return d.docs.RegisterExamples(toolID, examples)
}

// Search performs a search using the configured strategy.
// Returns results ordered by relevance score.
func (d *Discovery) Search(ctx context.Context, query string, limit int) (Results, error) {
	if d.compositeS != nil {
		docs := d.getSearchDocs()
		return d.compositeS.SearchWithScores(ctx, query, limit, docs)
	}

	// Fall back to standard search without scores
	summaries, err := d.idx.Search(query, limit)
	if err != nil {
		return nil, err
	}

	results := make(Results, len(summaries))
	for i, s := range summaries {
		results[i] = Result{
			Summary:   s,
			Score:     0, // No score available from standard searcher
			ScoreType: d.scoreType,
		}
	}

	return results, nil
}

// SearchPage performs paginated search.
func (d *Discovery) SearchPage(ctx context.Context, query string, limit int, cursor string) (Results, string, error) {
	summaries, nextCursor, err := d.idx.SearchPage(query, limit, cursor)
	if err != nil {
		return nil, "", err
	}

	results := make(Results, len(summaries))
	for i, s := range summaries {
		results[i] = Result{
			Summary:   s,
			Score:     0,
			ScoreType: d.scoreType,
		}
	}

	return results, nextCursor, nil
}

// GetTool retrieves a tool by its canonical ID.
func (d *Discovery) GetTool(id string) (model.Tool, model.ToolBackend, error) {
	return d.idx.GetTool(id)
}

// GetAllBackends returns all backends for a tool.
func (d *Discovery) GetAllBackends(id string) ([]model.ToolBackend, error) {
	return d.idx.GetAllBackends(id)
}

// DescribeTool returns documentation at the specified detail level.
func (d *Discovery) DescribeTool(id string, level tooldoc.DetailLevel) (tooldoc.ToolDoc, error) {
	return d.docs.DescribeTool(id, level)
}

// DescribeProvider returns provider metadata by ID.
func (d *Discovery) DescribeProvider(id string) (adapter.CanonicalProvider, error) {
	if d.providers == nil {
		return adapter.CanonicalProvider{}, provider.ErrNotFound
	}
	return d.providers.DescribeProvider(id)
}

// ListProviders returns all registered providers.
func (d *Discovery) ListProviders() ([]adapter.CanonicalProvider, error) {
	if d.providers == nil {
		return nil, provider.ErrNotFound
	}
	return d.providers.ListProviders()
}

// RegisterProvider registers a provider and returns the resolved ID.
func (d *Discovery) RegisterProvider(id string, p adapter.CanonicalProvider) (string, error) {
	if d.providers == nil {
		return "", provider.ErrInvalidProvider
	}
	return d.providers.RegisterProvider(id, p)
}

// ListExamples returns examples for a tool.
func (d *Discovery) ListExamples(id string, maxExamples int) ([]tooldoc.ToolExample, error) {
	return d.docs.ListExamples(id, maxExamples)
}

// ListNamespaces returns all registered namespaces.
func (d *Discovery) ListNamespaces() ([]string, error) {
	return d.idx.ListNamespaces()
}

// OnChange registers a listener for index changes.
// Returns an unsubscribe function.
func (d *Discovery) OnChange(listener index.ChangeListener) func() {
	if notifier, ok := d.idx.(index.ChangeNotifier); ok {
		return notifier.OnChange(listener)
	}
	return func() {}
}

// Index returns the underlying index for advanced operations.
func (d *Discovery) Index() index.Index {
	return d.idx
}

// DocStore returns the underlying documentation store.
func (d *Discovery) DocStore() *tooldoc.InMemoryStore {
	return d.docs
}

// ProviderStore returns the underlying provider store.
func (d *Discovery) ProviderStore() provider.Store {
	return d.providers
}

// getSearchDocs returns the current search documents.
func (d *Discovery) getSearchDocs() []index.SearchDoc {
	if d.searchDocs != nil {
		return d.searchDocs()
	}

	// Fallback: build from index search
	summaries, _ := d.idx.Search("", 1000000)
	docs := make([]index.SearchDoc, len(summaries))
	for i, s := range summaries {
		docs[i] = index.SearchDoc{
			ID:      s.ID,
			Summary: s,
		}
	}
	return docs
}
