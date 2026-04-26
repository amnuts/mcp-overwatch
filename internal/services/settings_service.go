package services

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"mcp-overwatch/internal/catalogue"
	"mcp-overwatch/internal/config"
	"mcp-overwatch/internal/integration"
	"mcp-overwatch/internal/logging"
	"mcp-overwatch/internal/paths"
	"mcp-overwatch/internal/runtime"
	"mcp-overwatch/internal/version"
)

// AppInfo is returned to the frontend for the About view.
type AppInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// SettingsService exposes app settings and integration management to the frontend.
type SettingsService struct {
	cfg      *config.Config
	cfgPath  string
	paths    *paths.Paths
	runtimes *runtime.Manager
	store    *catalogue.Store
	logger   *logging.Logger
}

// NewSettingsService creates a SettingsService with the given dependencies.
func NewSettingsService(cfg *config.Config, cfgPath string, p *paths.Paths, runtimes *runtime.Manager, store *catalogue.Store, logger *logging.Logger) *SettingsService {
	return &SettingsService{cfg: cfg, cfgPath: cfgPath, paths: p, runtimes: runtimes, store: store, logger: logger}
}

// log writes a system-scoped log entry, if a logger is configured.
func (s *SettingsService) log(direction, summary string) {
	if s.logger == nil {
		return
	}
	s.logger.Add(logging.Entry{
		ServerID:  "system",
		Direction: direction,
		Summary:   summary,
	})
}

// GetSettings returns the current application configuration.
func (s *SettingsService) GetSettings() *config.Config {
	return s.cfg
}

// GetAppInfo returns app metadata (name and build version) for display in the UI.
func (s *SettingsService) GetAppInfo() AppInfo {
	return AppInfo{Name: "MCP Overwatch", Version: version.Version}
}

// SaveSettings persists updated configuration to disk.
func (s *SettingsService) SaveSettings(cfg config.Config) error {
	*s.cfg = cfg
	if err := config.Save(s.cfgPath, s.cfg); err != nil {
		s.log("in", fmt.Sprintf("Settings save failed: %v", err))
		return err
	}
	s.log("in", "Settings saved")
	return nil
}

// RegisterClaudeDesktop registers the MCP Overwatch proxy with Claude Desktop.
func (s *SettingsService) RegisterClaudeDesktop() error {
	configPath := integration.ClaudeDesktopConfigPath()
	if err := integration.RegisterWithClaudeDesktop(configPath, s.cfg.Proxy.Port); err != nil {
		s.log("in", fmt.Sprintf("Claude Desktop registration failed: %v", err))
		return err
	}
	s.log("in", fmt.Sprintf("Registered MCP Overwatch with Claude Desktop on port %d", s.cfg.Proxy.Port))
	return nil
}

// RegisterClaudeCode registers the MCP Overwatch proxy with Claude Code.
func (s *SettingsService) RegisterClaudeCode() error {
	configPath := integration.ClaudeCodeConfigPath()
	if err := integration.RegisterWithClaudeCode(configPath, s.cfg.Proxy.Port); err != nil {
		s.log("in", fmt.Sprintf("Claude Code registration failed: %v", err))
		return err
	}
	s.log("in", fmt.Sprintf("Registered MCP Overwatch with Claude Code on port %d", s.cfg.Proxy.Port))
	return nil
}

// DetectClients returns a list of detected MCP client applications.
func (s *SettingsService) DetectClients() []integration.ClientInfo {
	return integration.DetectClients()
}

// GetRuntimes returns all tracked runtime installations.
func (s *SettingsService) GetRuntimes() ([]catalogue.Runtime, error) {
	return s.store.ListRuntimes()
}

// CheckDocker returns true if docker is available on PATH and the daemon is reachable.
func (s *SettingsService) CheckDocker() bool {
	cmd := exec.Command("docker", "info")
	return cmd.Run() == nil
}

// DownloadRuntime downloads and installs a portable runtime (e.g. "node", "python").
func (s *SettingsService) DownloadRuntime(runtimeID string) error {
	s.log("out", fmt.Sprintf("Downloading runtime %q", runtimeID))
	provider, err := s.runtimes.Get(runtimeID)
	if err != nil {
		s.log("in", fmt.Sprintf("Runtime %q download failed: %v", runtimeID, err))
		return err
	}
	runtimeDir := s.runtimes.RuntimeDir(runtimeID)
	if err := provider.Download(runtimeDir, nil); err != nil {
		s.log("in", fmt.Sprintf("Runtime %q download failed: %v", runtimeID, err))
		return err
	}

	version, _ := provider.Verify(runtimeDir)
	size := dirSize(runtimeDir)

	if err := s.store.UpsertRuntime(catalogue.Runtime{
		ID:          runtimeID,
		Version:     version,
		Path:        runtimeDir,
		SizeBytes:   size,
		InstalledAt: time.Now(),
		Status:      "ready",
	}); err != nil {
		s.log("in", fmt.Sprintf("Runtime %q install record failed: %v", runtimeID, err))
		return err
	}
	s.log("in", fmt.Sprintf("Installed runtime %q v%s", runtimeID, version))
	return nil
}

// DeleteRuntime removes a downloaded runtime from disk and the database.
func (s *SettingsService) DeleteRuntime(runtimeID string) error {
	runtimeDir := s.runtimes.RuntimeDir(runtimeID)
	if err := os.RemoveAll(runtimeDir); err != nil {
		s.log("in", fmt.Sprintf("Runtime %q delete failed: %v", runtimeID, err))
		return fmt.Errorf("removing runtime directory: %w", err)
	}
	if err := s.store.DeleteRuntime(runtimeID); err != nil {
		s.log("in", fmt.Sprintf("Runtime %q delete failed (db): %v", runtimeID, err))
		return err
	}
	s.log("in", fmt.Sprintf("Deleted runtime %q", runtimeID))
	return nil
}

// dirSize returns the total size of all files in a directory tree.
func dirSize(path string) int64 {
	var size int64
	_ = filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		size += info.Size()
		return nil
	})
	return size
}
