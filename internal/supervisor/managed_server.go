package supervisor

import (
	"sync"
	"time"

	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
)

// ServerStatus represents the current state of a managed MCP server.
type ServerStatus string

const (
	StatusStopped  ServerStatus = "stopped"
	StatusStarting ServerStatus = "starting"
	StatusRunning  ServerStatus = "running"
	StatusError    ServerStatus = "error"
)

// ManagedServer tracks a single MCP server connection and its cached capabilities.
// Supports multiple transport types: stdio subprocesses and remote HTTP/SSE endpoints.
type ManagedServer struct {
	ID          string
	DisplayName string

	// TransportType selects how to connect. Empty is treated as "stdio" for
	// backward compatibility. Recognised values: "stdio", "sse",
	// "streamable-http" (aliases: "http", "streaming-http").
	TransportType string

	// Stdio transport fields.
	Command string
	Args    []string
	Env     []string
	WorkDir string

	// Remote transport fields (sse / streamable-http).
	RemoteURL string
	Headers   map[string]string

	status     ServerStatus
	client     *client.Client
	tools      []mcp.Tool
	resources  []mcp.Resource
	prompts    []mcp.Prompt
	lastUsed   time.Time
	errorCount int
	stopCh     chan struct{}
	mu         sync.Mutex
}

// Status returns the current server status in a goroutine-safe manner.
func (ms *ManagedServer) Status() ServerStatus {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	return ms.status
}

// Client returns the underlying MCP client, or nil if not running.
func (ms *ManagedServer) Client() *client.Client {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	return ms.client
}

// Tools returns the cached list of tools from the MCP server.
func (ms *ManagedServer) Tools() []mcp.Tool {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	result := make([]mcp.Tool, len(ms.tools))
	copy(result, ms.tools)
	return result
}

// Resources returns the cached list of resources from the MCP server.
func (ms *ManagedServer) Resources() []mcp.Resource {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	result := make([]mcp.Resource, len(ms.resources))
	copy(result, ms.resources)
	return result
}

// Prompts returns the cached list of prompts from the MCP server.
func (ms *ManagedServer) Prompts() []mcp.Prompt {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	result := make([]mcp.Prompt, len(ms.prompts))
	copy(result, ms.prompts)
	return result
}

// LastUsed returns the timestamp of the last interaction with the server.
func (ms *ManagedServer) LastUsed() time.Time {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	return ms.lastUsed
}

// ErrorCount returns the number of consecutive errors encountered.
func (ms *ManagedServer) ErrorCount() int {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	return ms.errorCount
}
