package registry

import (
	"context"
	"fmt"
	"sync"

	"github.com/jonwraymond/tooldiscovery/index"
	"github.com/jonwraymond/tooldiscovery/search"
	"github.com/jonwraymond/toolfoundation/model"
)

// Config configures a Registry.
type Config struct {
	SearchConfig    *search.BM25Config
	ServerInfo      ServerInfo
	BackendSelector index.BackendSelector
}

// ServerInfo describes this MCP server for initialize response.
type ServerInfo struct {
	Name    string
	Version string
}

// Registry is a high-level MCP tool registry with built-in search,
// local tool registration, and MCP backend connection.
type Registry struct {
	mu       sync.RWMutex
	index    *index.InMemoryIndex
	searcher *search.BM25Searcher
	config   Config

	handlers map[string]ToolHandler
	backends map[string]*mcpBackend

	started bool
	stopCh  chan struct{}
}

// New creates a new Registry with the given config.
func New(cfg Config) *Registry {
	indexOpts := index.IndexOptions{}
	if cfg.BackendSelector != nil {
		indexOpts.BackendSelector = cfg.BackendSelector
	}

	searcher := search.NewBM25Searcher(search.BM25Config{})
	if cfg.SearchConfig != nil {
		searcher = search.NewBM25Searcher(*cfg.SearchConfig)
	}
	indexOpts.Searcher = searcher

	idx := index.NewInMemoryIndex(indexOpts)

	return &Registry{
		index:    idx,
		searcher: searcher,
		config:   cfg,
		handlers: make(map[string]ToolHandler),
		backends: make(map[string]*mcpBackend),
		stopCh:   make(chan struct{}),
	}
}

// RegisterLocal registers a tool with a local execution handler.
func (r *Registry) RegisterLocal(tool model.Tool, handler ToolHandler) error {
	if err := tool.Validate(); err != nil {
		return fmt.Errorf("invalid tool: %w", err)
	}

	backend := model.NewLocalBackend(tool.Name)
	if err := r.index.RegisterTool(tool, backend); err != nil {
		return err
	}

	r.mu.Lock()
	r.handlers[tool.ToolID()] = handler
	r.mu.Unlock()

	return nil
}

// RegisterLocalFunc is a convenience for inline tool definition.
func (r *Registry) RegisterLocalFunc(
	name, description string,
	inputSchema map[string]any,
	handler ToolHandler,
	opts ...LocalToolOption,
) error {
	cfg := applyLocalToolOptions(opts)
	tool := buildLocalTool(name, description, inputSchema, cfg)
	return r.RegisterLocal(tool, handler)
}

// Search performs a BM25 search and returns ranked tools.
func (r *Registry) Search(ctx context.Context, query string, limit int) ([]model.Tool, error) {
	summaries, err := r.index.Search(query, limit)
	if err != nil {
		return nil, err
	}

	tools := make([]model.Tool, 0, len(summaries))
	for _, summary := range summaries {
		tool, _, err := r.index.GetTool(summary.ID)
		if err != nil {
			continue
		}
		tools = append(tools, tool)
	}
	return tools, nil
}

// SearchSummaries returns lightweight summaries (faster for listing).
func (r *Registry) SearchSummaries(ctx context.Context, query string, limit int) ([]index.Summary, error) {
	return r.index.Search(query, limit)
}

// ListAll returns all registered tools.
func (r *Registry) ListAll(ctx context.Context) ([]model.Tool, error) {
	summaries, err := r.index.Search("", 10000)
	if err != nil {
		return nil, err
	}

	tools := make([]model.Tool, 0, len(summaries))
	for _, summary := range summaries {
		tool, _, err := r.index.GetTool(summary.ID)
		if err != nil {
			continue
		}
		tools = append(tools, tool)
	}
	return tools, nil
}

// ListNamespaces returns all tool namespaces.
func (r *Registry) ListNamespaces(ctx context.Context) ([]string, error) {
	return r.index.ListNamespaces()
}

// GetTool returns a tool by ID.
func (r *Registry) GetTool(ctx context.Context, id string) (model.Tool, error) {
	tool, _, err := r.index.GetTool(id)
	if err != nil {
		return model.Tool{}, err
	}
	return tool, nil
}

// Execute runs a tool by name with the given arguments.
func (r *Registry) Execute(ctx context.Context, name string, args map[string]any) (any, error) {
	tool, backend, err := r.index.GetTool(name)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrToolNotFound, name)
	}

	switch backend.Kind {
	case model.BackendKindLocal:
		r.mu.RLock()
		handler, ok := r.handlers[tool.ToolID()]
		r.mu.RUnlock()
		if !ok {
			return nil, fmt.Errorf("%w: %s", ErrHandlerNotFound, tool.ToolID())
		}
		return handler(ctx, args)

	case model.BackendKindMCP:
		if backend.MCP == nil {
			return nil, fmt.Errorf("%w: MCP backend missing server name", ErrInvalidRequest)
		}
		r.mu.RLock()
		mcpBackend, ok := r.backends[backend.MCP.ServerName]
		r.mu.RUnlock()
		if !ok {
			return nil, fmt.Errorf("%w: %s", ErrBackendNotFound, backend.MCP.ServerName)
		}
		return mcpBackend.callTool(ctx, tool.Name, args)

	default:
		return nil, fmt.Errorf("%w: backend kind %s not supported", ErrInvalidRequest, backend.Kind)
	}
}

// Start initializes the registry and connects MCP backends.
func (r *Registry) Start(ctx context.Context) error {
	r.mu.Lock()
	if r.started {
		r.mu.Unlock()
		return ErrAlreadyStarted
	}
	r.started = true
	r.stopCh = make(chan struct{})
	backends := make(map[string]*mcpBackend, len(r.backends))
	for name, backend := range r.backends {
		backends[name] = backend
	}
	r.mu.Unlock()

	connected := make([]string, 0, len(backends))
	for name, backend := range backends {
		if err := backend.connect(ctx); err != nil {
			for _, connectedName := range connected {
				_ = backends[connectedName].disconnect()
			}
			r.mu.Lock()
			r.started = false
			r.mu.Unlock()
			return fmt.Errorf("failed to connect backend %s: %w", name, err)
		}
		connected = append(connected, name)
		if err := r.index.RegisterToolsFromMCP(name, backend.toolsSnapshot()); err != nil {
			for _, connectedName := range connected {
				_ = backends[connectedName].disconnect()
			}
			r.mu.Lock()
			r.started = false
			r.mu.Unlock()
			return fmt.Errorf("failed to register backend %s tools: %w", name, err)
		}
	}

	return nil
}

// Stop gracefully shuts down all backend connections.
func (r *Registry) Stop() error {
	r.mu.Lock()
	if !r.started {
		r.mu.Unlock()
		return nil
	}
	r.started = false
	close(r.stopCh)
	backends := make(map[string]*mcpBackend, len(r.backends))
	for name, backend := range r.backends {
		backends[name] = backend
	}
	r.mu.Unlock()

	for name, backend := range backends {
		if err := backend.disconnect(); err != nil {
			return fmt.Errorf("failed to disconnect backend %s: %w", name, err)
		}
	}

	return nil
}

// RegistryStats returns registry statistics.
type RegistryStats struct {
	TotalTools   int
	LocalTools   int
	MCPTools     int
	Backends     int
	IndexVersion uint64
}

// Stats returns registry statistics.
func (r *Registry) Stats() RegistryStats {
	r.mu.RLock()
	defer r.mu.RUnlock()

	tools, _ := r.index.Search("", 10000)
	localCount := 0
	mcpCount := 0

	for _, summary := range tools {
		_, backend, err := r.index.GetTool(summary.ID)
		if err != nil {
			continue
		}
		switch backend.Kind {
		case model.BackendKindLocal:
			localCount++
		case model.BackendKindMCP:
			mcpCount++
		}
	}

	return RegistryStats{
		TotalTools:   len(tools),
		LocalTools:   localCount,
		MCPTools:     mcpCount,
		Backends:     len(r.backends),
		IndexVersion: r.index.Version(),
	}
}

// HealthCheck returns nil if the registry is healthy.
func (r *Registry) HealthCheck(ctx context.Context) error {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if !r.started {
		return ErrNotStarted
	}

	for name, backend := range r.backends {
		if !backend.connected {
			return fmt.Errorf("backend %s not connected", name)
		}
	}

	return nil
}

// Refresh triggers a refresh of search indexes.
func (r *Registry) Refresh() uint64 {
	return r.index.Refresh()
}
