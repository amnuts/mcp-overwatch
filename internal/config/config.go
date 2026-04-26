package config

import (
	"os"
	"path/filepath"

	"github.com/pelletier/go-toml/v2"
)

type Config struct {
	Proxy   ProxyConfig   `toml:"proxy" json:"proxy"`
	Sync    SyncConfig    `toml:"sync" json:"sync"`
	App     AppConfig     `toml:"app" json:"app"`
	Logging LoggingConfig `toml:"logging" json:"logging"`
	Stats   StatsConfig   `toml:"stats" json:"stats"`
}

type ProxyConfig struct {
	Port            int    `toml:"port" json:"port"`
	NamespaceFormat string `toml:"namespace_format" json:"namespace_format"` // "{server}__{tool}" or "{tool}"
}

type SyncConfig struct {
	IntervalHours int    `toml:"interval_hours" json:"interval_hours"`
	LastSync      string `toml:"last_sync" json:"last_sync"` // RFC3339
}

type AppConfig struct {
	StartOnBoot    bool `toml:"start_on_boot" json:"start_on_boot"`
	MinimizeToTray bool `toml:"minimize_to_tray" json:"minimize_to_tray"`
}

type LoggingConfig struct {
	RetentionDays  int `toml:"retention_days" json:"retention_days"`
	RingBufferSize int `toml:"ring_buffer_size" json:"ring_buffer_size"`
}

type StatsConfig struct {
	RetentionDays int `toml:"retention_days" json:"retention_days"`
	FlushSeconds  int `toml:"flush_seconds" json:"flush_seconds"`
}

func DefaultConfig() *Config {
	return &Config{
		Proxy: ProxyConfig{
			Port:            3100,
			NamespaceFormat: "{server}__{tool}",
		},
		Sync: SyncConfig{
			IntervalHours: 24,
		},
		App: AppConfig{
			StartOnBoot:    false,
			MinimizeToTray: true,
		},
		Logging: LoggingConfig{
			RetentionDays:  7,
			RingBufferSize: 10000,
		},
		Stats: StatsConfig{
			RetentionDays: 90,
			FlushSeconds:  30,
		},
	}
}

func LoadOrCreate(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		cfg := DefaultConfig()
		if err := Save(path, cfg); err != nil {
			return nil, err
		}
		return cfg, nil
	}
	if err != nil {
		return nil, err
	}

	cfg := DefaultConfig() // start with defaults so missing fields get defaults
	if err := toml.Unmarshal(data, cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

func Save(path string, cfg *Config) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	data, err := toml.Marshal(cfg)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}
