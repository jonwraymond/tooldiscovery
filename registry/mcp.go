package registry

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/jonwraymond/toolfoundation/model"
)

// MCPRequest represents an incoming MCP JSON-RPC request.
type MCPRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      any             `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// MCPResponse represents an MCP JSON-RPC response.
type MCPResponse struct {
	JSONRPC string    `json:"jsonrpc"`
	ID      any       `json:"id"`
	Result  any       `json:"result,omitempty"`
	Error   *MCPError `json:"error,omitempty"`
}

// MCPError is a JSON-RPC error object.
type MCPError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

// HandleRequest processes an MCP request and returns a response.
func (r *Registry) HandleRequest(ctx context.Context, req MCPRequest) MCPResponse {
	switch req.Method {
	case "initialize":
		return r.handleInitialize(ctx, req.ID, req.Params)
	case "tools/list":
		return r.handleToolsList(ctx, req.ID, req.Params)
	case "tools/call":
		return r.handleToolsCall(ctx, req.ID, req.Params)
	default:
		return MCPResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &MCPError{
				Code:    ErrCodeMethodNotFound,
				Message: fmt.Sprintf("method %s not found", req.Method),
			},
		}
	}
}

func (r *Registry) handleInitialize(ctx context.Context, id any, params json.RawMessage) MCPResponse {
	result := map[string]any{
		"protocolVersion": model.MCPVersion,
		"capabilities": map[string]any{
			"tools": map[string]any{},
		},
		"serverInfo": map[string]any{
			"name":    r.config.ServerInfo.Name,
			"version": r.config.ServerInfo.Version,
		},
	}

	return MCPResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result:  result,
	}
}

func (r *Registry) handleToolsList(ctx context.Context, id any, params json.RawMessage) MCPResponse {
	tools, err := r.ListAll(ctx)
	if err != nil {
		return MCPResponse{
			JSONRPC: "2.0",
			ID:      id,
			Error: &MCPError{
				Code:    ErrCodeInternal,
				Message: err.Error(),
			},
		}
	}

	mcpTools := make([]map[string]any, 0, len(tools))
	for _, tool := range tools {
		mcpTool := map[string]any{
			"name":        tool.Name,
			"description": tool.Description,
			"inputSchema": tool.InputSchema,
		}
		mcpTools = append(mcpTools, mcpTool)
	}

	result := map[string]any{
		"tools": mcpTools,
	}

	return MCPResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result:  result,
	}
}

type toolsCallParams struct {
	Name      string         `json:"name"`
	Arguments map[string]any `json:"arguments"`
}

func (r *Registry) handleToolsCall(ctx context.Context, id any, params json.RawMessage) MCPResponse {
	var callParams toolsCallParams
	if err := json.Unmarshal(params, &callParams); err != nil {
		return MCPResponse{
			JSONRPC: "2.0",
			ID:      id,
			Error: &MCPError{
				Code:    ErrCodeInvalidParams,
				Message: err.Error(),
			},
		}
	}

	result, err := r.Execute(ctx, callParams.Name, callParams.Arguments)
	if err != nil {
		code := ErrCodeToolExecFailed
		if errors.Is(err, ErrToolNotFound) {
			code = ErrCodeToolNotFound
		}
		return MCPResponse{
			JSONRPC: "2.0",
			ID:      id,
			Error: &MCPError{
				Code:    code,
				Message: err.Error(),
			},
		}
	}

	return MCPResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result:  result,
	}
}

func toMCPTool(tool mcp.Tool) map[string]any {
	return map[string]any{
		"name":        tool.Name,
		"description": tool.Description,
		"inputSchema": tool.InputSchema,
	}
}
