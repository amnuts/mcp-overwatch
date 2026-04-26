package catalogue

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"mcp-overwatch/internal/database"
)

func wrapServers(servers ...RegistryServerRaw) []RegistryServerWrapper {
	wrappers := make([]RegistryServerWrapper, len(servers))
	for i, s := range servers {
		wrappers[i] = RegistryServerWrapper{Server: s}
	}
	return wrappers
}

func TestFetchPage(t *testing.T) {
	response := RegistryResponse{
		Servers: wrapServers(
			RegistryServerRaw{
				Name:        "test-server",
				Description: "A test server",
				Version:     "1.0.0",
				Packages: []RegistryPackage{
					{
						RegistryType: "npm",
						Name:         "@test/server",
						Version:      "1.0.0",
					},
				},
			},
		),
		Metadata: RegistryMetadata{NextCursor: ""},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v0.1/servers" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := NewRegistryClient(server.URL)
	result, nextCursor, err := client.FetchPage("", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("expected 1 server, got %d", len(result))
	}
	if result[0].Name != "test-server" {
		t.Errorf("expected name test-server, got %s", result[0].Name)
	}
	if nextCursor != "" {
		t.Errorf("expected empty cursor, got %s", nextCursor)
	}
}

func TestFetchAllWithPagination(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		cursor := r.URL.Query().Get("cursor")
		var resp RegistryResponse
		if cursor == "" {
			resp = RegistryResponse{
				Servers:  wrapServers(RegistryServerRaw{Name: "server-1", Version: "1.0.0"}),
				Metadata: RegistryMetadata{NextCursor: "page2"},
			}
		} else if cursor == "page2" {
			resp = RegistryResponse{
				Servers:  wrapServers(RegistryServerRaw{Name: "server-2", Version: "2.0.0"}),
				Metadata: RegistryMetadata{NextCursor: ""},
			}
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewRegistryClient(server.URL)
	result, err := client.FetchAll("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 2 {
		t.Fatalf("expected 2 servers, got %d", len(result))
	}
	if callCount != 2 {
		t.Errorf("expected 2 HTTP calls, got %d", callCount)
	}
}

func TestSyncToStore(t *testing.T) {
	// Set up mock HTTP server with registry data.
	registryServers := RegistryResponse{
		Servers: wrapServers(
			RegistryServerRaw{
				Name:        "my-server",
				Description: "My MCP server",
				Version:     "2.0.0",
				Packages: []RegistryPackage{
					{
						RegistryType: "npm",
						Name:         "@my/server",
						Version:      "2.0.0",
						Transport:    []RegistryTransport{{Type: "stdio"}},
						EnvironmentVars: []RegistryEnvVar{
							{Name: "API_KEY", Description: "The API key", IsRequired: true, IsSecret: true},
						},
						PackageArguments: []RegistryPackageArg{
							{Name: "--verbose", Description: "Enable verbose output"},
						},
					},
				},
			},
			RegistryServerRaw{
				Name:    "deleted-server",
				Version: "1.0.0",
			},
		),
		Metadata: RegistryMetadata{NextCursor: ""},
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(registryServers)
	}))
	defer srv.Close()

	// Set up real SQLite DB.
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	db, err := database.OpenCatalogue(dbPath)
	if err != nil {
		t.Fatalf("failed to open catalogue: %v", err)
	}
	defer db.Close()

	store := NewStore(db)

	// Insert an installed server with an older version.
	err = store.InsertInstalledServer(InstalledServer{
		ID:                "my-server",
		Source:            "registry",
		DisplayName:       "My Server",
		Version:           "1.0.0",
		RegistryType:      "npm",
		PackageIdentifier: "@my/server",
		TransportType:     "stdio",
		Status:            "stopped",
		InstalledAt:       time.Now(),
	})
	if err != nil {
		t.Fatalf("failed to insert installed server: %v", err)
	}

	// Track update callbacks.
	var updates []struct {
		serverID, current, newVer string
	}
	onUpdate := func(serverID, currentVersion, newVersion string) {
		updates = append(updates, struct {
			serverID, current, newVer string
		}{serverID, currentVersion, newVersion})
	}

	client := NewRegistryClient(srv.URL)
	result, err := client.SyncToStore(store, onUpdate)
	if err != nil {
		t.Fatalf("SyncToStore failed: %v", err)
	}

	// Verify sync result counts (both are new, no deleted status in new format).
	if result.NewCount != 2 {
		t.Errorf("expected 2 new, got %d", result.NewCount)
	}
	if result.UpdatedCount != 1 {
		t.Errorf("expected 1 updated, got %d", result.UpdatedCount)
	}

	// Verify registry server was upserted.
	regServers, _, err := store.ListRegistryServers(0, 10, "", "", "")
	if err != nil {
		t.Fatalf("failed to list registry servers: %v", err)
	}
	if len(regServers) != 2 {
		t.Fatalf("expected 2 registry servers, got %d", len(regServers))
	}

	// Find the active server and verify fields.
	var activeServer *RegistryServer
	for _, s := range regServers {
		if s.ID == "my-server" {
			activeServer = &s
			break
		}
	}
	if activeServer == nil {
		t.Fatal("my-server not found in registry servers")
	}
	if activeServer.RegistryType != "npm" {
		t.Errorf("expected registry type npm, got %s", activeServer.RegistryType)
	}
	if activeServer.PackageIdentifier != "@my/server" {
		t.Errorf("expected package identifier @my/server, got %s", activeServer.PackageIdentifier)
	}
	if activeServer.TransportType != "stdio" {
		t.Errorf("expected transport type stdio, got %s", activeServer.TransportType)
	}
	if activeServer.EnvVarsJSON == "" {
		t.Error("expected env vars JSON to be populated")
	}

	// Verify the callback was called with correct versions.
	if len(updates) != 1 {
		t.Fatalf("expected 1 update callback, got %d", len(updates))
	}
	if updates[0].serverID != "my-server" {
		t.Errorf("expected server ID my-server, got %s", updates[0].serverID)
	}
	if updates[0].current != "1.0.0" {
		t.Errorf("expected current version 1.0.0, got %s", updates[0].current)
	}
	if updates[0].newVer != "2.0.0" {
		t.Errorf("expected new version 2.0.0, got %s", updates[0].newVer)
	}

	// Verify available_version was set on the installed server.
	inst, err := store.GetInstalledServer("my-server")
	if err != nil {
		t.Fatalf("failed to get installed server: %v", err)
	}
	if inst.AvailableVersion != "2.0.0" {
		t.Errorf("expected available version 2.0.0, got %s", inst.AvailableVersion)
	}
}
