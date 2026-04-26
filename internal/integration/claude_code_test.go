package integration

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestRegisterWithClaudeCode_NewFile(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".claude.json")

	err := RegisterWithClaudeCode(configPath, 3100)
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
	if entry["type"] != "http" {
		t.Errorf("unexpected type: %v, want 'http'", entry["type"])
	}
}

func TestRegisterWithClaudeCode_PreservesExisting(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".claude.json")

	existing := `{"numStartups":5,"mcpServers":{"other-server":{"type":"stdio","command":"node","args":["server.js"]}}}`
	os.WriteFile(configPath, []byte(existing), 0644)

	err := RegisterWithClaudeCode(configPath, 3100)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, _ := os.ReadFile(configPath)
	var config map[string]interface{}
	json.Unmarshal(data, &config)

	// Verify existing top-level keys preserved
	if config["numStartups"].(float64) != 5 {
		t.Error("existing top-level key was overwritten")
	}

	servers := config["mcpServers"].(map[string]interface{})
	if _, ok := servers["other-server"]; !ok {
		t.Error("existing server entry was overwritten")
	}
	if _, ok := servers["mcp-overwatch"]; !ok {
		t.Error("mcp-overwatch entry not added")
	}

	entry := servers["mcp-overwatch"].(map[string]interface{})
	if entry["type"] != "http" {
		t.Errorf("unexpected type: %v, want 'http'", entry["type"])
	}
}
