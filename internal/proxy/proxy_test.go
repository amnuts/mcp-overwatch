package proxy

import (
	"context"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
)

type mockProvider struct {
	servers map[string]ServerInfo
	tools   map[string][]mcp.Tool
	calls   []mockCall
}

type mockCall struct {
	ServerID string
	ToolName string
	Args     map[string]interface{}
}

func newMockProvider() *mockProvider {
	return &mockProvider{
		servers: make(map[string]ServerInfo),
		tools:   make(map[string][]mcp.Tool),
	}
}

func (m *mockProvider) addServer(id, displayName string, toolNames []string) {
	m.servers[id] = ServerInfo{ID: id, DisplayName: displayName}
	tools := make([]mcp.Tool, len(toolNames))
	for i, name := range toolNames {
		tools[i] = mcp.Tool{
			Name:        name,
			Description: "Test tool " + name,
		}
	}
	m.tools[id] = tools
}

func (m *mockProvider) CallTool(_ context.Context, serverID, toolName string, args map[string]interface{}) (*mcp.CallToolResult, error) {
	m.calls = append(m.calls, mockCall{ServerID: serverID, ToolName: toolName, Args: args})
	return mcp.NewToolResultText("ok"), nil
}

func (m *mockProvider) ListToolsForServer(serverID string) []mcp.Tool {
	return m.tools[serverID]
}

func (m *mockProvider) ActiveServers() []ServerInfo {
	result := make([]ServerInfo, 0, len(m.servers))
	for _, s := range m.servers {
		result = append(result, s)
	}
	return result
}

func TestProxyAggregation(t *testing.T) {
	mp := newMockProvider()
	mp.addServer("s1", "Brave Search", []string{"web_search", "local_search"})
	mp.addServer("s2", "Slack", []string{"send_message"})

	p := NewProxy(0, mp)
	p.RebuildRoutes()

	// Check that routing table has all namespaced tools
	allTools := p.Routing().AllTools()

	if _, ok := allTools["brave_search__web_search"]; !ok {
		t.Error("expected brave_search__web_search in routing table")
	}
	if _, ok := allTools["brave_search__local_search"]; !ok {
		t.Error("expected brave_search__local_search in routing table")
	}
	if _, ok := allTools["slack__send_message"]; !ok {
		t.Error("expected slack__send_message in routing table")
	}
}

func TestProxyRouting(t *testing.T) {
	mp := newMockProvider()
	mp.addServer("s1", "Brave Search", []string{"web_search"})

	p := NewProxy(0, mp)
	p.RebuildRoutes()

	serverID, toolName, ok := p.Routing().Resolve("brave_search__web_search")
	if !ok {
		t.Fatal("expected resolution to succeed")
	}
	if serverID != "s1" {
		t.Errorf("expected serverID=s1, got %s", serverID)
	}
	if toolName != "web_search" {
		t.Errorf("expected toolName=web_search, got %s", toolName)
	}
}

func TestProxyRebuildRoutes(t *testing.T) {
	mp := newMockProvider()
	mp.addServer("s1", "Brave Search", []string{"web_search"})

	p := NewProxy(0, mp)
	p.RebuildRoutes()

	// Verify initial state
	allTools := p.Routing().AllTools()
	if len(allTools) != 1 {
		t.Fatalf("expected 1 tool, got %d", len(allTools))
	}

	// Add a new server and rebuild
	mp.addServer("s2", "Slack", []string{"send_message"})
	p.RebuildRoutes()

	allTools = p.Routing().AllTools()
	if len(allTools) != 2 {
		t.Fatalf("expected 2 tools after rebuild, got %d", len(allTools))
	}
}

func TestProxyToolCallCallback(t *testing.T) {
	mp := newMockProvider()
	mp.addServer("s1", "Test", []string{"do_thing"})

	p := NewProxy(0, mp)
	p.RebuildRoutes()

	var cbServerID, cbToolName, cbEventType string
	p.SetToolCallCallback(func(serverID, toolName string, latencyMs int64, eventType, errMsg string) {
		cbServerID = serverID
		cbToolName = toolName
		cbEventType = eventType
	})

	req := mcp.CallToolRequest{}
	_, err := p.handleCallTool(context.Background(), "test__do_thing", req)
	if err != nil {
		t.Fatal(err)
	}

	if cbServerID != "s1" || cbToolName != "do_thing" || cbEventType != "tool_call" {
		t.Errorf("callback got serverID=%s toolName=%s eventType=%s", cbServerID, cbToolName, cbEventType)
	}
}
