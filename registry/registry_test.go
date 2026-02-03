package registry

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/jonwraymond/tooldiscovery/search"
	"github.com/jonwraymond/toolfoundation/model"
)

func TestNew(t *testing.T) {
	cfg := Config{
		ServerInfo: ServerInfo{
			Name:    "test-server",
			Version: "1.0.0",
		},
	}

	reg := New(cfg)

	if reg == nil {
		t.Fatal("expected non-nil registry")
	}
	if reg.config.ServerInfo.Name != "test-server" {
		t.Errorf("expected server name 'test-server', got %s", reg.config.ServerInfo.Name)
	}
}

func TestRegisterLocal(t *testing.T) {
	reg := New(Config{
		ServerInfo: ServerInfo{Name: "test", Version: "1.0.0"},
	})

	callCount := 0
	handler := func(ctx context.Context, args map[string]any) (any, error) {
		callCount++
		return map[string]any{"echo": args["message"]}, nil
	}

	err := reg.RegisterLocalFunc(
		"echo",
		"Echoes back input",
		map[string]any{
			"type": "object",
			"properties": map[string]any{
				"message": map[string]any{"type": "string"},
			},
		},
		handler,
		WithNamespace("test"),
		WithTags("echo", "utility"),
	)

	if err != nil {
		t.Fatalf("RegisterLocalFunc failed: %v", err)
	}

	ctx := context.Background()
	result, err := reg.Execute(ctx, "test:echo", map[string]any{"message": "hello"})
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if callCount != 1 {
		t.Errorf("expected handler to be called once, got %d", callCount)
	}

	resultMap, ok := result.(map[string]any)
	if !ok {
		t.Fatalf("expected result to be map[string]any, got %T", result)
	}

	if resultMap["echo"] != "hello" {
		t.Errorf("expected echo='hello', got %v", resultMap["echo"])
	}
}

func TestSearch(t *testing.T) {
	reg := New(Config{
		SearchConfig: &search.BM25Config{
			NameBoost: 3,
		},
		ServerInfo: ServerInfo{Name: "test", Version: "1.0.0"},
	})

	handler := func(ctx context.Context, args map[string]any) (any, error) {
		return nil, nil
	}

	_ = reg.RegisterLocalFunc("echo", "Echoes input", map[string]any{"type": "object"}, handler)
	_ = reg.RegisterLocalFunc("list", "Lists items", map[string]any{"type": "object"}, handler)

	ctx := context.Background()
	results, err := reg.Search(ctx, "echo", 10)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	if len(results) == 0 {
		t.Fatal("expected at least one result")
	}

	if results[0].Name != "echo" {
		t.Errorf("expected first result to be 'echo', got %s", results[0].Name)
	}
}

func TestHandleRequest_Initialize(t *testing.T) {
	reg := New(Config{
		ServerInfo: ServerInfo{
			Name:    "test-server",
			Version: "1.0.0",
		},
	})

	req := MCPRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "initialize",
	}

	resp := reg.HandleRequest(context.Background(), req)

	if resp.Error != nil {
		t.Fatalf("expected no error, got %v", resp.Error)
	}

	resultMap, ok := resp.Result.(map[string]any)
	if !ok {
		t.Fatalf("expected result to be map, got %T", resp.Result)
	}

	if resultMap["protocolVersion"] != model.MCPVersion {
		t.Errorf("expected protocolVersion %s, got %v", model.MCPVersion, resultMap["protocolVersion"])
	}

	serverInfo := resultMap["serverInfo"].(map[string]any)
	if serverInfo["name"] != "test-server" {
		t.Errorf("expected name 'test-server', got %v", serverInfo["name"])
	}
}

func TestHandleRequest_ToolsList(t *testing.T) {
	reg := New(Config{
		ServerInfo: ServerInfo{Name: "test", Version: "1.0.0"},
	})

	handler := func(ctx context.Context, args map[string]any) (any, error) {
		return nil, nil
	}

	_ = reg.RegisterLocalFunc("echo", "Echoes input", map[string]any{"type": "object"}, handler)

	req := MCPRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "tools/list",
	}

	resp := reg.HandleRequest(context.Background(), req)

	if resp.Error != nil {
		t.Fatalf("expected no error, got %v", resp.Error)
	}

	resultMap := resp.Result.(map[string]any)
	tools := resultMap["tools"].([]map[string]any)

	if len(tools) == 0 {
		t.Fatal("expected at least one tool")
	}

	if tools[0]["name"] != "echo" {
		t.Errorf("expected tool name 'echo', got %v", tools[0]["name"])
	}
}

func TestHandleRequest_ToolsCall(t *testing.T) {
	reg := New(Config{
		ServerInfo: ServerInfo{Name: "test", Version: "1.0.0"},
	})

	handler := func(ctx context.Context, args map[string]any) (any, error) {
		return map[string]any{"result": args["input"]}, nil
	}

	_ = reg.RegisterLocalFunc("process", "Processes input", map[string]any{"type": "object"}, handler)

	params, _ := json.Marshal(map[string]any{
		"name":      "process",
		"arguments": map[string]any{"input": "test"},
	})

	req := MCPRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "tools/call",
		Params:  params,
	}

	resp := reg.HandleRequest(context.Background(), req)

	if resp.Error != nil {
		t.Fatalf("expected no error, got %v", resp.Error)
	}

	resultMap := resp.Result.(map[string]any)
	if resultMap["result"] != "test" {
		t.Errorf("expected result='test', got %v", resultMap["result"])
	}
}

func TestHandleRequest_ToolsCall_NotFound(t *testing.T) {
	reg := New(Config{
		ServerInfo: ServerInfo{Name: "test", Version: "1.0.0"},
	})

	params, _ := json.Marshal(map[string]any{
		"name":      "missing",
		"arguments": map[string]any{},
	})

	req := MCPRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "tools/call",
		Params:  params,
	}

	resp := reg.HandleRequest(context.Background(), req)

	if resp.Error == nil {
		t.Fatal("expected error response")
	}
	if resp.Error.Code != ErrCodeToolNotFound {
		t.Errorf("expected ErrCodeToolNotFound, got %d", resp.Error.Code)
	}
}

func TestStats(t *testing.T) {
	reg := New(Config{
		ServerInfo: ServerInfo{Name: "test", Version: "1.0.0"},
	})

	handler := func(ctx context.Context, args map[string]any) (any, error) {
		return nil, nil
	}

	_ = reg.RegisterLocalFunc("tool1", "Tool 1", map[string]any{"type": "object"}, handler)
	_ = reg.RegisterLocalFunc("tool2", "Tool 2", map[string]any{"type": "object"}, handler)

	stats := reg.Stats()

	if stats.TotalTools != 2 {
		t.Errorf("expected 2 total tools, got %d", stats.TotalTools)
	}

	if stats.LocalTools != 2 {
		t.Errorf("expected 2 local tools, got %d", stats.LocalTools)
	}
}

func TestLifecycle(t *testing.T) {
	reg := New(Config{
		ServerInfo: ServerInfo{Name: "test", Version: "1.0.0"},
	})

	ctx := context.Background()

	if err := reg.Start(ctx); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	if err := reg.HealthCheck(ctx); err != nil {
		t.Fatalf("HealthCheck failed: %v", err)
	}

	if err := reg.Stop(); err != nil {
		t.Fatalf("Stop failed: %v", err)
	}
}

func TestRegisterMCPAndExecute(t *testing.T) {
	server := mcp.NewServer(&mcp.Implementation{Name: "backend-server"}, nil)
	type echoArgs struct {
		Message string `json:"message"`
	}

	mcp.AddTool(server, &mcp.Tool{
		Name:        "echo",
		Description: "Echo tool",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args echoArgs) (*mcp.CallToolResult, any, error) {
		return nil, map[string]any{"echo": args.Message}, nil
	})

	serverTransport, clientTransport := mcp.NewInMemoryTransports()
	ctx := context.Background()
	serverSession, err := server.Connect(ctx, serverTransport, nil)
	if err != nil {
		t.Fatalf("server connect failed: %v", err)
	}
	defer func() {
		if err := serverSession.Close(); err != nil {
			t.Fatalf("server session close failed: %v", err)
		}
	}()

	reg := New(Config{
		ServerInfo: ServerInfo{Name: "test", Version: "1.0.0"},
	})
	if err := reg.RegisterMCP(BackendConfig{
		Name:      "remote",
		Transport: clientTransport,
	}); err != nil {
		t.Fatalf("RegisterMCP failed: %v", err)
	}

	if err := reg.Start(ctx); err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	defer func() {
		if err := reg.Stop(); err != nil {
			t.Fatalf("Stop failed: %v", err)
		}
	}()

	result, err := reg.Execute(ctx, "echo", map[string]any{"message": "hi"})
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	resultMap, ok := result.(map[string]any)
	if !ok {
		t.Fatalf("expected map result, got %T", result)
	}
	if resultMap["echo"] != "hi" {
		t.Fatalf("expected echo='hi', got %v", resultMap["echo"])
	}
}

func TestServeHTTP(t *testing.T) {
	reg := New(Config{
		ServerInfo: ServerInfo{Name: "test", Version: "1.0.0"},
	})
	_ = reg.RegisterLocalFunc("echo", "Echo", map[string]any{"type": "object"}, func(ctx context.Context, args map[string]any) (any, error) {
		return args, nil
	})

	srv := httptest.NewServer(ServeHTTP(reg))
	defer srv.Close()

	body := bytes.NewBufferString(`{"jsonrpc":"2.0","id":1,"method":"tools/list"}`)
	resp, err := http.Post(srv.URL, "application/json", body)
	if err != nil {
		t.Fatalf("POST failed: %v", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	var mcpResp MCPResponse
	if err := json.NewDecoder(resp.Body).Decode(&mcpResp); err != nil {
		t.Fatalf("decode failed: %v", err)
	}
	if mcpResp.Error != nil {
		t.Fatalf("expected no error, got %v", mcpResp.Error)
	}
	resultMap, ok := mcpResp.Result.(map[string]any)
	if !ok {
		t.Fatalf("expected result map, got %T", mcpResp.Result)
	}
	tools, ok := resultMap["tools"].([]any)
	if !ok || len(tools) == 0 {
		t.Fatal("expected at least one tool")
	}
}

func TestServeSSE(t *testing.T) {
	reg := New(Config{
		ServerInfo: ServerInfo{Name: "test", Version: "1.0.0"},
	})
	_ = reg.RegisterLocalFunc("echo", "Echo", map[string]any{"type": "object"}, func(ctx context.Context, args map[string]any) (any, error) {
		return args, nil
	})

	srv := httptest.NewServer(ServeSSE(reg))
	defer srv.Close()

	reqBody := bytes.NewBufferString(`{"jsonrpc":"2.0","id":1,"method":"tools/list"}`)
	resp, err := http.Post(srv.URL, "application/json", reqBody)
	if err != nil {
		t.Fatalf("POST failed: %v", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	scanner := bufio.NewScanner(resp.Body)
	var dataLine string
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "data: ") {
			dataLine = strings.TrimPrefix(line, "data: ")
			break
		}
	}
	if err := scanner.Err(); err != nil {
		t.Fatalf("scanner failed: %v", err)
	}
	if dataLine == "" {
		t.Fatal("expected SSE data line")
	}

	var mcpResp MCPResponse
	if err := json.Unmarshal([]byte(dataLine), &mcpResp); err != nil {
		t.Fatalf("unmarshal SSE data failed: %v", err)
	}
	if mcpResp.Error != nil {
		t.Fatalf("expected no error, got %v", mcpResp.Error)
	}
	resultMap, ok := mcpResp.Result.(map[string]any)
	if !ok {
		t.Fatalf("expected result map, got %T", mcpResp.Result)
	}
	tools, ok := resultMap["tools"].([]any)
	if !ok || len(tools) == 0 {
		t.Fatal("expected at least one tool")
	}
}

func TestWithVersion(t *testing.T) {
	reg := New(Config{
		ServerInfo: ServerInfo{Name: "test", Version: "1.0.0"},
	})

	handler := func(ctx context.Context, args map[string]any) (any, error) {
		return nil, nil
	}

	err := reg.RegisterLocalFunc(
		"versioned",
		"Versioned tool",
		map[string]any{"type": "object"},
		handler,
		WithVersion("2.0.0"),
	)
	if err != nil {
		t.Fatalf("RegisterLocalFunc failed: %v", err)
	}

	ctx := context.Background()
	tool, err := reg.GetTool(ctx, "versioned")
	if err != nil {
		t.Fatalf("GetTool failed: %v", err)
	}
	if tool.Version != "2.0.0" {
		t.Errorf("expected version '2.0.0', got %s", tool.Version)
	}
}

func TestGetTool(t *testing.T) {
	reg := New(Config{
		ServerInfo: ServerInfo{Name: "test", Version: "1.0.0"},
	})

	handler := func(ctx context.Context, args map[string]any) (any, error) {
		return nil, nil
	}

	_ = reg.RegisterLocalFunc("mytool", "My tool", map[string]any{"type": "object"}, handler, WithNamespace("ns"))

	ctx := context.Background()
	tool, err := reg.GetTool(ctx, "ns:mytool")
	if err != nil {
		t.Fatalf("GetTool failed: %v", err)
	}
	if tool.Name != "mytool" {
		t.Errorf("expected name 'mytool', got %s", tool.Name)
	}

	_, err = reg.GetTool(ctx, "nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent tool")
	}
}

func TestListNamespaces(t *testing.T) {
	reg := New(Config{
		ServerInfo: ServerInfo{Name: "test", Version: "1.0.0"},
	})

	handler := func(ctx context.Context, args map[string]any) (any, error) {
		return nil, nil
	}

	_ = reg.RegisterLocalFunc("tool1", "Tool 1", map[string]any{"type": "object"}, handler, WithNamespace("ns1"))
	_ = reg.RegisterLocalFunc("tool2", "Tool 2", map[string]any{"type": "object"}, handler, WithNamespace("ns2"))

	ctx := context.Background()
	namespaces, err := reg.ListNamespaces(ctx)
	if err != nil {
		t.Fatalf("ListNamespaces failed: %v", err)
	}
	if len(namespaces) < 2 {
		t.Errorf("expected at least 2 namespaces, got %d", len(namespaces))
	}
}

func TestSearchSummaries(t *testing.T) {
	reg := New(Config{
		ServerInfo: ServerInfo{Name: "test", Version: "1.0.0"},
	})

	handler := func(ctx context.Context, args map[string]any) (any, error) {
		return nil, nil
	}

	_ = reg.RegisterLocalFunc("searchable", "A searchable tool", map[string]any{"type": "object"}, handler)

	ctx := context.Background()
	summaries, err := reg.SearchSummaries(ctx, "searchable", 10)
	if err != nil {
		t.Fatalf("SearchSummaries failed: %v", err)
	}
	if len(summaries) == 0 {
		t.Error("expected at least one summary")
	}
}

func TestRefresh(t *testing.T) {
	reg := New(Config{
		ServerInfo: ServerInfo{Name: "test", Version: "1.0.0"},
	})

	handler := func(ctx context.Context, args map[string]any) (any, error) {
		return nil, nil
	}

	_ = reg.RegisterLocalFunc("tool", "Tool", map[string]any{"type": "object"}, handler)

	version := reg.Refresh()
	if version == 0 {
		t.Error("expected non-zero version after refresh")
	}
}

func TestUnregisterMCP(t *testing.T) {
	server := mcp.NewServer(&mcp.Implementation{Name: "backend-server"}, nil)
	mcp.AddTool(server, &mcp.Tool{
		Name:        "tool1",
		Description: "Tool 1",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args struct{}) (*mcp.CallToolResult, any, error) {
		return nil, nil, nil
	})

	serverTransport, clientTransport := mcp.NewInMemoryTransports()
	ctx := context.Background()
	serverSession, err := server.Connect(ctx, serverTransport, nil)
	if err != nil {
		t.Fatalf("server connect failed: %v", err)
	}
	defer func() {
		_ = serverSession.Close()
	}()

	reg := New(Config{
		ServerInfo: ServerInfo{Name: "test", Version: "1.0.0"},
	})

	if err := reg.RegisterMCP(BackendConfig{
		Name:      "backend1",
		Transport: clientTransport,
	}); err != nil {
		t.Fatalf("RegisterMCP failed: %v", err)
	}

	if err := reg.Start(ctx); err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	defer func() {
		_ = reg.Stop()
	}()

	// Verify tool exists
	_, err = reg.GetTool(ctx, "tool1")
	if err != nil {
		t.Fatalf("expected tool1 to exist: %v", err)
	}

	// Unregister
	if err := reg.UnregisterMCP("backend1"); err != nil {
		t.Fatalf("UnregisterMCP failed: %v", err)
	}

	// Verify backend is gone
	err = reg.UnregisterMCP("backend1")
	if err == nil {
		t.Error("expected error when unregistering non-existent backend")
	}
}

func TestRegisterMCPEmptyName(t *testing.T) {
	reg := New(Config{
		ServerInfo: ServerInfo{Name: "test", Version: "1.0.0"},
	})

	err := reg.RegisterMCP(BackendConfig{
		Name: "",
		URL:  "http://example.com",
	})
	if err == nil {
		t.Error("expected error for empty backend name")
	}
}

func TestRegisterMCPDuplicate(t *testing.T) {
	server := mcp.NewServer(&mcp.Implementation{Name: "backend-server"}, nil)
	serverTransport, clientTransport := mcp.NewInMemoryTransports()
	ctx := context.Background()
	serverSession, _ := server.Connect(ctx, serverTransport, nil)
	defer func() {
		_ = serverSession.Close()
	}()

	reg := New(Config{
		ServerInfo: ServerInfo{Name: "test", Version: "1.0.0"},
	})

	_ = reg.RegisterMCP(BackendConfig{
		Name:      "backend",
		Transport: clientTransport,
	})

	_, clientTransport2 := mcp.NewInMemoryTransports()
	err := reg.RegisterMCP(BackendConfig{
		Name:      "backend",
		Transport: clientTransport2,
	})
	if err == nil {
		t.Error("expected error for duplicate backend name")
	}
}

func TestHealthCheckNotStarted(t *testing.T) {
	reg := New(Config{
		ServerInfo: ServerInfo{Name: "test", Version: "1.0.0"},
	})

	ctx := context.Background()
	err := reg.HealthCheck(ctx)
	if err != ErrNotStarted {
		t.Errorf("expected ErrNotStarted, got %v", err)
	}
}

func TestStartAlreadyStarted(t *testing.T) {
	reg := New(Config{
		ServerInfo: ServerInfo{Name: "test", Version: "1.0.0"},
	})

	ctx := context.Background()
	_ = reg.Start(ctx)
	defer func() {
		_ = reg.Stop()
	}()

	err := reg.Start(ctx)
	if err != ErrAlreadyStarted {
		t.Errorf("expected ErrAlreadyStarted, got %v", err)
	}
}

func TestHandleRequest_MethodNotFound(t *testing.T) {
	reg := New(Config{
		ServerInfo: ServerInfo{Name: "test", Version: "1.0.0"},
	})

	req := MCPRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "unknown/method",
	}

	resp := reg.HandleRequest(context.Background(), req)
	if resp.Error == nil {
		t.Fatal("expected error for unknown method")
	}
	if resp.Error.Code != ErrCodeMethodNotFound {
		t.Errorf("expected ErrCodeMethodNotFound, got %d", resp.Error.Code)
	}
}

func TestServeHTTP_MethodNotAllowed(t *testing.T) {
	reg := New(Config{
		ServerInfo: ServerInfo{Name: "test", Version: "1.0.0"},
	})

	srv := httptest.NewServer(ServeHTTP(reg))
	defer srv.Close()

	resp, err := http.Get(srv.URL)
	if err != nil {
		t.Fatalf("GET failed: %v", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", resp.StatusCode)
	}
}

func TestServeHTTP_InvalidJSON(t *testing.T) {
	reg := New(Config{
		ServerInfo: ServerInfo{Name: "test", Version: "1.0.0"},
	})

	srv := httptest.NewServer(ServeHTTP(reg))
	defer srv.Close()

	body := bytes.NewBufferString(`{invalid json`)
	resp, err := http.Post(srv.URL, "application/json", body)
	if err != nil {
		t.Fatalf("POST failed: %v", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	var mcpResp MCPResponse
	_ = json.NewDecoder(resp.Body).Decode(&mcpResp)
	if mcpResp.Error == nil {
		t.Fatal("expected error for invalid JSON")
	}
	if mcpResp.Error.Code != ErrCodeParseError {
		t.Errorf("expected ErrCodeParseError, got %d", mcpResp.Error.Code)
	}
}
