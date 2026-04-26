package database

import (
	"path/filepath"
	"testing"
)

func TestOpenCatalogueDB(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "catalogue.db")

	db, err := OpenCatalogue(dbPath)
	if err != nil {
		t.Fatalf("failed to open catalogue db: %v", err)
	}
	defer db.Close()

	// Verify tables exist
	tables := []string{"schema_version", "registry_servers", "installed_servers", "runtimes"}
	for _, table := range tables {
		var name string
		err := db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name=?", table).Scan(&name)
		if err != nil {
			t.Errorf("table %s not found: %v", table, err)
		}
	}

	// Verify schema version
	var version int
	err = db.QueryRow("SELECT version FROM schema_version").Scan(&version)
	if err != nil {
		t.Fatalf("failed to read schema version: %v", err)
	}
	if version != 1 {
		t.Errorf("expected schema version 1, got %d", version)
	}
}

func TestOpenStatsDB(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "stats.db")

	db, err := OpenStats(dbPath)
	if err != nil {
		t.Fatalf("failed to open stats db: %v", err)
	}
	defer db.Close()

	var name string
	err = db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='server_events'").Scan(&name)
	if err != nil {
		t.Errorf("table server_events not found: %v", err)
	}
}
