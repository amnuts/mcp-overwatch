package paths

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEnsureDirectories(t *testing.T) {
	tmpDir := t.TempDir()
	p := New(tmpDir)

	if err := p.EnsureAll(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	dirs := []string{
		p.Runtimes(), p.RuntimeNode(), p.RuntimePython(),
		p.Packages(), p.PackagesNPM(), p.PackagesNPMCache(), p.PackagesPyPI(),
		p.Data(), p.Logs(), p.LogsServers(),
	}
	for _, d := range dirs {
		if _, err := os.Stat(d); os.IsNotExist(err) {
			t.Errorf("directory not created: %s", d)
		}
	}
}

func TestPaths(t *testing.T) {
	p := New("/base")
	if p.ConfigFile() != filepath.Join("/base", "config.toml") {
		t.Errorf("unexpected config path: %s", p.ConfigFile())
	}
	if p.CatalogueDB() != filepath.Join("/base", "data", "catalogue.db") {
		t.Errorf("unexpected catalogue db path: %s", p.CatalogueDB())
	}
	if p.StatsDB() != filepath.Join("/base", "data", "stats.db") {
		t.Errorf("unexpected stats db path: %s", p.StatsDB())
	}
}
