package config

import "testing"

func TestSecretKeyFormat(t *testing.T) {
	key := SecretKey("server-123", "API_KEY")
	expected := "mcp-overwatch:server-123:API_KEY"
	if key != expected {
		t.Errorf("expected %q, got %q", expected, key)
	}
}

func TestSecretKeyDifferentInputs(t *testing.T) {
	tests := []struct {
		serverID string
		envVar   string
		expected string
	}{
		{"srv-1", "TOKEN", "mcp-overwatch:srv-1:TOKEN"},
		{"my-server", "SECRET_KEY", "mcp-overwatch:my-server:SECRET_KEY"},
		{"", "KEY", "mcp-overwatch::KEY"},
	}

	for _, tt := range tests {
		got := SecretKey(tt.serverID, tt.envVar)
		if got != tt.expected {
			t.Errorf("SecretKey(%q, %q) = %q, want %q", tt.serverID, tt.envVar, got, tt.expected)
		}
	}
}
