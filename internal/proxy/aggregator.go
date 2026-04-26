package proxy

import (
	"context"
	"fmt"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
)

func (p *Proxy) handleCallTool(ctx context.Context, namespacedName string, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	serverID, toolName, ok := p.routing.Resolve(namespacedName)
	if !ok {
		return nil, fmt.Errorf("unknown tool: %s", namespacedName)
	}

	start := time.Now()
	result, err := p.provider.CallTool(ctx, serverID, toolName, request.GetArguments())
	latencyMs := time.Since(start).Milliseconds()

	eventType := "tool_call"
	errMsg := ""
	if err != nil {
		errMsg = err.Error()
		eventType = "tool_error"
	} else if result != nil && result.IsError {
		eventType = "tool_error"
		// Extract error text from content if available
		for _, c := range result.Content {
			if tc, ok := c.(mcp.TextContent); ok {
				errMsg = tc.Text
				break
			}
		}
	}

	if p.onToolCall != nil {
		p.onToolCall(serverID, toolName, latencyMs, eventType, errMsg)
	}

	return result, err
}
