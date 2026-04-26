package services

import (
	"fmt"

	"mcp-overwatch/internal/catalogue"
	"mcp-overwatch/internal/logging"
)

// CatalogueService exposes registry browsing to the frontend.
type CatalogueService struct {
	store             *catalogue.Store
	registry          *catalogue.RegistryClient
	logger            *logging.Logger
	onUpdateAvailable func(serverID, currentVersion, newVersion string)
}

// NewCatalogueService creates a CatalogueService with the given dependencies.
func NewCatalogueService(store *catalogue.Store, registry *catalogue.RegistryClient, logger *logging.Logger, onUpdateAvailable func(string, string, string)) *CatalogueService {
	return &CatalogueService{store: store, registry: registry, logger: logger, onUpdateAvailable: onUpdateAvailable}
}

// ListAvailable returns a paginated, filtered list of registry servers.
func (s *CatalogueService) ListAvailable(search, registryType, transportType string, offset, limit int) ([]catalogue.RegistryServer, int, error) {
	return s.store.ListRegistryServers(offset, limit, search, registryType, transportType)
}

// SyncRegistry fetches the latest data from the MCP registry and upserts it.
func (s *CatalogueService) SyncRegistry() error {
	s.logger.Add(logging.Entry{
		ServerID:  "system",
		Direction: "out",
		Summary:   "Registry sync started",
	})

	result, err := s.registry.SyncToStore(s.store, s.onUpdateAvailable)
	if err != nil {
		s.logger.Add(logging.Entry{
			ServerID:  "system",
			Direction: "in",
			Summary:   fmt.Sprintf("Registry sync failed: %v", err),
		})
		return err
	}

	s.logger.Add(logging.Entry{
		ServerID:  "system",
		Direction: "in",
		Summary:   fmt.Sprintf("Registry sync complete: %d new, %d updated", result.NewCount, result.UpdatedCount),
	})
	return nil
}
