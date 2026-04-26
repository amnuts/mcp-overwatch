/**
 * Frontend service calls for MCP Overwatch.
 *
 * These delegate to the generated Wails bindings which communicate
 * with the Go backend via the Wails runtime bridge.
 */

import type {
    InstalledServer,
    RegistryServer,
    LogEntry,
    AppConfig,
    RuntimeInfo,
    ClientInfo,
    StatsSummary,
    ServerStat,
    ToolStat,
    ActivityBucket,
    StatsWindow,
} from '../types';

import {
    ServerService,
    CatalogueService,
    ImportService,
    LogService,
    SettingsService,
    StatsService,
} from '../../bindings/mcp-overwatch/internal/services/index.js';

// ── Server Service ──────────────────────────────────────────────────────

export async function listInstalled(): Promise<InstalledServer[]> {
    try {
        const result = await ServerService.ListInstalled();
        return result as unknown as InstalledServer[];
    } catch (e) {
        console.error('listInstalled failed', e);
        return [];
    }
}

export async function getInstalled(id: string): Promise<InstalledServer | null> {
    try {
        const result = await ServerService.GetInstalled(id);
        return result as unknown as InstalledServer | null;
    } catch (e) {
        console.error('getInstalled failed', e);
        return null;
    }
}

export async function installServer(registryServerID: string): Promise<InstalledServer | null> {
    try {
        const result = await ServerService.Install(registryServerID);
        return result as unknown as InstalledServer | null;
    } catch (e) {
        console.error('installServer failed', e);
        return null;
    }
}

export async function uninstallServer(id: string): Promise<void> {
    await ServerService.Uninstall(id);
}

export async function toggleServer(id: string, active: boolean): Promise<void> {
    await ServerService.Toggle(id, active);
}

export async function configureServer(id: string, userConfigJSON: string): Promise<void> {
    await ServerService.Configure(id, userConfigJSON);
}

export async function updateServer(id: string): Promise<void> {
    await ServerService.Update(id);
}

export async function restartServer(id: string): Promise<void> {
    await ServerService.Restart(id);
}

// ── Import Service ──────────────────────────────────────────────────────

export async function importFromGitHub(url: string): Promise<InstalledServer | null> {
    const result = await ImportService.ImportFromGitHub(url);
    return result as unknown as InstalledServer | null;
}

export async function importFromLocal(dir: string, copy: boolean): Promise<InstalledServer | null> {
    const result = await ImportService.ImportFromLocal(dir, copy);
    return result as unknown as InstalledServer | null;
}

export async function browseDirectory(): Promise<string> {
    const result = await ImportService.BrowseDirectory();
    return result as unknown as string;
}

// ── Catalogue Service ───────────────────────────────────────────────────

export async function listAvailable(
    search: string,
    registryType: string,
    transportType: string,
    offset: number,
    limit: number,
): Promise<{ servers: RegistryServer[]; total: number }> {
    try {
        const result = await CatalogueService.ListAvailable(search, registryType, transportType, offset, limit);
        // Go multi-return comes back as a tuple: [RegistryServer[], number]
        const [servers, total] = result as unknown as [RegistryServer[], number];
        return { servers: servers || [], total };
    } catch (e) {
        console.error('listAvailable failed', e);
        return { servers: [], total: 0 };
    }
}

export async function syncRegistry(): Promise<void> {
    try {
        await CatalogueService.SyncRegistry();
    } catch (e) {
        console.error('syncRegistry failed', e);
        throw e;
    }
}

// ── Log Service ─────────────────────────────────────────────────────────

export async function getRecentLogs(count: number): Promise<LogEntry[]> {
    try {
        const result = await LogService.GetRecentLogs(count);
        return result as unknown as LogEntry[];
    } catch (e) {
        console.error('getRecentLogs failed', e);
        return [];
    }
}

// ── Settings Service ────────────────────────────────────────────────────

export async function getSettings(): Promise<AppConfig> {
    try {
        const result = await SettingsService.GetSettings();
        return result as unknown as AppConfig;
    } catch (e) {
        console.error('getSettings failed', e);
        return {
            proxy: { port: 3100, namespace_format: '{server}__{tool}' },
            sync: { interval_hours: 24, last_sync: '' },
            app: { start_on_boot: false, minimize_to_tray: true },
            logging: { retention_days: 7, ring_buffer_size: 1000 },
            stats: { retention_days: 30, flush_seconds: 30 },
        };
    }
}

export async function saveSettings(cfg: AppConfig): Promise<void> {
    await SettingsService.SaveSettings(cfg as any);
}

export async function registerClaudeDesktop(): Promise<void> {
    await SettingsService.RegisterClaudeDesktop();
}

export async function registerClaudeCode(): Promise<void> {
    await SettingsService.RegisterClaudeCode();
}

export async function detectClients(): Promise<ClientInfo[]> {
    try {
        const result = await SettingsService.DetectClients();
        return result as unknown as ClientInfo[];
    } catch (e) {
        console.error('detectClients failed', e);
        return [];
    }
}

export async function getRuntimes(): Promise<RuntimeInfo[]> {
    try {
        const result = await SettingsService.GetRuntimes();
        return result as unknown as RuntimeInfo[];
    } catch (e) {
        console.error('getRuntimes failed', e);
        return [];
    }
}

export async function downloadRuntime(runtimeID: string): Promise<void> {
    await SettingsService.DownloadRuntime(runtimeID);
}

export async function deleteRuntime(runtimeID: string): Promise<void> {
    await SettingsService.DeleteRuntime(runtimeID);
}

export async function checkDocker(): Promise<boolean> {
    try {
        const result = await SettingsService.CheckDocker();
        return result as unknown as boolean;
    } catch {
        return false;
    }
}

// ── Stats Service ───────────────────────────────────────────────────────

export async function getStatsSummary(window: StatsWindow): Promise<StatsSummary> {
    try {
        const result = await StatsService.GetSummary(window);
        return result as unknown as StatsSummary;
    } catch (e) {
        console.error('getStatsSummary failed', e);
        return {
            total_calls: 0,
            total_errors: 0,
            error_rate: 0,
            avg_latency_ms: 0,
            unique_servers: 0,
            unique_tools: 0,
        };
    }
}

export async function getServerStats(window: StatsWindow): Promise<ServerStat[]> {
    try {
        const result = await StatsService.GetServerStats(window);
        return (result as unknown as ServerStat[]) || [];
    } catch (e) {
        console.error('getServerStats failed', e);
        return [];
    }
}

export async function getToolStats(serverID: string, window: StatsWindow): Promise<ToolStat[]> {
    try {
        const result = await StatsService.GetToolStats(serverID, window);
        return (result as unknown as ToolStat[]) || [];
    } catch (e) {
        console.error('getToolStats failed', e);
        return [];
    }
}

export async function getActivityBuckets(serverID: string, window: StatsWindow): Promise<ActivityBucket[]> {
    try {
        const result = await StatsService.GetActivityBuckets(serverID, window);
        return (result as unknown as ActivityBucket[]) || [];
    } catch (e) {
        console.error('getActivityBuckets failed', e);
        return [];
    }
}

export async function getAppInfo(): Promise<{ name: string; version: string }> {
    try {
        const result = await SettingsService.GetAppInfo();
        return result as unknown as { name: string; version: string };
    } catch {
        return { name: 'MCP Overwatch', version: 'dev' };
    }
}
