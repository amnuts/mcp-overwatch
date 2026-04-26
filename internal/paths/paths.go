package paths

import (
	"os"
	"path/filepath"
)

type Paths struct {
	base string
}

func New(base string) *Paths {
	return &Paths{base: base}
}

// DefaultBase returns {os.UserConfigDir()}/MCPOverwatch
func DefaultBase() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(configDir, "MCPOverwatch"), nil
}

func (p *Paths) Base() string             { return p.base }
func (p *Paths) ConfigFile() string       { return filepath.Join(p.base, "config.toml") }
func (p *Paths) Data() string             { return filepath.Join(p.base, "data") }
func (p *Paths) CatalogueDB() string      { return filepath.Join(p.base, "data", "catalogue.db") }
func (p *Paths) StatsDB() string          { return filepath.Join(p.base, "data", "stats.db") }
func (p *Paths) Runtimes() string         { return filepath.Join(p.base, "runtimes") }
func (p *Paths) RuntimeNode() string      { return filepath.Join(p.base, "runtimes", "node") }
func (p *Paths) RuntimePython() string    { return filepath.Join(p.base, "runtimes", "python") }
func (p *Paths) Packages() string         { return filepath.Join(p.base, "packages") }
func (p *Paths) PackagesNPM() string      { return filepath.Join(p.base, "packages", "npm") }
func (p *Paths) PackagesNPMCache() string { return filepath.Join(p.base, "packages", "npm", ".cache") }
func (p *Paths) PackagesPyPI() string     { return filepath.Join(p.base, "packages", "pypi") }
func (p *Paths) Repos() string            { return filepath.Join(p.base, "repos") }
func (p *Paths) Logs() string             { return filepath.Join(p.base, "logs") }
func (p *Paths) LogsServers() string      { return filepath.Join(p.base, "logs", "servers") }
func (p *Paths) ServerLogDir(id string) string {
	return filepath.Join(p.base, "logs", "servers", id)
}

func (p *Paths) EnsureAll() error {
	dirs := []string{
		p.Data(), p.Runtimes(), p.RuntimeNode(), p.RuntimePython(),
		p.Packages(), p.PackagesNPM(), p.PackagesNPMCache(), p.PackagesPyPI(),
		p.Repos(), p.Logs(), p.LogsServers(),
	}
	for _, d := range dirs {
		if err := os.MkdirAll(d, 0755); err != nil {
			return err
		}
	}
	return nil
}
