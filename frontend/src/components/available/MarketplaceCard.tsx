import type { RegistryServer } from '../../types';

interface Props {
    server: RegistryServer;
    dockerAvailable: boolean;
    onInstall: () => void;
}

const typeColors: Record<string, string> = {
    npm: 'bg-emerald-500/15 text-emerald-400 border-emerald-500/20',
    pip: 'bg-blue-500/15 text-blue-400 border-blue-500/20',
    pypi: 'bg-blue-500/15 text-blue-400 border-blue-500/20',
    oci: 'bg-purple-500/15 text-purple-400 border-purple-500/20',
};

export default function MarketplaceCard({ server, dockerAvailable, onInstall }: Props) {
    const needsDocker = server.registry_type === 'oci';
    const dockerMissing = needsDocker && !dockerAvailable;

    return (
        <div className="p-5 rounded-xl border border-gray-600 bg-gray-800 hover:border-slate-500 hover:bg-slate-800 transition-all">
            <div className="flex items-start justify-between gap-4">
                <div className="flex-1 min-w-0">
                    <h3 className="text-sm font-semibold text-slate-200 truncate">{server.display_name}</h3>
                    <p className="text-sm text-slate-400 line-clamp-2 mt-1 mb-3">{server.description || 'No description'}</p>
                    <div className="flex items-center gap-2 flex-wrap text-xs">
                        {server.registry_type !== '' && (
                            <span className={`px-2.5 py-0.5 rounded-full border ${typeColors[server.registry_type] ?? 'bg-white/6 text-slate-400 border-white/6'}`}>
                                {server.registry_type}
                            </span>
                        )}
                        <span className="px-2.5 py-0.5 bg-white/6 text-slate-400 rounded-full">{server.transport_type || 'stdio'}</span>
                        {server.version && <span className="text-slate-500">v{server.version}</span>}
                        {dockerMissing && (
                            <span className="px-2.5 py-0.5 rounded-full border bg-amber-500/15 text-amber-400 border-amber-500/20">
                                Requires Docker
                            </span>
                        )}
                    </div>
                </div>
                <button
                    onClick={onInstall}
                    className="shrink-0 px-5 py-2 text-sm font-medium bg-slate-600 hover:bg-cyan-600 text-white rounded-lg transition-colors"
                >
                    Install
                </button>
            </div>
        </div>
    );
}
