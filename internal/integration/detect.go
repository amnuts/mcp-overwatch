package integration

import (
	"os"
	"path/filepath"
)

type ClientInfo struct {
	Name       string `json:"name"`
	Installed  bool   `json:"installed"`
	ConfigPath string `json:"configPath"`
}

func DetectClients() []ClientInfo {
	var clients []ClientInfo

	cdPath := ClaudeDesktopConfigPath()
	_, err := os.Stat(filepath.Dir(cdPath))
	clients = append(clients, ClientInfo{
		Name:       "Claude Desktop",
		Installed:  err == nil,
		ConfigPath: cdPath,
	})

	ccPath := ClaudeCodeConfigPath()
	_, err = os.Stat(ccPath)
	clients = append(clients, ClientInfo{
		Name:       "Claude Code",
		Installed:  err == nil,
		ConfigPath: ccPath,
	})

	return clients
}
