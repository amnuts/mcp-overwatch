package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadOrCreateDefault(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.toml")

	cfg, err := LoadOrCreate(configPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Proxy.Port != 3100 {
		t.Errorf("expected default port 3100, got %d", cfg.Proxy.Port)
	}
	if cfg.Sync.IntervalHours != 24 {
		t.Errorf("expected default sync interval 24h, got %d", cfg.Sync.IntervalHours)
	}
	// File should exist now
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Error("config file was not created")
	}
}

func TestLoadExisting(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.toml")

	content := []byte(`[proxy]
port = 4200

[sync]
interval_hours = 12
`)
	os.WriteFile(configPath, content, 0644)

	cfg, err := LoadOrCreate(configPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Proxy.Port != 4200 {
		t.Errorf("expected port 4200, got %d", cfg.Proxy.Port)
	}
	if cfg.Sync.IntervalHours != 12 {
		t.Errorf("expected sync interval 12, got %d", cfg.Sync.IntervalHours)
	}
}

func TestSave(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.toml")

	cfg := DefaultConfig()
	cfg.Proxy.Port = 5000
	err := Save(configPath, cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	loaded, err := LoadOrCreate(configPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if loaded.Proxy.Port != 5000 {
		t.Errorf("expected port 5000, got %d", loaded.Proxy.Port)
	}
}
