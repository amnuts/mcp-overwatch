import type { InstalledServer } from '../../types';
import { toggleServer } from '../../services/api';
import { useApp } from '../../context/AppContext';

const statusColors: Record<string, string> = {
    running: 'bg-emerald-400 shadow-[0_0_6px] shadow-emerald-400/60',
    stopped: 'bg-[#64748b]',
    error: 'bg-[#f87171] shadow-[0_0_6px] shadow-red-400/60',
    starting: 'bg-[#fbbf24] animate-pulse',
};

const registryBadge: Record<string, string> = {
    npm: 'bg-emerald-500/15 text-emerald-400',
    pypi: 'bg-blue-500/15 text-blue-400',
    custom: 'bg-slate-500/15 text-[#94a3b8]',
};

const transportBadge: Record<string, string> = {
    stdio: 'bg-slate-500/15 text-[#94a3b8]',
    sse: 'bg-purple-500/15 text-purple-400',
};

interface Props {
    server: InstalledServer;
    selected: boolean;
    onSelect: () => void;
}

export default function ServerCard({ server, selected, onSelect }: Props) {
    const { refreshInstalled } = useApp();

    const toolCount = (() => {
        try {
            const tools = JSON.parse(server.cached_tools_json || '[]');
            return Array.isArray(tools) ? tools.length : 0;
        } catch {
            return 0;
        }
    })();

    const handleToggle = async (e: React.MouseEvent) => {
        e.stopPropagation();
        await toggleServer(server.id, !server.is_active);
        await refreshInstalled();
    };

    return (
        <div
            onClick={onSelect}
            className={`p-5 rounded-xl border cursor-pointer transition-all ${
                selected
                    ? 'border-indigo-500/50 bg-indigo-500/[0.08] shadow-[0_0_15px] shadow-indigo-500/10'
                    : 'border-white/[0.06] bg-[#1a1f2e] hover:bg-[#222838] hover:border-white/[0.12]'
            }`}
        >
            <div className="flex items-start justify-between gap-4">
                <div className="flex-1 min-w-0">
                    <div className="flex items-center gap-2.5 mb-1.5">
                        <span className={`inline-block w-2.5 h-2.5 rounded-full flex-shrink-0 ${statusColors[server.status] ?? 'bg-[#64748b]'}`} />
                        <h3 className="text-sm font-semibold text-[#e2e8f0] truncate">{server.display_name}</h3>
                    </div>
                    <p className="text-sm text-[#94a3b8] line-clamp-2 mb-2.5">{server.description || 'No description'}</p>
                    <div className="flex items-center gap-2 text-xs text-[#64748b]">
                        <span className={`px-2 py-0.5 rounded-full ${registryBadge[server.registry_type] || registryBadge.custom}`}>
                            {server.registry_type || 'custom'}
                        </span>
                        <span className={`px-2 py-0.5 rounded-full ${transportBadge[server.transport_type] || transportBadge.stdio}`}>
                            {server.transport_type || 'stdio'}
                        </span>
                        {toolCount > 0 && <span>{toolCount} tools</span>}
                        {server.version && <span>v{server.version}</span>}
                    </div>
                </div>
                <button
                    onClick={handleToggle}
                    className={`relative w-10 h-6 rounded-full transition-colors flex-shrink-0 mt-0.5 ${
                        server.is_active ? 'bg-[#6366f1]' : 'bg-slate-600'
                    }`}
                    aria-label={server.is_active ? 'Disable server' : 'Enable server'}
                >
                    <span
                        className={`absolute left-0 top-0.5 w-5 h-5 rounded-full bg-white shadow transition-transform ${
                            server.is_active ? 'translate-x-[18px]' : 'translate-x-0.5'
                        }`}
                    />
                </button>
            </div>
        </div>
    );
}
