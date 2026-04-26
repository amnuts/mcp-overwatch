package services

import (
	"encoding/json"
	"fmt"
	"time"

	"mcp-overwatch/internal/catalogue"
	"mcp-overwatch/internal/config"
	"mcp-overwatch/internal/logging"
	"mcp-overwatch/internal/paths"
	"mcp-overwatch/internal/runtime"
	"mcp-overwatch/internal/supervisor"
)

// ServerService manages installed MCP servers — install, uninstall, toggle, configure, update.
type ServerService struct {
	store      *catalogue.Store
	supervisor *supervisor.Supervisor
	runtimes   *runtime.Manager
	paths      *paths.Paths
	logger     *logging.Logger
}

// NewServerService creates a ServerService with the given dependencies.
func NewServerService(store *catalogue.Store, sup *supervisor.Supervisor, runtimes *runtime.Manager, p *paths.Paths, logger *logging.Logger) *ServerService {
	return &ServerService{store: store, supervisor: sup, runtimes: runtimes, paths: p, logger: logger}
}

// ListInstalled returns all installed MCP servers.
func (s *ServerService) ListInstalled() ([]catalogue.InstalledServer, error) {
	return s.store.ListInstalledServers()
}

// GetInstalled returns a single installed server by ID.
func (s *ServerService) GetInstalled(id string) (*catalogue.InstalledServer, error) {
	return s.store.GetInstalledServer(id)
}

// Install creates an installed server record from a registry server entry.
func (s *ServerService) Install(registryServerID string) (*catalogue.InstalledServer, error) {
	// Fetch the registry server by exact ID.
	reg, err := s.store.GetRegistryServer(registryServerID)
	if err != nil {
		return nil, fmt.Errorf("registry server %q not found: %w", registryServerID, err)
	}

	now := time.Now().UTC()
	srv := catalogue.InstalledServer{
		ID:                reg.ID,
		Source:            "registry",
		DisplayName:       reg.DisplayName,
		Description:       reg.Description,
		Version:           reg.Version,
		RegistryType:      reg.RegistryType,
		PackageIdentifier: reg.PackageIdentifier,
		TransportType:     reg.TransportType,
		RemoteURL:         reg.RemoteURL,
		EnvVarsJSON:       reg.EnvVarsJSON,
		IsActive:          false,
		Status:            "stopped",
		InstalledAt:       now,
	}

	// Resolve the command based on registry type.
	switch reg.RegistryType {
	case "npm":
		srv.Command = "npx"
		srv.CommandArgsJSON = mustJSON([]string{"-y", reg.PackageIdentifier})
	case "pip", "pypi":
		srv.Command = "uvx"
		srv.CommandArgsJSON = mustJSON([]string{reg.PackageIdentifier})
	case "oci":
		// OCI packages run via docker.
		srv.Command = "docker"
		// Build docker run args: run --rm -i, pass env vars via -e, then the image.
		args := []string{"run", "--rm", "-i"}
		// Add -e flags for each env var so docker forwards them.
		var envDefs []struct {
			Name string `json:"name"`
		}
		if reg.EnvVarsJSON != "" {
			_ = json.Unmarshal([]byte(reg.EnvVarsJSON), &envDefs)
		}
		for _, e := range envDefs {
			args = append(args, "-e", e.Name)
		}
		// The image is the package identifier (e.g. "docker.io/hashicorp/terraform-mcp-server:0.3.2").
		args = append(args, reg.PackageIdentifier)
		srv.CommandArgsJSON = mustJSON(args)
	default:
		// For remote/SSE servers, no local command needed.
	}

	if err := s.store.InsertInstalledServer(srv); err != nil {
		s.logger.Add(logging.Entry{
			ServerID:  srv.ID,
			Direction: "in",
			Summary:   fmt.Sprintf("Install failed: %v", err),
		})
		return nil, fmt.Errorf("inserting installed server: %w", err)
	}

	s.logger.Add(logging.Entry{
		ServerID:  srv.ID,
		Direction: "in",
		Summary:   fmt.Sprintf("Installed %s v%s (%s)", srv.DisplayName, srv.Version, srv.RegistryType),
	})

	return &srv, nil
}

// Uninstall removes a server from the installed list and stops it if running.
func (s *ServerService) Uninstall(id string) error {
	// Stop the server if running.
	if ms := s.supervisor.Get(id); ms != nil {
		_ = s.supervisor.Stop(id)
	}
	if err := s.store.DeleteInstalledServer(id); err != nil {
		s.logger.Add(logging.Entry{
			ServerID:  id,
			Direction: "in",
			Summary:   fmt.Sprintf("Uninstall failed: %v", err),
		})
		return err
	}
	s.logger.Add(logging.Entry{
		ServerID:  id,
		Direction: "in",
		Summary:   fmt.Sprintf("Uninstalled server %s", id),
	})
	return nil
}

// Toggle enables or disables a server. When enabling, it starts the process;
// when disabling, it stops the process. The is_active flag is always persisted
// regardless of whether the process start/stop succeeds, so the user's intent
// is recorded and the UI stays consistent.
func (s *ServerService) Toggle(id string, active bool) error {
	srv, err := s.store.GetInstalledServer(id)
	if err != nil {
		s.logger.Add(logging.Entry{
			ServerID:  id,
			Direction: "in",
			Summary:   fmt.Sprintf("Toggle failed: %v", err),
		})
		return fmt.Errorf("getting server %s: %w", id, err)
	}

	// Persist is_active flag first so the UI reflects the user's intent
	// even if the process operation fails.
	if err := s.store.UpdateIsActive(id, active); err != nil {
		s.logger.Add(logging.Entry{
			ServerID:  id,
			Direction: "in",
			Summary:   fmt.Sprintf("Toggle failed (persisting is_active): %v", err),
		})
		return fmt.Errorf("persisting is_active for %s: %w", id, err)
	}

	if active {
		s.logger.Add(logging.Entry{
			ServerID:  id,
			Direction: "out",
			Summary:   fmt.Sprintf("Start requested for %s", srv.DisplayName),
		})

		// Validate transport-specific requirements before doing any I/O.
		// Empty transport type is treated as stdio for backward compatibility
		// with installs created before remote transports were supported.
		transportType := srv.TransportType
		if transportType == "" {
			transportType = "stdio"
		}
		switch transportType {
		case "stdio":
			if srv.Command == "" {
				_ = s.store.UpdateIsActive(id, false)
				err := fmt.Errorf("server %s has no command configured — it may require a runtime that is not installed", id)
				s.logger.Add(logging.Entry{
					ServerID:  id,
					Direction: "in",
					Summary:   fmt.Sprintf("Start aborted: %v", err),
				})
				return err
			}
		case "sse", "streamable-http", "http", "streaming-http":
			if srv.RemoteURL == "" {
				_ = s.store.UpdateIsActive(id, false)
				err := fmt.Errorf("server %s has no remote URL configured for %s transport", id, transportType)
				s.logger.Add(logging.Entry{
					ServerID:  id,
					Direction: "in",
					Summary:   fmt.Sprintf("Start aborted: %v", err),
				})
				return err
			}
		default:
			_ = s.store.UpdateIsActive(id, false)
			err := fmt.Errorf("server %s uses unsupported transport type %q", id, transportType)
			s.logger.Add(logging.Entry{
				ServerID:  id,
				Direction: "in",
				Summary:   fmt.Sprintf("Start aborted: %v", err),
			})
			return err
		}

		// Resolve env var values from definitions + user config + secrets.
		// For stdio these become subprocess env; for remote transports they
		// become HTTP headers (the same names cover most auth-token style
		// configuration in the registry today).
		envMap, err := s.buildEnvMap(srv)
		if err != nil {
			_ = s.store.UpdateIsActive(id, false)
			s.logger.Add(logging.Entry{
				ServerID:  id,
				Direction: "in",
				Summary:   fmt.Sprintf("Start aborted (env resolution): %v", err),
			})
			return fmt.Errorf("building env for %s: %w", id, err)
		}

		ms := &supervisor.ManagedServer{
			ID:            srv.ID,
			DisplayName:   srv.DisplayName,
			TransportType: transportType,
		}
		switch transportType {
		case "stdio":
			ms.Command = srv.Command
			ms.Args = parseJSONStringSlice(srv.CommandArgsJSON)
			ms.Env = envMapToSlice(envMap)
			ms.WorkDir = srv.RuntimePath
		default:
			ms.RemoteURL = srv.RemoteURL
			ms.Headers = envMap
		}

		s.supervisor.Register(ms)
		if err := s.supervisor.Start(id); err != nil {
			// Failed to start — revert is_active so the toggle reflects
			// reality. Status is already set to "error" by supervisor.
			_ = s.store.UpdateIsActive(id, false)
			s.logger.Add(logging.Entry{
				ServerID:  id,
				Direction: "in",
				Summary:   fmt.Sprintf("Start failed: %v", err),
			})
			return err
		}
	} else {
		s.logger.Add(logging.Entry{
			ServerID:  id,
			Direction: "out",
			Summary:   fmt.Sprintf("Stop requested for %s", srv.DisplayName),
		})
		// Best-effort stop — the server may not be in the supervisor map
		// (e.g. process already crashed). The is_active flag is already
		// persisted as false above.
		_ = s.supervisor.Stop(id)
		// Ensure DB status reflects stopped regardless of whether the
		// supervisor callback fired (handles the case where the server
		// isn't in the supervisor map or Stop returns early).
		_ = s.store.UpdateInstalledServerStatus(id, "stopped")
	}

	return nil
}

// buildEnvMap resolves the server's env var schema + user config + secrets
// into a NAME→VALUE map. For stdio transports this is converted to KEY=VALUE
// strings via envMapToSlice; for remote transports it is passed through as
// HTTP headers. Secret values are loaded from the OS keychain.
func (s *ServerService) buildEnvMap(srv *catalogue.InstalledServer) (map[string]string, error) {
	var defs []catalogue.EnvVarDef
	if srv.EnvVarsJSON != "" {
		if err := json.Unmarshal([]byte(srv.EnvVarsJSON), &defs); err != nil {
			return nil, fmt.Errorf("parsing env var defs: %w", err)
		}
	}

	userVals := make(map[string]string)
	if srv.UserConfigJSON != "" {
		if err := json.Unmarshal([]byte(srv.UserConfigJSON), &userVals); err != nil {
			return nil, fmt.Errorf("parsing user config: %w", err)
		}
	}

	resolved := make(map[string]string, len(defs))
	for _, def := range defs {
		val := userVals[def.Name]
		if val == "" && def.IsSecret {
			if secret, err := config.GetSecret(srv.ID, def.Name); err == nil {
				val = secret
			}
		}
		if val == "" {
			val = def.Default
		}
		if val != "" {
			resolved[def.Name] = val
		}
	}
	return resolved, nil
}

// envMapToSlice converts a NAME→VALUE map into KEY=VALUE strings suitable for
// passing to a stdio subprocess via exec.Cmd.Env.
func envMapToSlice(m map[string]string) []string {
	if len(m) == 0 {
		return nil
	}
	out := make([]string, 0, len(m))
	for k, v := range m {
		out = append(out, k+"="+v)
	}
	return out
}

// Configure updates user configuration for a server (env vars, args, etc.).
func (s *ServerService) Configure(id string, userConfigJSON string) error {
	if err := s.store.UpdateInstalledServerConfig(id, userConfigJSON); err != nil {
		s.logger.Add(logging.Entry{
			ServerID:  id,
			Direction: "in",
			Summary:   fmt.Sprintf("Configuration update failed: %v", err),
		})
		return err
	}
	s.logger.Add(logging.Entry{
		ServerID:  id,
		Direction: "in",
		Summary:   "Configuration updated",
	})
	return nil
}

// Update upgrades (or downgrades) a server to the available version from the registry.
func (s *ServerService) Update(id string) error {
	srv, err := s.store.GetInstalledServer(id)
	if err != nil {
		s.logger.Add(logging.Entry{
			ServerID:  id,
			Direction: "in",
			Summary:   fmt.Sprintf("Version change failed: %v", err),
		})
		return err
	}
	if srv.AvailableVersion == "" {
		err := fmt.Errorf("no update available for %s", id)
		s.logger.Add(logging.Entry{
			ServerID:  id,
			Direction: "in",
			Summary:   "Version change skipped: no available version",
		})
		return err
	}

	wasActive := srv.IsActive

	s.logger.Add(logging.Entry{
		ServerID:  id,
		Direction: "out",
		Summary:   fmt.Sprintf("Changing %s from v%s to v%s", srv.DisplayName, srv.Version, srv.AvailableVersion),
	})

	// Stop if running.
	if wasActive {
		_ = s.supervisor.Stop(id)
		_ = s.store.UpdateInstalledServerStatus(id, "stopped")
	}

	// Fetch the latest registry entry to get the updated package identifier.
	newVersion := srv.AvailableVersion
	packageIdentifier := srv.PackageIdentifier
	commandArgsJSON := srv.CommandArgsJSON

	// For registry servers, re-fetch to get the updated package identifier.
	if srv.Source == "registry" {
		if reg, err := s.store.GetRegistryServer(id); err == nil {
			packageIdentifier = reg.PackageIdentifier
			// Re-resolve command args with the updated package identifier.
			switch srv.RegistryType {
			case "npm":
				commandArgsJSON = mustJSON([]string{"-y", reg.PackageIdentifier})
			case "pip", "pypi":
				commandArgsJSON = mustJSON([]string{reg.PackageIdentifier})
			case "oci":
				args := []string{"run", "--rm", "-i"}
				var envDefs []struct {
					Name string `json:"name"`
				}
				if reg.EnvVarsJSON != "" {
					_ = json.Unmarshal([]byte(reg.EnvVarsJSON), &envDefs)
				}
				for _, e := range envDefs {
					args = append(args, "-e", e.Name)
				}
				args = append(args, reg.PackageIdentifier)
				commandArgsJSON = mustJSON(args)
			}
		}
	}

	// Persist the version update.
	if err := s.store.UpdateInstalledServerVersion(id, newVersion, packageIdentifier, commandArgsJSON); err != nil {
		s.logger.Add(logging.Entry{
			ServerID:  id,
			Direction: "in",
			Summary:   fmt.Sprintf("Version change failed: %v", err),
		})
		return fmt.Errorf("updating version for %s: %w", id, err)
	}

	s.logger.Add(logging.Entry{
		ServerID:  id,
		Direction: "in",
		Summary:   fmt.Sprintf("Changed %s from v%s to v%s", srv.DisplayName, srv.Version, newVersion),
	})

	// Restart if it was running before the update.
	if wasActive {
		go func() {
			if err := s.Toggle(id, true); err != nil {
				s.logger.Add(logging.Entry{
					ServerID:  id,
					Direction: "in",
					Summary:   fmt.Sprintf("Failed to restart after update: %v", err),
				})
			}
		}()
	}

	return nil
}

// Restart restarts a running server.
func (s *ServerService) Restart(id string) error {
	if err := s.supervisor.Restart(id); err != nil {
		s.logger.Add(logging.Entry{
			ServerID:  id,
			Direction: "in",
			Summary:   fmt.Sprintf("Restart failed: %v", err),
		})
		return err
	}
	return nil
}

// mustJSON marshals v to a JSON string, panicking on error (for static data only).
func mustJSON(v interface{}) string {
	data, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return string(data)
}

// parseJSONStringSlice parses a JSON string slice, returning nil on failure.
func parseJSONStringSlice(raw string) []string {
	if raw == "" {
		return nil
	}
	var result []string
	if err := json.Unmarshal([]byte(raw), &result); err != nil {
		return nil
	}
	return result
}
