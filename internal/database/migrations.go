package database

import "database/sql"

// IMPORTANT: modernc.org/sqlite does not support multiple statements in a single
// db.Exec() call. Each DDL statement must be executed separately.
func applyCatalogueMigrations(db *sql.DB) error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS schema_version (
			version INTEGER NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS registry_servers (
			id TEXT PRIMARY KEY,
			display_name TEXT NOT NULL,
			description TEXT,
			version TEXT NOT NULL,
			status TEXT DEFAULT 'active',
			registry_type TEXT,
			package_identifier TEXT,
			package_version TEXT,
			transport_type TEXT,
			remote_url TEXT,
			env_vars_json TEXT,
			package_args_json TEXT,
			website_url TEXT,
			repository_url TEXT,
			raw_json TEXT,
			synced_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS installed_servers (
			id TEXT PRIMARY KEY,
			source TEXT NOT NULL,
			display_name TEXT NOT NULL,
			description TEXT,
			version TEXT,
			available_version TEXT,
			registry_type TEXT,
			package_identifier TEXT,
			transport_type TEXT NOT NULL,
			command TEXT,
			command_args_json TEXT,
			remote_url TEXT,
			env_vars_json TEXT,
			user_config_json TEXT,
			is_active INTEGER DEFAULT 0,
			status TEXT DEFAULT 'stopped',
			error_count INTEGER DEFAULT 0,
			cached_tools_json TEXT,
			cached_resources_json TEXT,
			cached_prompts_json TEXT,
			installed_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			last_used_at TIMESTAMP,
			runtime_path TEXT
		)`,
		`CREATE TABLE IF NOT EXISTS runtimes (
			id TEXT PRIMARY KEY,
			version TEXT NOT NULL,
			path TEXT NOT NULL,
			size_bytes INTEGER,
			installed_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			status TEXT DEFAULT 'ready'
		)`,
	}
	for _, stmt := range stmts {
		if _, err := db.Exec(stmt); err != nil {
			return err
		}
	}

	// Insert schema version if not present
	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM schema_version").Scan(&count); err != nil {
		return err
	}
	if count == 0 {
		_, err := db.Exec("INSERT INTO schema_version VALUES (1)")
		return err
	}
	return nil
}

func applyStatsMigrations(db *sql.DB) error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS schema_version (
			version INTEGER NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS server_events (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			server_id TEXT NOT NULL,
			event_type TEXT NOT NULL,
			tool_name TEXT,
			latency_ms INTEGER,
			payload_bytes_in INTEGER,
			payload_bytes_out INTEGER,
			error_message TEXT,
			timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_events_server ON server_events(server_id, timestamp)`,
		`CREATE INDEX IF NOT EXISTS idx_events_type ON server_events(event_type, timestamp)`,
	}
	for _, stmt := range stmts {
		if _, err := db.Exec(stmt); err != nil {
			return err
		}
	}

	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM schema_version").Scan(&count); err != nil {
		return err
	}
	if count == 0 {
		_, err := db.Exec("INSERT INTO schema_version VALUES (1)")
		return err
	}
	return nil
}
