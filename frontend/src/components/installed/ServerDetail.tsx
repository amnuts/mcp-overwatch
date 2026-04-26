import type { InstalledServer } from '../../types';
import { useApp } from '../../context/AppContext';
import { uninstallServer, restartServer, updateServer } from '../../services/api';
import ConfigForm from './ConfigForm';
import ToolsList from './ToolsList';

function compareVersions(a: string, b: string): number {
    const pa = a.replace(/^v/, '').split('.').map(Number);
    const pb = b.replace(/^v/, '').split('.').map(Number);
    for (let i = 0; i < Math.max(pa.length, pb.length); i++) {
        const na = pa[i] || 0;
        const nb = pb[i] || 0;
        if (na !== nb) return na - nb;
    }
    return 0;
}

interface Props {
    server: InstalledServer;
}

export default function ServerDetail({ server }: Props) {
    const { dispatch, refreshInstalled } = useApp();

    const handleUninstall = async () => {
        if (!confirm(`Uninstall "${server.display_name}"?`)) return;
        await uninstallServer(server.id);
        dispatch({ type: 'SET_SELECTED_SERVER', payload: null });
        await refreshInstalled();
    };

    const handleRestart = async () => {
        await restartServer(server.id);
        await refreshInstalled();
    };

    const handleUpdate = async () => {
        await updateServer(server.id);
        await refreshInstalled();
    };

    return (
        <div className="h-full overflow-y-auto p-6 space-y-5">
            <div className="bg-[#1a1f2e] rounded-xl border border-white/[0.06] p-5">
                <div className="flex items-start justify-between">
                    <div>
                        <h2 className="text-lg font-semibold text-[#e2e8f0]">{server.display_name}</h2>
                        <p className="text-sm text-[#94a3b8] mt-1">{server.description || 'No description'}</p>
                    </div>
                    <button
                        onClick={() => dispatch({ type: 'SET_SELECTED_SERVER', payload: null })}
                        className="w-8 h-8 flex items-center justify-center rounded-lg text-[#64748b] hover:text-[#e2e8f0] hover:bg-white/10 transition-colors"
                        aria-label="Close detail"
                    >
                        <svg width="14" height="14" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round">
                            <line x1="4" y1="4" x2="12" y2="12" />
                            <line x1="12" y1="4" x2="4" y2="12" />
                        </svg>
                    </button>
                </div>
            </div>

            <div className="bg-[#1a1f2e] rounded-xl border border-white/[0.06] p-5">
                <div className="grid grid-cols-2 gap-3 text-sm">
                    <div className="p-3 rounded-lg bg-[#161b27]">
                        <span className="text-[#64748b] text-xs">Version</span>
                        <p className="text-[#e2e8f0] font-mono mt-0.5">{server.version || '-'}</p>
                    </div>
                    <div className="p-3 rounded-lg bg-[#161b27]">
                        <span className="text-[#64748b] text-xs">Status</span>
                        <p className="text-[#e2e8f0] capitalize mt-0.5">{server.status}</p>
                    </div>
                    <div className="p-3 rounded-lg bg-[#161b27]">
                        <span className="text-[#64748b] text-xs">Transport</span>
                        <p className="text-[#e2e8f0] mt-0.5">{server.transport_type || 'stdio'}</p>
                    </div>
                    <div className="p-3 rounded-lg bg-[#161b27]">
                        <span className="text-[#64748b] text-xs">Package</span>
                        <p className="text-[#e2e8f0] font-mono truncate mt-0.5">{server.package_identifier || '-'}</p>
                    </div>
                    <div className="p-3 rounded-lg bg-[#161b27]">
                        <span className="text-[#64748b] text-xs">Installed</span>
                        <p className="text-[#e2e8f0] mt-0.5">{new Date(server.installed_at).toLocaleDateString()}</p>
                    </div>
                    <div className="p-3 rounded-lg bg-[#161b27]">
                        <span className="text-[#64748b] text-xs">Errors</span>
                        <p className={`mt-0.5 ${server.error_count > 0 ? 'text-[#f87171]' : 'text-[#e2e8f0]'}`}>{server.error_count}</p>
                    </div>
                </div>
            </div>

            {server.available_version && (() => {
                const isUpgrade = compareVersions(server.available_version, server.version) > 0;
                return (
                    <div className={`flex items-center justify-between px-5 py-3 rounded-xl text-sm border ${
                        isUpgrade
                            ? 'bg-indigo-500/[0.08] border-indigo-500/30'
                            : 'bg-amber-500/[0.08] border-amber-500/30'
                    }`}>
                        <span className={isUpgrade ? 'text-indigo-300' : 'text-amber-300'}>
                            {isUpgrade ? 'Upgrade' : 'Downgrade'} available: v{server.available_version}
                        </span>
                        <button
                            onClick={handleUpdate}
                            className={`px-4 py-1.5 text-white rounded-lg transition-colors text-sm font-medium ${
                                isUpgrade
                                    ? 'bg-indigo-600 hover:bg-indigo-500'
                                    : 'bg-amber-600 hover:bg-amber-500'
                            }`}
                        >
                            {isUpgrade ? 'Upgrade' : 'Downgrade'}
                        </button>
                    </div>
                );
            })()}

            <div className="bg-[#1a1f2e] rounded-xl border border-white/[0.06] p-5">
                <h3 className="text-sm font-medium text-[#e2e8f0] mb-3">Configuration</h3>
                <ConfigForm
                    serverId={server.id}
                    envVarsJson={server.env_vars_json}
                    userConfigJson={server.user_config_json}
                />
            </div>

            <div className="bg-[#1a1f2e] rounded-xl border border-white/[0.06] p-5">
                <h3 className="text-sm font-medium text-[#e2e8f0] mb-3">Tools</h3>
                <ToolsList toolsJson={server.cached_tools_json} />
            </div>

            <div className="flex items-center gap-3 pt-4 border-t border-white/[0.06]">
                {server.is_active && (
                    <button
                        onClick={handleRestart}
                        className="px-4 py-2 text-sm font-medium bg-[#1a1f2e] hover:bg-[#222838] text-[#e2e8f0] border border-white/[0.06] hover:border-white/[0.12] rounded-lg transition-colors"
                    >
                        Restart
                    </button>
                )}
                <button
                    onClick={handleUninstall}
                    className="px-4 py-2 text-sm font-medium bg-red-500/10 hover:bg-red-500/20 text-[#f87171] border border-red-500/20 rounded-lg transition-colors"
                >
                    Uninstall
                </button>
            </div>
        </div>
    );
}
