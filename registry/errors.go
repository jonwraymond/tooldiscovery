package registry

import "errors"

// Sentinel errors for consistent error handling.
var (
	ErrNotStarted      = errors.New("registry not started")
	ErrAlreadyStarted  = errors.New("registry already started")
	ErrToolNotFound    = errors.New("tool not found")
	ErrBackendNotFound = errors.New("backend not found")
	ErrHandlerNotFound = errors.New("handler not found")
	ErrExecutionFailed = errors.New("tool execution failed")
	ErrInvalidRequest  = errors.New("invalid request")
)

// MCP JSON-RPC 2.0 error codes as per the spec.
const (
	ErrCodeParseError     = -32700
	ErrCodeInvalidRequest = -32600
	ErrCodeMethodNotFound = -32601
	ErrCodeInvalidParams  = -32602
	ErrCodeInternal       = -32603
	ErrCodeToolNotFound   = -32001
	ErrCodeToolExecFailed = -32002
)
