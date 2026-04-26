package catalogue

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

const DefaultRegistryURL = "https://registry.modelcontextprotocol.io"

type RegistryClient struct {
	baseURL    string
	httpClient *http.Client
}

// RegistryResponse matches the current v0.1 API response shape.
type RegistryResponse struct {
	Servers  []RegistryServerWrapper `json:"servers"`
	Metadata RegistryMetadata        `json:"metadata"`
}

type RegistryServerWrapper struct {
	Server RegistryServerRaw      `json:"server"`
	Meta   map[string]interface{} `json:"_meta"`
}

type RegistryMetadata struct {
	NextCursor string `json:"nextCursor"`
	Count      int    `json:"count"`
}

type RegistryServerRaw struct {
	Name        string              `json:"name"`
	Title       string              `json:"title"`
	Description string              `json:"description"`
	Version     string              `json:"version"`
	WebsiteURL  string              `json:"websiteUrl"`
	Repository  *RegistryRepository `json:"repository"`
	Packages    []RegistryPackage   `json:"packages"`
	Remotes     []RegistryRemote    `json:"remotes"`
}

type RegistryRepository struct {
	URL       string `json:"url"`
	Source    string `json:"source"`
	ID        string `json:"id"`
	Subfolder string `json:"subfolder"`
}

type RegistryPackage struct {
	RegistryType     string               `json:"registryType"`
	Name             string               `json:"name"`
	Identifier       string               `json:"identifier"`
	Version          string               `json:"version"`
	RuntimeHint      string               `json:"runtimeHint"`
	Transport        FlexTransportList    `json:"transport"`
	EnvironmentVars  []RegistryEnvVar     `json:"environmentVariables"`
	PackageArguments []RegistryPackageArg `json:"packageArguments"`
}

// PackageID returns the best identifier for this package (identifier field, falling back to name).
func (p RegistryPackage) PackageID() string {
	if p.Identifier != "" {
		return p.Identifier
	}
	return p.Name
}

type RegistryTransport struct {
	Type string `json:"type"`
}

// FlexTransportList handles the registry API returning transport as either
// a single object {"type":"stdio"} or an array [{"type":"stdio"}].
type FlexTransportList []RegistryTransport

func (f *FlexTransportList) UnmarshalJSON(data []byte) error {
	// Try array first.
	var arr []RegistryTransport
	if err := json.Unmarshal(data, &arr); err == nil {
		*f = arr
		return nil
	}
	// Try single object.
	var single RegistryTransport
	if err := json.Unmarshal(data, &single); err == nil {
		*f = []RegistryTransport{single}
		return nil
	}
	// Ignore unparseable transport data.
	*f = nil
	return nil
}

type RegistryRemote struct {
	Type string `json:"type"`
	URL  string `json:"url"`
}

type RegistryEnvVar struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	IsRequired  bool   `json:"isRequired"`
	IsSecret    bool   `json:"isSecret"`
	Default     string `json:"default"`
}

type RegistryPackageArg struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	IsRequired  bool   `json:"isRequired"`
	Default     string `json:"default"`
}

type SyncResult struct {
	NewCount     int
	UpdatedCount int
	DeletedCount int
}

func NewRegistryClient(baseURL string) *RegistryClient {
	return &RegistryClient{
		baseURL:    baseURL,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

func (c *RegistryClient) FetchPage(cursor, updatedSince string) ([]RegistryServerRaw, string, error) {
	u, err := url.Parse(c.baseURL + "/v0.1/servers")
	if err != nil {
		return nil, "", fmt.Errorf("invalid base URL: %w", err)
	}
	q := u.Query()
	q.Set("limit", "100")
	if cursor != "" {
		q.Set("cursor", cursor)
	}
	if updatedSince != "" {
		q.Set("updated_since", updatedSince)
	}
	u.RawQuery = q.Encode()

	resp, err := c.httpClient.Get(u.String())
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("registry returned status %d", resp.StatusCode)
	}

	var result RegistryResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, "", err
	}

	// Unwrap server entries from their wrapper objects.
	servers := make([]RegistryServerRaw, 0, len(result.Servers))
	for _, w := range result.Servers {
		servers = append(servers, w.Server)
	}

	return servers, result.Metadata.NextCursor, nil
}

// FetchAll paginates through all servers from the registry.
func (c *RegistryClient) FetchAll(updatedSince string) ([]RegistryServerRaw, error) {
	var all []RegistryServerRaw
	cursor := ""
	for {
		servers, next, err := c.FetchPage(cursor, updatedSince)
		if err != nil {
			return nil, err
		}
		all = append(all, servers...)
		if next == "" {
			break
		}
		cursor = next
	}
	return all, nil
}

// SyncToStore fetches all servers from the registry and upserts them into the store.
// It compares versions against installed servers and calls onUpdateAvailable when
// a newer version is found. Returns a SyncResult with counts.
func (c *RegistryClient) SyncToStore(store *Store, onUpdateAvailable func(serverID, currentVersion, newVersion string)) (*SyncResult, error) {
	servers, err := c.FetchAll("")
	if err != nil {
		return nil, fmt.Errorf("fetching registry servers: %w", err)
	}

	// Build a map of installed servers for version comparison.
	installed, err := store.ListInstalledServers()
	if err != nil {
		return nil, fmt.Errorf("listing installed servers: %w", err)
	}
	installedByID := make(map[string]InstalledServer, len(installed))
	for _, s := range installed {
		installedByID[s.ID] = s
	}

	result := &SyncResult{}
	now := time.Now().UTC()

	for _, raw := range servers {
		// Skip entries with no name.
		if raw.Name == "" {
			continue
		}

		// Marshal raw JSON for storage.
		rawJSON, err := json.Marshal(raw)
		if err != nil {
			return nil, fmt.Errorf("marshalling raw server %s: %w", raw.Name, err)
		}

		displayName := raw.Title
		if displayName == "" {
			displayName = raw.Name
		}

		srv := RegistryServer{
			ID:          raw.Name,
			DisplayName: displayName,
			Description: raw.Description,
			Version:     raw.Version,
			WebsiteURL:  raw.WebsiteURL,
			RawJSON:     string(rawJSON),
			SyncedAt:    now,
		}

		// Extract repository URL if available.
		if raw.Repository != nil {
			srv.RepositoryURL = raw.Repository.URL
		}

		// Flatten the first package entry if available.
		if len(raw.Packages) > 0 {
			pkg := raw.Packages[0]
			srv.RegistryType = pkg.RegistryType
			srv.PackageIdentifier = pkg.PackageID()
			srv.PackageVersion = pkg.Version

			if len(pkg.Transport) > 0 {
				srv.TransportType = pkg.Transport[0].Type
			}

			if len(pkg.EnvironmentVars) > 0 {
				envJSON, _ := json.Marshal(pkg.EnvironmentVars)
				srv.EnvVarsJSON = string(envJSON)
			}
			if len(pkg.PackageArguments) > 0 {
				argsJSON, _ := json.Marshal(pkg.PackageArguments)
				srv.PackageArgsJSON = string(argsJSON)
			}
		}

		// Flatten the first remote entry if available.
		if len(raw.Remotes) > 0 {
			if srv.TransportType == "" {
				srv.TransportType = raw.Remotes[0].Type
			}
			srv.RemoteURL = raw.Remotes[0].URL
		}

		result.NewCount++

		if err := store.UpsertRegistryServer(srv); err != nil {
			return nil, fmt.Errorf("upserting server %s: %w", raw.Name, err)
		}

		// Check if installed version is older than registry version.
		if inst, ok := installedByID[raw.Name]; ok {
			if inst.Version != "" && raw.Version != "" && inst.Version != raw.Version {
				if err := store.SetAvailableVersion(inst.ID, raw.Version); err != nil {
					return nil, fmt.Errorf("setting available version for %s: %w", inst.ID, err)
				}
				if onUpdateAvailable != nil {
					onUpdateAvailable(inst.ID, inst.Version, raw.Version)
				}
				result.UpdatedCount++
			}
		}
	}

	return result, nil
}
