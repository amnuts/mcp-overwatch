package integration

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

func ClaudeCodeConfigPath() string {
	switch runtime.GOOS {
	case "windows":
		home, _ := os.UserHomeDir()
		return filepath.Join(home, ".claude.json")
	default: // darwin, linux
		home, _ := os.UserHomeDir()
		return filepath.Join(home, ".claude.json")
	}
}

func RegisterWithClaudeCode(configPath string, port int) error {
	var config map[string]interface{}

	data, err := os.ReadFile(configPath)
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	if len(data) > 0 {
		if err := json.Unmarshal(data, &config); err != nil {
			return fmt.Errorf("failed to parse existing config: %w", err)
		}
	} else {
		config = make(map[string]interface{})
	}

	servers, ok := config["mcpServers"].(map[string]interface{})
	if !ok {
		servers = make(map[string]interface{})
	}

	servers["mcp-overwatch"] = map[string]interface{}{
		"type": "http",
		"url":  fmt.Sprintf("http://localhost:%d/mcp", port),
	}

	config["mcpServers"] = servers

	out, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
		return err
	}

	return os.WriteFile(configPath, out, 0644)
}
