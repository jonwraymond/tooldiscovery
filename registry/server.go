package registry

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
)

// ServeStdio runs the registry as an MCP server over stdio.
// Blocks until stdin is closed or context is cancelled.
func ServeStdio(ctx context.Context, r *Registry) error {
	scanner := bufio.NewScanner(os.Stdin)
	encoder := json.NewEncoder(os.Stdout)

	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		var req MCPRequest
		if err := json.Unmarshal(scanner.Bytes(), &req); err != nil {
			resp := MCPResponse{
				JSONRPC: "2.0",
				Error:   &MCPError{Code: ErrCodeParseError, Message: err.Error()},
			}
			if err := encoder.Encode(resp); err != nil {
				return fmt.Errorf("failed to encode error response: %w", err)
			}
			continue
		}

		resp := r.HandleRequest(ctx, req)
		if err := encoder.Encode(resp); err != nil {
			return fmt.Errorf("failed to encode response: %w", err)
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("scanner error: %w", err)
	}

	return nil
}

// ServeHTTP returns an http.Handler for streamable HTTP transport.
// Handles POST requests with JSON-RPC bodies, returns JSON responses.
func ServeHTTP(r *Registry) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var mcpReq MCPRequest
		if err := json.NewDecoder(req.Body).Decode(&mcpReq); err != nil {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(MCPResponse{
				JSONRPC: "2.0",
				Error:   &MCPError{Code: ErrCodeParseError, Message: err.Error()},
			})
			return
		}

		resp := r.HandleRequest(req.Context(), mcpReq)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	})
}

// ServeSSE returns an http.Handler for Server-Sent Events transport.
// Clients POST to establish connection, receive events via SSE stream.
func ServeSSE(r *Registry) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")

		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "SSE not supported", http.StatusInternalServerError)
			return
		}

		var mcpReq MCPRequest
		if err := json.NewDecoder(req.Body).Decode(&mcpReq); err != nil {
			writeSSEEvent(w, flusher, "error", MCPResponse{
				JSONRPC: "2.0",
				Error:   &MCPError{Code: ErrCodeParseError, Message: err.Error()},
			})
			return
		}

		resp := r.HandleRequest(req.Context(), mcpReq)
		writeSSEEvent(w, flusher, "message", resp)
	})
}

func writeSSEEvent(w http.ResponseWriter, f http.Flusher, event string, data any) {
	jsonData, _ := json.Marshal(data)
	if _, err := fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event, jsonData); err != nil {
		return
	}
	f.Flush()
}
