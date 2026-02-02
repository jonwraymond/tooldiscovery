package registry

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/jonwraymond/toolfoundation/model"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// BackendConfig describes an MCP backend connection.
type BackendConfig struct {
	// Name is a unique identifier for the backend.
	Name string
	// URL is the MCP server URL (http(s)://, sse://, stdio://).
	URL string
	// Headers are optional HTTP headers for authenticated backends.
	Headers map[string]string
	// MaxRetries controls reconnect attempts for streamable HTTP transport.
	MaxRetries int
	// RetryInterval is reserved for future use.
	RetryInterval time.Duration
	// Transport overrides URL handling when provided (useful for tests).
	Transport mcp.Transport
}

type mcpBackend struct {
	config    BackendConfig
	client    *mcp.Client
	session   *mcp.ClientSession
	tools     []model.Tool
	mu        sync.RWMutex
	connected bool
}

// RegisterMCP registers an MCP server as a backend.
// Tools from this backend are discovered and registered on Start.
func (r *Registry) RegisterMCP(cfg BackendConfig) error {
	if strings.TrimSpace(cfg.Name) == "" {
		return fmt.Errorf("%w: backend name is required", ErrInvalidRequest)
	}

	r.mu.Lock()
	if _, exists := r.backends[cfg.Name]; exists {
		r.mu.Unlock()
		return fmt.Errorf("backend %s already registered", cfg.Name)
	}

	backend := &mcpBackend{config: cfg}
	r.backends[cfg.Name] = backend
	started := r.started
	r.mu.Unlock()

	if started {
		if err := backend.connect(context.Background()); err != nil {
			return fmt.Errorf("failed to connect backend %s: %w", cfg.Name, err)
		}
		if err := r.index.RegisterToolsFromMCP(cfg.Name, backend.toolsSnapshot()); err != nil {
			_ = backend.disconnect()
			return fmt.Errorf("failed to register backend %s tools: %w", cfg.Name, err)
		}
	}

	return nil
}

// UnregisterMCP removes a registered MCP backend.
func (r *Registry) UnregisterMCP(name string) error {
	r.mu.Lock()
	backend, exists := r.backends[name]
	if !exists {
		r.mu.Unlock()
		return fmt.Errorf("%w: %s", ErrBackendNotFound, name)
	}
	delete(r.backends, name)
	r.mu.Unlock()

	tools := backend.toolsSnapshot()
	for _, tool := range tools {
		_ = r.index.UnregisterBackend(tool.ToolID(), model.BackendKindMCP, name)
	}

	if backend.connected {
		if err := backend.disconnect(); err != nil {
			return err
		}
	}

	return nil
}

func (b *mcpBackend) connect(ctx context.Context) error {
	b.mu.Lock()
	if b.connected {
		b.mu.Unlock()
		return nil
	}
	b.mu.Unlock()

	transport, err := b.transport()
	if err != nil {
		return err
	}

	client := mcp.NewClient(&mcp.Implementation{Name: "tooldiscovery-registry"}, nil)
	session, err := client.Connect(ctx, transport, nil)
	if err != nil {
		return err
	}

	res, err := session.ListTools(ctx, nil)
	if err != nil {
		_ = session.Close()
		return err
	}

	tools := make([]model.Tool, 0, len(res.Tools))
	for _, tool := range res.Tools {
		if tool == nil {
			continue
		}
		tools = append(tools, model.Tool{Tool: *tool})
	}

	b.mu.Lock()
	b.client = client
	b.session = session
	b.tools = tools
	b.connected = true
	b.mu.Unlock()
	return nil
}

func (b *mcpBackend) disconnect() error {
	b.mu.Lock()
	if !b.connected {
		b.mu.Unlock()
		return nil
	}
	session := b.session
	b.client = nil
	b.session = nil
	b.connected = false
	b.mu.Unlock()

	if session != nil {
		return session.Close()
	}
	return nil
}

func (b *mcpBackend) callTool(ctx context.Context, name string, args map[string]any) (any, error) {
	b.mu.RLock()
	session := b.session
	connected := b.connected
	b.mu.RUnlock()

	if !connected || session == nil {
		return nil, fmt.Errorf("%w: backend not connected", ErrBackendNotFound)
	}

	result, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name:      name,
		Arguments: args,
	})
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrExecutionFailed, err)
	}
	if result == nil {
		return nil, nil
	}
	if result.IsError {
		return nil, fmt.Errorf("%w: %s", ErrExecutionFailed, toolResultError(result))
	}
	return toolResultValue(result), nil
}

func (b *mcpBackend) toolsSnapshot() []model.Tool {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if len(b.tools) == 0 {
		return nil
	}
	out := make([]model.Tool, len(b.tools))
	copy(out, b.tools)
	return out
}

func (b *mcpBackend) transport() (mcp.Transport, error) {
	if b.config.Transport != nil {
		return b.config.Transport, nil
	}
	if strings.TrimSpace(b.config.URL) == "" {
		return nil, errors.New("backend URL is required")
	}

	parsed, err := url.Parse(b.config.URL)
	if err != nil {
		return nil, fmt.Errorf("invalid backend URL: %w", err)
	}

	httpClient := httpClientWithHeaders(b.config.Headers)

	switch parsed.Scheme {
	case "http", "https":
		return &mcp.StreamableClientTransport{
			Endpoint:   b.config.URL,
			HTTPClient: httpClient,
			MaxRetries: b.config.MaxRetries,
		}, nil
	case "sse":
		parsed.Scheme = "http"
		return &mcp.SSEClientTransport{
			Endpoint:   parsed.String(),
			HTTPClient: httpClient,
		}, nil
	case "stdio":
		return &mcp.StdioTransport{}, nil
	default:
		return nil, fmt.Errorf("unsupported backend URL scheme %q", parsed.Scheme)
	}
}

func httpClientWithHeaders(headers map[string]string) *http.Client {
	if len(headers) == 0 {
		return nil
	}
	clone := make(map[string]string, len(headers))
	for k, v := range headers {
		if strings.TrimSpace(k) == "" {
			continue
		}
		clone[k] = v
	}
	if len(clone) == 0 {
		return nil
	}
	return &http.Client{
		Transport: &headerRoundTripper{
			base:    http.DefaultTransport,
			headers: clone,
		},
	}
}

type headerRoundTripper struct {
	base    http.RoundTripper
	headers map[string]string
}

func (h *headerRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	base := h.base
	if base == nil {
		base = http.DefaultTransport
	}
	for key, value := range h.headers {
		if req.Header.Get(key) == "" {
			req.Header.Set(key, value)
		}
	}
	return base.RoundTrip(req)
}

func toolResultValue(result *mcp.CallToolResult) any {
	if result == nil {
		return nil
	}
	if result.StructuredContent != nil {
		return result.StructuredContent
	}
	if len(result.Content) == 1 {
		if text, ok := result.Content[0].(*mcp.TextContent); ok {
			return text.Text
		}
	}
	return result.Content
}

func toolResultError(result *mcp.CallToolResult) string {
	if result == nil {
		return "tool execution failed"
	}
	for _, content := range result.Content {
		if text, ok := content.(*mcp.TextContent); ok && text.Text != "" {
			return text.Text
		}
	}
	if result.StructuredContent != nil {
		return fmt.Sprintf("%v", result.StructuredContent)
	}
	return "tool execution failed"
}
