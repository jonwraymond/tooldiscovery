// Package registry provides high-level helpers for building MCP servers
// with tool discovery, registration, and execution.
//
// Registry combines toolfoundation/model, tooldiscovery/index, and
// tooldiscovery/search into a unified API for creating MCP servers quickly.
//
// Features:
//   - Local tool registration with handlers
//   - MCP backend connections (streamable HTTP, SSE, stdio)
//   - BM25-based tool search
//   - MCP protocol handlers (initialize, tools/list, tools/call)
//   - Multiple transports (stdio, HTTP, SSE)
//
// Example usage:
//
//	reg := registry.New(registry.Config{
//	    ServerInfo: registry.ServerInfo{
//	        Name:    "my-server",
//	        Version: "1.0.0",
//	    },
//	})
//
//	reg.RegisterLocalFunc(
//	    "echo",
//	    "Echoes back the input",
//	    map[string]any{
//	        "type": "object",
//	        "properties": map[string]any{
//	            "message": map[string]any{"type": "string"},
//	        },
//	    },
//	    func(ctx context.Context, args map[string]any) (any, error) {
//	        return args, nil
//	    },
//	)
//
//	ctx := context.Background()
//	reg.Start(ctx)
//	defer reg.Stop()
//
//	registry.ServeStdio(ctx, reg)
package registry
