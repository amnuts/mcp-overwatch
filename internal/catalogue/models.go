package catalogue

import "time"

type RegistryServer struct {
	ID                string    `json:"id"`
	DisplayName       string    `json:"display_name"`
	Description       string    `json:"description"`
	Version           string    `json:"version"`
	Status            string    `json:"status"`
	RegistryType      string    `json:"registry_type"`
	PackageIdentifier string    `json:"package_identifier"`
	PackageVersion    string    `json:"package_version"`
	TransportType     string    `json:"transport_type"`
	RemoteURL         string    `json:"remote_url"`
	EnvVarsJSON       string    `json:"env_vars_json"`
	PackageArgsJSON   string    `json:"package_args_json"`
	WebsiteURL        string    `json:"website_url"`
	RepositoryURL     string    `json:"repository_url"`
	RawJSON           string    `json:"raw_json"`
	SyncedAt          time.Time `json:"synced_at"`
}

type InstalledServer struct {
	ID                  string     `json:"id"`
	Source              string     `json:"source"` // "registry" or "custom"
	DisplayName         string     `json:"display_name"`
	Description         string     `json:"description"`
	Version             string     `json:"version"`
	AvailableVersion    string     `json:"available_version"`
	RegistryType        string     `json:"registry_type"`
	PackageIdentifier   string     `json:"package_identifier"`
	TransportType       string     `json:"transport_type"`
	Command             string     `json:"command"`
	CommandArgsJSON     string     `json:"command_args_json"`
	RemoteURL           string     `json:"remote_url"`
	EnvVarsJSON         string     `json:"env_vars_json"`
	UserConfigJSON      string     `json:"user_config_json"`
	IsActive            bool       `json:"is_active"`
	Status              string     `json:"status"`
	ErrorCount          int        `json:"error_count"`
	CachedToolsJSON     string     `json:"cached_tools_json"`
	CachedResourcesJSON string     `json:"cached_resources_json"`
	CachedPromptsJSON   string     `json:"cached_prompts_json"`
	InstalledAt         time.Time  `json:"installed_at"`
	LastUsedAt          *time.Time `json:"last_used_at"`
	RuntimePath         string     `json:"runtime_path"`
}

type Runtime struct {
	ID          string    `json:"id"`
	Version     string    `json:"version"`
	Path        string    `json:"path"`
	SizeBytes   int64     `json:"size_bytes"`
	InstalledAt time.Time `json:"installed_at"`
	Status      string    `json:"status"`
}

type EnvVarDef struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	IsRequired  bool   `json:"isRequired"`
	IsSecret    bool   `json:"isSecret"`
	Default     string `json:"default"`
}
