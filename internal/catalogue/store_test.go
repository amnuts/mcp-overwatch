package catalogue

import (
	"path/filepath"
	"testing"
	"time"

	"mcp-overwatch/internal/database"
)

func newTestStore(t *testing.T) *Store {
	t.Helper()
	tmpDir := t.TempDir()
	db, err := database.OpenCatalogue(filepath.Join(tmpDir, "test.db"))
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return NewStore(db)
}

func TestUpsertAndListRegistryServers(t *testing.T) {
	store := newTestStore(t)

	s := RegistryServer{
		ID:                "srv-1",
		DisplayName:       "Test Server",
		Description:       "A test server",
		Version:           "1.0.0",
		Status:            "active",
		RegistryType:      "npm",
		PackageIdentifier: "@test/server",
		PackageVersion:    "1.0.0",
		TransportType:     "stdio",
		SyncedAt:          time.Now().UTC().Truncate(time.Second),
	}

	if err := store.UpsertRegistryServer(s); err != nil {
		t.Fatalf("UpsertRegistryServer failed: %v", err)
	}

	servers, total, err := store.ListRegistryServers(0, 10, "", "", "")
	if err != nil {
		t.Fatalf("ListRegistryServers failed: %v", err)
	}
	if total != 1 {
		t.Errorf("expected total 1, got %d", total)
	}
	if len(servers) != 1 {
		t.Fatalf("expected 1 server, got %d", len(servers))
	}
	if servers[0].DisplayName != "Test Server" {
		t.Errorf("expected display_name 'Test Server', got %q", servers[0].DisplayName)
	}

	// Upsert again with updated name
	s.DisplayName = "Updated Server"
	if err := store.UpsertRegistryServer(s); err != nil {
		t.Fatalf("UpsertRegistryServer (update) failed: %v", err)
	}
	servers, total, err = store.ListRegistryServers(0, 10, "", "", "")
	if err != nil {
		t.Fatalf("ListRegistryServers after update failed: %v", err)
	}
	if total != 1 {
		t.Errorf("expected total 1 after upsert, got %d", total)
	}
	if servers[0].DisplayName != "Updated Server" {
		t.Errorf("expected 'Updated Server', got %q", servers[0].DisplayName)
	}
}

func TestSearchRegistryServers(t *testing.T) {
	store := newTestStore(t)

	servers := []RegistryServer{
		{ID: "srv-1", DisplayName: "GitHub Copilot", Description: "AI coding", Version: "1.0", RegistryType: "npm", TransportType: "stdio", SyncedAt: time.Now().UTC()},
		{ID: "srv-2", DisplayName: "Slack Bot", Description: "Messaging", Version: "2.0", RegistryType: "pip", TransportType: "sse", SyncedAt: time.Now().UTC()},
		{ID: "srv-3", DisplayName: "Database Explorer", Description: "SQL tools", Version: "1.0", RegistryType: "npm", TransportType: "stdio", SyncedAt: time.Now().UTC()},
	}
	for _, s := range servers {
		if err := store.UpsertRegistryServer(s); err != nil {
			t.Fatalf("UpsertRegistryServer failed: %v", err)
		}
	}

	// Search by name
	results, total, err := store.ListRegistryServers(0, 10, "Slack", "", "")
	if err != nil {
		t.Fatalf("search failed: %v", err)
	}
	if total != 1 || results[0].ID != "srv-2" {
		t.Errorf("expected srv-2 from search, got total=%d", total)
	}

	// Filter by registry type
	results, total, err = store.ListRegistryServers(0, 10, "", "npm", "")
	if err != nil {
		t.Fatalf("filter by registry_type failed: %v", err)
	}
	if total != 2 {
		t.Errorf("expected 2 npm servers, got %d", total)
	}

	// Filter by transport type
	results, total, err = store.ListRegistryServers(0, 10, "", "", "sse")
	if err != nil {
		t.Fatalf("filter by transport_type failed: %v", err)
	}
	if total != 1 {
		t.Errorf("expected 1 sse server, got %d", total)
	}

	// Pagination
	results, total, err = store.ListRegistryServers(0, 2, "", "", "")
	if err != nil {
		t.Fatalf("pagination failed: %v", err)
	}
	if total != 3 {
		t.Errorf("expected total 3, got %d", total)
	}
	if len(results) != 2 {
		t.Errorf("expected 2 results with limit 2, got %d", len(results))
	}

	results, _, err = store.ListRegistryServers(2, 2, "", "", "")
	if err != nil {
		t.Fatalf("pagination offset failed: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("expected 1 result with offset 2, got %d", len(results))
	}
}

func TestInstalledServerCRUD(t *testing.T) {
	store := newTestStore(t)

	s := InstalledServer{
		ID:              "inst-1",
		Source:          "registry",
		DisplayName:     "My Server",
		Description:     "Installed server",
		Version:         "1.0.0",
		TransportType:   "stdio",
		Command:         "node",
		CommandArgsJSON: `["server.js"]`,
		IsActive:        true,
		Status:          "running",
		InstalledAt:     time.Now().UTC().Truncate(time.Second),
	}

	if err := store.InsertInstalledServer(s); err != nil {
		t.Fatalf("InsertInstalledServer failed: %v", err)
	}

	// List
	servers, err := store.ListInstalledServers()
	if err != nil {
		t.Fatalf("ListInstalledServers failed: %v", err)
	}
	if len(servers) != 1 {
		t.Fatalf("expected 1 installed server, got %d", len(servers))
	}
	if servers[0].DisplayName != "My Server" {
		t.Errorf("expected 'My Server', got %q", servers[0].DisplayName)
	}
	if !servers[0].IsActive {
		t.Error("expected IsActive to be true")
	}

	// Get
	got, err := store.GetInstalledServer("inst-1")
	if err != nil {
		t.Fatalf("GetInstalledServer failed: %v", err)
	}
	if got.ID != "inst-1" {
		t.Errorf("expected id inst-1, got %q", got.ID)
	}

	// Get nonexistent
	_, err = store.GetInstalledServer("nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent server")
	}

	// Update status
	if err := store.UpdateInstalledServerStatus("inst-1", "stopped"); err != nil {
		t.Fatalf("UpdateInstalledServerStatus failed: %v", err)
	}
	got, _ = store.GetInstalledServer("inst-1")
	if got.Status != "stopped" {
		t.Errorf("expected status 'stopped', got %q", got.Status)
	}

	// Update config
	if err := store.UpdateInstalledServerConfig("inst-1", `{"key":"val"}`); err != nil {
		t.Fatalf("UpdateInstalledServerConfig failed: %v", err)
	}
	got, _ = store.GetInstalledServer("inst-1")
	if got.UserConfigJSON != `{"key":"val"}` {
		t.Errorf("expected user_config_json to be set, got %q", got.UserConfigJSON)
	}

	// Update cached data
	if err := store.UpdateInstalledServerCachedData("inst-1", `["tool1"]`, `["res1"]`, `["prompt1"]`); err != nil {
		t.Fatalf("UpdateInstalledServerCachedData failed: %v", err)
	}
	got, _ = store.GetInstalledServer("inst-1")
	if got.CachedToolsJSON != `["tool1"]` {
		t.Errorf("expected cached_tools_json, got %q", got.CachedToolsJSON)
	}

	// Set available version
	if err := store.SetAvailableVersion("inst-1", "2.0.0"); err != nil {
		t.Fatalf("SetAvailableVersion failed: %v", err)
	}
	got, _ = store.GetInstalledServer("inst-1")
	if got.AvailableVersion != "2.0.0" {
		t.Errorf("expected available_version '2.0.0', got %q", got.AvailableVersion)
	}

	// Delete
	if err := store.DeleteInstalledServer("inst-1"); err != nil {
		t.Fatalf("DeleteInstalledServer failed: %v", err)
	}
	servers, _ = store.ListInstalledServers()
	if len(servers) != 0 {
		t.Errorf("expected 0 servers after delete, got %d", len(servers))
	}
}

func TestResetAllStatuses(t *testing.T) {
	store := newTestStore(t)

	for _, s := range []InstalledServer{
		{ID: "s1", Source: "registry", DisplayName: "S1", TransportType: "stdio", Status: "running", InstalledAt: time.Now().UTC()},
		{ID: "s2", Source: "custom", DisplayName: "S2", TransportType: "stdio", Status: "error", InstalledAt: time.Now().UTC()},
	} {
		if err := store.InsertInstalledServer(s); err != nil {
			t.Fatalf("insert failed: %v", err)
		}
	}

	if err := store.ResetAllStatuses(); err != nil {
		t.Fatalf("ResetAllStatuses failed: %v", err)
	}

	servers, _ := store.ListInstalledServers()
	for _, s := range servers {
		if s.Status != "stopped" {
			t.Errorf("expected status 'stopped' for %s, got %q", s.ID, s.Status)
		}
	}
}

func TestRuntimeCRUD(t *testing.T) {
	store := newTestStore(t)

	r := Runtime{
		ID:          "node-22",
		Version:     "22.0.0",
		Path:        "/opt/node/22",
		SizeBytes:   50000000,
		InstalledAt: time.Now().UTC().Truncate(time.Second),
		Status:      "ready",
	}

	if err := store.UpsertRuntime(r); err != nil {
		t.Fatalf("UpsertRuntime failed: %v", err)
	}

	got, err := store.GetRuntime("node-22")
	if err != nil {
		t.Fatalf("GetRuntime failed: %v", err)
	}
	if got.Version != "22.0.0" {
		t.Errorf("expected version 22.0.0, got %q", got.Version)
	}
	if got.SizeBytes != 50000000 {
		t.Errorf("expected size 50000000, got %d", got.SizeBytes)
	}

	// List
	runtimes, err := store.ListRuntimes()
	if err != nil {
		t.Fatalf("ListRuntimes failed: %v", err)
	}
	if len(runtimes) != 1 {
		t.Errorf("expected 1 runtime, got %d", len(runtimes))
	}

	// Get nonexistent
	_, err = store.GetRuntime("nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent runtime")
	}

	// Upsert update
	r.Version = "22.1.0"
	if err := store.UpsertRuntime(r); err != nil {
		t.Fatalf("UpsertRuntime (update) failed: %v", err)
	}
	got, _ = store.GetRuntime("node-22")
	if got.Version != "22.1.0" {
		t.Errorf("expected version 22.1.0 after upsert, got %q", got.Version)
	}
}
