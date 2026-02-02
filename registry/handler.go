package registry

import (
	"context"

	"github.com/jonwraymond/toolfoundation/model"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// ToolHandler executes a local tool with the given arguments.
// It receives a context for cancellation and a map of arguments parsed from the MCP request.
// It returns the result as any (typically a map or struct) and an error if execution fails.
type ToolHandler func(ctx context.Context, args map[string]any) (any, error)

// LocalToolOption configures local tool registration.
type LocalToolOption func(*localToolConfig)

type localToolConfig struct {
	namespace string
	tags      []string
	version   string
}

// WithNamespace sets the namespace for a local tool.
func WithNamespace(ns string) LocalToolOption {
	return func(c *localToolConfig) {
		c.namespace = ns
	}
}

// WithTags sets the tags for a local tool.
func WithTags(tags ...string) LocalToolOption {
	return func(c *localToolConfig) {
		c.tags = tags
	}
}

// WithVersion sets the version for a local tool.
func WithVersion(v string) LocalToolOption {
	return func(c *localToolConfig) {
		c.version = v
	}
}

func applyLocalToolOptions(opts []LocalToolOption) localToolConfig {
	cfg := localToolConfig{}
	for _, opt := range opts {
		opt(&cfg)
	}
	return cfg
}

func buildLocalTool(name, description string, inputSchema map[string]any, cfg localToolConfig) model.Tool {
	tool := model.Tool{
		Tool: mcp.Tool{
			Name:        name,
			Description: description,
			InputSchema: inputSchema,
		},
		Namespace: cfg.namespace,
		Version:   cfg.version,
		Tags:      model.NormalizeTags(cfg.tags),
	}
	return tool
}
