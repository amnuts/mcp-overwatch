package proxy

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// ServerInfo describes a managed MCP server.
type ServerInfo struct {
	ID          string
	DisplayName string
}

// ServerProvider is an interface for accessing managed MCP servers.
type ServerProvider interface {
	CallTool(ctx context.Context, serverID, toolName string, args map[string]interface{}) (*mcp.CallToolResult, error)
	ListToolsForServer(serverID string) []mcp.Tool
	ActiveServers() []ServerInfo
}

// ToolCallCallback is called after every tool call with timing and error info.
type ToolCallCallback func(serverID, toolName string, latencyMs int64, eventType, errMsg string)

// LogCallback is invoked for proxy lifecycle events (start, stop, route rebuild).
type LogCallback func(serverID, stream, message string)

// Proxy is a unified MCP proxy that aggregates tools from multiple servers.
type Proxy struct {
	mcpServer  *server.MCPServer
	httpServer *http.Server
	routing    *RoutingTable
	provider   ServerProvider
	port       int
	onToolCall ToolCallCallback
	onLog      LogCallback
}

// NewProxy creates a new unified MCP proxy.
func NewProxy(port int, provider ServerProvider) *Proxy {
	p := &Proxy{
		routing:  NewRoutingTable(),
		provider: provider,
		port:     port,
	}

	mcpSrv := server.NewMCPServer(
		"mcp-overwatch",
		"1.0.0",
		server.WithToolCapabilities(true),
	)

	p.mcpServer = mcpSrv
	return p
}

// SetToolCallCallback sets a callback invoked after each tool call.
func (p *Proxy) SetToolCallCallback(cb ToolCallCallback) {
	p.onToolCall = cb
}

// SetLogCallback sets a callback invoked for proxy lifecycle events.
func (p *Proxy) SetLogCallback(cb LogCallback) {
	p.onLog = cb
}

// log emits a proxy lifecycle log entry, if a callback is configured.
func (p *Proxy) log(stream, message string) {
	if p.onLog != nil {
		p.onLog("proxy", stream, message)
	}
}

// Start starts the HTTP server for the proxy.
func (p *Proxy) Start() error {
	p.RebuildRoutes()

	streamServer := server.NewStreamableHTTPServer(p.mcpServer)

	p.httpServer = &http.Server{
		Addr:    fmt.Sprintf("127.0.0.1:%d", p.port),
		Handler: streamServer,
	}

	p.log("lifecycle", fmt.Sprintf("listening on %s", p.httpServer.Addr))
	err := p.httpServer.ListenAndServe()
	if err != nil && err != http.ErrServerClosed {
		p.log("lifecycle", fmt.Sprintf("listener exited with error: %v", err))
	}
	return err
}

// Stop gracefully shuts down the HTTP server.
func (p *Proxy) Stop() error {
	if p.httpServer == nil {
		return nil
	}
	p.log("lifecycle", "shutting down")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := p.httpServer.Shutdown(ctx); err != nil {
		p.log("lifecycle", fmt.Sprintf("shutdown error: %v", err))
		return err
	}
	return nil
}

// RebuildRoutes rebuilds the routing table and re-registers tools from all active servers.
func (p *Proxy) RebuildRoutes() {
	newRT := NewRoutingTable()

	servers := p.provider.ActiveServers()
	var allTools []server.ServerTool

	for _, srv := range servers {
		tools := p.provider.ListToolsForServer(srv.ID)
		toolNames := make([]string, len(tools))
		for i, t := range tools {
			toolNames[i] = t.Name
		}
		newRT.AddServer(srv.ID, srv.DisplayName, toolNames)

		// Create namespaced ServerTool entries
		for namespacedName, entry := range newRT.AllEntries() {
			if entry.ServerID != srv.ID {
				continue
			}
			// Find the original tool definition
			for _, t := range tools {
				if t.Name == entry.ToolName {
					namespacedTool := t
					namespacedTool.Name = namespacedName
					if namespacedTool.Description != "" {
						namespacedTool.Description = fmt.Sprintf("[%s] %s", srv.DisplayName, namespacedTool.Description)
					} else {
						namespacedTool.Description = fmt.Sprintf("[%s] %s", srv.DisplayName, entry.ToolName)
					}

					capturedName := namespacedName
					allTools = append(allTools, server.ServerTool{
						Tool: namespacedTool,
						Handler: func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
							return p.handleCallTool(ctx, capturedName, request)
						},
					})
					break
				}
			}
		}
	}

	p.routing = newRT
	p.mcpServer.SetTools(allTools...)
	p.log("lifecycle", fmt.Sprintf("routes rebuilt — %d active server(s), %d tool(s)", len(servers), len(allTools)))
}

// Routing returns the current routing table (for testing/inspection).
func (p *Proxy) Routing() *RoutingTable {
	return p.routing
}
