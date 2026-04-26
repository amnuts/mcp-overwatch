package integration

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestRegisterWithClaudeDesktop_NewFile(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "claude_desktop_config.json")

	err := RegisterWithClaudeDesktop(configPath, 3100)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, _ := os.ReadFile(configPath)
	var config map[string]interface{}
	json.Unmarshal(data, &config)

	servers := config["mcpServers"].(map[string]interface{})
	entry := servers["mcp-overwatch"].(map[string]interface{})
	if entry["url"] != "http://localhost:3100/mcp" {
		t.Errorf("unexpected url: %v", entry["url"])
	}
	if entry["type"] != "streamable-http" {
		t.Errorf("unexpected type: %v", entry["type"])
	}
}

func TestRegisterWithClaudeDesktop_PreservesExisting(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "claude_desktop_config.json")

	existing := `{"mcpServers":{"other-server":{"command":"node","args":["server.js"]}}}`
	os.WriteFile(configPath, []byte(existing), 0644)

	err := RegisterWithClaudeDesktop(configPath, 3100)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, _ := os.ReadFile(configPath)
	var config map[string]interface{}
	json.Unmarshal(data, &config)

	servers := config["mcpServers"].(map[string]interface{})
	if _, ok := servers["other-server"]; !ok {
		t.Error("existing server entry was overwritten")
	}
	if _, ok := servers["mcp-overwatch"]; !ok {
		t.Error("mcp-overwatch entry not added")
	}
}
