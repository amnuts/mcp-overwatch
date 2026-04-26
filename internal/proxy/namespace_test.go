package proxy

import "testing"

func TestShortName(t *testing.T) {
	tests := []struct {
		displayName string
		expected    string
	}{
		{"Brave Search", "brave_search"},
		{"GitHub MCP Server", "github_mcp_server"},
		{"slack-notifications", "slack_notifications"},
		{"My.Server.Name", "my_server_name"},
	}
	for _, tt := range tests {
		got := ShortName(tt.displayName)
		if got != tt.expected {
			t.Errorf("ShortName(%q) = %q, want %q", tt.displayName, got, tt.expected)
		}
	}
}

func TestNamespacing(t *testing.T) {
	rt := NewRoutingTable()
	rt.AddServer("server-1", "Brave Search", []string{"web_search", "local_search"})
	rt.AddServer("server-2", "Slack", []string{"send_message", "read_channel"})

	serverID, toolName, ok := rt.Resolve("brave_search__web_search")
	if !ok || serverID != "server-1" || toolName != "web_search" {
		t.Errorf("unexpected resolve: %s, %s, %v", serverID, toolName, ok)
	}
}

func TestCollisionHandling(t *testing.T) {
	rt := NewRoutingTable()
	rt.AddServer("server-1", "Slack", []string{"send_message"})
	rt.AddServer("server-2", "Slack", []string{"send_message"})

	s1, _, ok := rt.Resolve("slack__send_message")
	if !ok || s1 != "server-1" {
		t.Errorf("first server should be slack__send_message")
	}

	s2, _, ok := rt.Resolve("slack_2__send_message")
	if !ok || s2 != "server-2" {
		t.Errorf("second server should be slack_2__send_message")
	}
}
