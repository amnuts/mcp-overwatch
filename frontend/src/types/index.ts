// TypeScript types matching Go models from internal/catalogue/models.go

export interface InstalledServer {
    id: string;
    source: string;
    display_name: string;
    description: string;
    version: string;
    available_version: string;
    registry_type: string;
    package_identifier: string;
    transport_type: string;
    command: string;
    command_args_json: string;
    remote_url: string;
    env_vars_json: string;
    user_config_json: string;
    is_active: boolean;
    status: 'stopped' | 'starting' | 'running' | 'error';
    error_count: number;
    cached_tools_json: string;
    cached_resources_json: string;
    cached_prompts_json: string;
    installed_at: string;
    last_used_at: string | null;
    runtime_path: string;
}

export interface RegistryServer {
    id: string;
    display_name: string;
    description: string;
    version: string;
    status: string;
    registry_type: string;
    package_identifier: string;
    package_version: string;
    transport_type: string;
    remote_url: string;
    env_vars_json: string;
    package_args_json: string;
    website_url: string;
    repository_url: string;
    raw_json: string;
    synced_at: string;
}

export interface LogEntry {
    timestamp: string;
    server_id: string;
    direction: string;
    summary: string;
    raw?: string;
}

export interface EnvVarDef {
    name: string;
    description: string;
    isRequired: boolean;
    isSecret: boolean;
    default: string;
}

export interface AppConfig {
    proxy: { port: number; namespace_format: string };
    sync: { interval_hours: number; last_sync: string };
    app: { start_on_boot: boolean; minimize_to_tray: boolean };
    logging: { retention_days: number; ring_buffer_size: number };
    stats: { retention_days: number; flush_seconds: number };
}

export interface RuntimeInfo {
    id: string;
    version: string;
    path: string;
    size_bytes: number;
    installed_at: string;
    status: string;
}

export interface ClientInfo {
    name: string;
    installed: boolean;
    configPath: string;
}

export type TabId = 'installed' | 'marketplace' | 'import' | 'logs' | 'stats';

export type StatsWindow = '24h' | '7d' | '30d' | 'all';

export interface StatsSummary {
    total_calls: number;
    total_errors: number;
    error_rate: number;
    avg_latency_ms: number;
    unique_servers: number;
    unique_tools: number;
}

export interface ServerStat {
    server_id: string;
    calls: number;
    errors: number;
    error_rate: number;
    avg_latency_ms: number;
    p95_latency_ms: number;
    max_latency_ms: number;
    last_activity: string;
}

export interface ToolStat {
    tool_name: string;
    calls: number;
    errors: number;
    avg_latency_ms: number;
    max_latency_ms: number;
    last_call: string;
}

export interface ActivityBucket {
    bucket_start: string;
    bucket_seconds: number;
    calls: number;
    errors: number;
}
