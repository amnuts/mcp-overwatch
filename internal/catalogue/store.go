package catalogue

import (
	"database/sql"
	"fmt"
	"strings"
	"time"
)

type Store struct {
	db *sql.DB
}

func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

// --- Registry Servers ---

func (s *Store) UpsertRegistryServer(srv RegistryServer) error {
	_, err := s.db.Exec(`INSERT OR REPLACE INTO registry_servers
		(id, display_name, description, version, status, registry_type,
		 package_identifier, package_version, transport_type, remote_url,
		 env_vars_json, package_args_json, website_url, repository_url,
		 raw_json, synced_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		srv.ID, srv.DisplayName, srv.Description, srv.Version, srv.Status,
		srv.RegistryType, srv.PackageIdentifier, srv.PackageVersion,
		srv.TransportType, srv.RemoteURL, srv.EnvVarsJSON, srv.PackageArgsJSON,
		srv.WebsiteURL, srv.RepositoryURL, srv.RawJSON, srv.SyncedAt,
	)
	return err
}

func (s *Store) ListRegistryServers(offset, limit int, search, registryType, transportType string) ([]RegistryServer, int, error) {
	where := []string{"1=1"}
	args := []any{}

	if search != "" {
		where = append(where, "(display_name LIKE ? OR description LIKE ?)")
		like := "%" + search + "%"
		args = append(args, like, like)
	}
	if registryType != "" {
		where = append(where, "registry_type = ?")
		args = append(args, registryType)
	}
	if transportType != "" {
		where = append(where, "transport_type = ?")
		args = append(args, transportType)
	}

	whereClause := strings.Join(where, " AND ")

	// Count total
	var total int
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM registry_servers WHERE %s", whereClause)
	if err := s.db.QueryRow(countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	// Fetch page
	query := fmt.Sprintf("SELECT id, display_name, description, version, status, registry_type, package_identifier, package_version, transport_type, remote_url, env_vars_json, package_args_json, website_url, repository_url, raw_json, synced_at FROM registry_servers WHERE %s ORDER BY display_name LIMIT ? OFFSET ?", whereClause)
	args = append(args, limit, offset)

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var servers []RegistryServer
	for rows.Next() {
		var srv RegistryServer
		var description, registryT, pkgIdent, pkgVer, transportT, remoteURL sql.NullString
		var envVars, pkgArgs, websiteURL, repoURL, rawJSON sql.NullString
		var status sql.NullString
		var syncedAt sql.NullTime

		if err := rows.Scan(
			&srv.ID, &srv.DisplayName, &description, &srv.Version, &status,
			&registryT, &pkgIdent, &pkgVer, &transportT, &remoteURL,
			&envVars, &pkgArgs, &websiteURL, &repoURL, &rawJSON, &syncedAt,
		); err != nil {
			return nil, 0, err
		}

		srv.Description = description.String
		srv.Status = status.String
		srv.RegistryType = registryT.String
		srv.PackageIdentifier = pkgIdent.String
		srv.PackageVersion = pkgVer.String
		srv.TransportType = transportT.String
		srv.RemoteURL = remoteURL.String
		srv.EnvVarsJSON = envVars.String
		srv.PackageArgsJSON = pkgArgs.String
		srv.WebsiteURL = websiteURL.String
		srv.RepositoryURL = repoURL.String
		srv.RawJSON = rawJSON.String
		if syncedAt.Valid {
			srv.SyncedAt = syncedAt.Time
		}

		servers = append(servers, srv)
	}

	return servers, total, rows.Err()
}

func (s *Store) GetRegistryServer(id string) (*RegistryServer, error) {
	var srv RegistryServer
	var description, registryT, pkgIdent, pkgVer, transportT, remoteURL sql.NullString
	var envVars, pkgArgs, websiteURL, repoURL, rawJSON sql.NullString
	var status sql.NullString
	var syncedAt sql.NullTime

	err := s.db.QueryRow(`SELECT id, display_name, description, version, status, registry_type,
		package_identifier, package_version, transport_type, remote_url,
		env_vars_json, package_args_json, website_url, repository_url,
		raw_json, synced_at FROM registry_servers WHERE id = ?`, id).Scan(
		&srv.ID, &srv.DisplayName, &description, &srv.Version, &status,
		&registryT, &pkgIdent, &pkgVer, &transportT, &remoteURL,
		&envVars, &pkgArgs, &websiteURL, &repoURL, &rawJSON, &syncedAt,
	)
	if err != nil {
		return nil, err
	}
	srv.Description = description.String
	srv.Status = status.String
	srv.RegistryType = registryT.String
	srv.PackageIdentifier = pkgIdent.String
	srv.PackageVersion = pkgVer.String
	srv.TransportType = transportT.String
	srv.RemoteURL = remoteURL.String
	srv.EnvVarsJSON = envVars.String
	srv.PackageArgsJSON = pkgArgs.String
	srv.WebsiteURL = websiteURL.String
	srv.RepositoryURL = repoURL.String
	srv.RawJSON = rawJSON.String
	if syncedAt.Valid {
		srv.SyncedAt = syncedAt.Time
	}
	return &srv, nil
}

// --- Installed Servers ---

func (s *Store) InsertInstalledServer(srv InstalledServer) error {
	isActive := 0
	if srv.IsActive {
		isActive = 1
	}
	var lastUsedAt *time.Time
	if srv.LastUsedAt != nil {
		lastUsedAt = srv.LastUsedAt
	}
	_, err := s.db.Exec(`INSERT INTO installed_servers
		(id, source, display_name, description, version, available_version,
		 registry_type, package_identifier, transport_type, command,
		 command_args_json, remote_url, env_vars_json, user_config_json,
		 is_active, status, error_count, cached_tools_json,
		 cached_resources_json, cached_prompts_json, installed_at,
		 last_used_at, runtime_path)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		srv.ID, srv.Source, srv.DisplayName, srv.Description, srv.Version,
		srv.AvailableVersion, srv.RegistryType, srv.PackageIdentifier,
		srv.TransportType, srv.Command, srv.CommandArgsJSON, srv.RemoteURL,
		srv.EnvVarsJSON, srv.UserConfigJSON, isActive, srv.Status,
		srv.ErrorCount, srv.CachedToolsJSON, srv.CachedResourcesJSON,
		srv.CachedPromptsJSON, srv.InstalledAt, lastUsedAt, srv.RuntimePath,
	)
	return err
}

func (s *Store) ListInstalledServers() ([]InstalledServer, error) {
	rows, err := s.db.Query(`SELECT id, source, display_name, description, version,
		available_version, registry_type, package_identifier, transport_type,
		command, command_args_json, remote_url, env_vars_json, user_config_json,
		is_active, status, error_count, cached_tools_json, cached_resources_json,
		cached_prompts_json, installed_at, last_used_at, runtime_path
		FROM installed_servers ORDER BY display_name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var servers []InstalledServer
	for rows.Next() {
		srv, err := scanInstalledServer(rows)
		if err != nil {
			return nil, err
		}
		servers = append(servers, srv)
	}
	return servers, rows.Err()
}

func (s *Store) GetInstalledServer(id string) (*InstalledServer, error) {
	row := s.db.QueryRow(`SELECT id, source, display_name, description, version,
		available_version, registry_type, package_identifier, transport_type,
		command, command_args_json, remote_url, env_vars_json, user_config_json,
		is_active, status, error_count, cached_tools_json, cached_resources_json,
		cached_prompts_json, installed_at, last_used_at, runtime_path
		FROM installed_servers WHERE id = ?`, id)

	srv, err := scanInstalledServerRow(row)
	if err != nil {
		return nil, err
	}
	return &srv, nil
}

func (s *Store) UpdateInstalledServerStatus(id, status string) error {
	_, err := s.db.Exec("UPDATE installed_servers SET status = ? WHERE id = ?", status, id)
	return err
}

func (s *Store) UpdateIsActive(id string, active bool) error {
	v := 0
	if active {
		v = 1
	}
	_, err := s.db.Exec("UPDATE installed_servers SET is_active = ? WHERE id = ?", v, id)
	return err
}

func (s *Store) UpdateInstalledServerConfig(id, userConfigJSON string) error {
	_, err := s.db.Exec("UPDATE installed_servers SET user_config_json = ? WHERE id = ?", userConfigJSON, id)
	return err
}

func (s *Store) UpdateInstalledServerCachedData(id, toolsJSON, resourcesJSON, promptsJSON string) error {
	_, err := s.db.Exec(`UPDATE installed_servers
		SET cached_tools_json = ?, cached_resources_json = ?, cached_prompts_json = ?
		WHERE id = ?`, toolsJSON, resourcesJSON, promptsJSON, id)
	return err
}

func (s *Store) SetAvailableVersion(id, version string) error {
	_, err := s.db.Exec("UPDATE installed_servers SET available_version = ? WHERE id = ?", version, id)
	return err
}

// UpdateInstalledServerVersion updates version fields and command args after an upgrade/downgrade.
func (s *Store) UpdateInstalledServerVersion(id, version, packageIdentifier, commandArgsJSON string) error {
	_, err := s.db.Exec(`UPDATE installed_servers
		SET version = ?, available_version = '', package_identifier = ?, command_args_json = ?
		WHERE id = ?`, version, packageIdentifier, commandArgsJSON, id)
	return err
}

func (s *Store) DeleteInstalledServer(id string) error {
	_, err := s.db.Exec("DELETE FROM installed_servers WHERE id = ?", id)
	return err
}

func (s *Store) ResetAllStatuses() error {
	_, err := s.db.Exec("UPDATE installed_servers SET status = 'stopped'")
	return err
}

// --- Runtimes ---

func (s *Store) UpsertRuntime(r Runtime) error {
	_, err := s.db.Exec(`INSERT OR REPLACE INTO runtimes
		(id, version, path, size_bytes, installed_at, status)
		VALUES (?, ?, ?, ?, ?, ?)`,
		r.ID, r.Version, r.Path, r.SizeBytes, r.InstalledAt, r.Status,
	)
	return err
}

func (s *Store) DeleteRuntime(id string) error {
	_, err := s.db.Exec("DELETE FROM runtimes WHERE id = ?", id)
	return err
}

func (s *Store) GetRuntime(id string) (*Runtime, error) {
	var r Runtime
	var sizeBytes sql.NullInt64
	var status sql.NullString
	var installedAt sql.NullTime

	err := s.db.QueryRow(`SELECT id, version, path, size_bytes, installed_at, status
		FROM runtimes WHERE id = ?`, id).Scan(
		&r.ID, &r.Version, &r.Path, &sizeBytes, &installedAt, &status,
	)
	if err != nil {
		return nil, err
	}

	r.SizeBytes = sizeBytes.Int64
	r.Status = status.String
	if installedAt.Valid {
		r.InstalledAt = installedAt.Time
	}
	return &r, nil
}

func (s *Store) ListRuntimes() ([]Runtime, error) {
	rows, err := s.db.Query("SELECT id, version, path, size_bytes, installed_at, status FROM runtimes ORDER BY id")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var runtimes []Runtime
	for rows.Next() {
		var r Runtime
		var sizeBytes sql.NullInt64
		var status sql.NullString
		var installedAt sql.NullTime

		if err := rows.Scan(&r.ID, &r.Version, &r.Path, &sizeBytes, &installedAt, &status); err != nil {
			return nil, err
		}
		r.SizeBytes = sizeBytes.Int64
		r.Status = status.String
		if installedAt.Valid {
			r.InstalledAt = installedAt.Time
		}
		runtimes = append(runtimes, r)
	}
	return runtimes, rows.Err()
}

// --- scan helpers ---

type scanner interface {
	Scan(dest ...any) error
}

func scanInstalledServerFromScanner(sc scanner) (InstalledServer, error) {
	var srv InstalledServer
	var description, version, availVer, registryT, pkgIdent sql.NullString
	var command, cmdArgs, remoteURL, envVars, userConfig sql.NullString
	var cachedTools, cachedResources, cachedPrompts, runtimePath sql.NullString
	var status sql.NullString
	var isActive int
	var errorCount sql.NullInt64
	var installedAt sql.NullTime
	var lastUsedAt sql.NullTime

	if err := sc.Scan(
		&srv.ID, &srv.Source, &srv.DisplayName, &description, &version,
		&availVer, &registryT, &pkgIdent, &srv.TransportType,
		&command, &cmdArgs, &remoteURL, &envVars, &userConfig,
		&isActive, &status, &errorCount, &cachedTools, &cachedResources,
		&cachedPrompts, &installedAt, &lastUsedAt, &runtimePath,
	); err != nil {
		return srv, err
	}

	srv.Description = description.String
	srv.Version = version.String
	srv.AvailableVersion = availVer.String
	srv.RegistryType = registryT.String
	srv.PackageIdentifier = pkgIdent.String
	srv.Command = command.String
	srv.CommandArgsJSON = cmdArgs.String
	srv.RemoteURL = remoteURL.String
	srv.EnvVarsJSON = envVars.String
	srv.UserConfigJSON = userConfig.String
	srv.IsActive = isActive != 0
	srv.Status = status.String
	srv.ErrorCount = int(errorCount.Int64)
	srv.CachedToolsJSON = cachedTools.String
	srv.CachedResourcesJSON = cachedResources.String
	srv.CachedPromptsJSON = cachedPrompts.String
	srv.RuntimePath = runtimePath.String
	if installedAt.Valid {
		srv.InstalledAt = installedAt.Time
	}
	if lastUsedAt.Valid {
		t := lastUsedAt.Time
		srv.LastUsedAt = &t
	}

	return srv, nil
}

func scanInstalledServer(rows *sql.Rows) (InstalledServer, error) {
	return scanInstalledServerFromScanner(rows)
}

func scanInstalledServerRow(row *sql.Row) (InstalledServer, error) {
	return scanInstalledServerFromScanner(row)
}
