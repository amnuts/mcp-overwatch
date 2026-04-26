import { useState, useEffect, useRef, useCallback } from 'react';
import type { RegistryServer } from '../../types';
import { useApp } from '../../context/AppContext';
import { listAvailable, syncRegistry } from '../../services/api';
import MarketplaceCard from './MarketplaceCard';
import InstallModal from './InstallModal';

const PAGE_SIZE = 50;

const filterChips: { label: string; field: 'registryType' | 'transportType'; value: string }[] = [
    { label: 'npm', field: 'registryType', value: 'npm' },
    { label: 'pypi', field: 'registryType', value: 'pypi' },
    { label: 'stdio', field: 'transportType', value: 'stdio' },
    { label: 'sse', field: 'transportType', value: 'sse' },
];

export default function MarketplaceTab() {
    const { state } = useApp();
    const [servers, setServers] = useState<RegistryServer[]>([]);
    const [total, setTotal] = useState(0);
    const [search, setSearch] = useState('');
    const [registryType, setRegistryType] = useState('');
    const [transportType, setTransportType] = useState('');
    const [offset, setOffset] = useState(0);
    const [loading, setLoading] = useState(false);
    const [syncing, setSyncing] = useState(false);
    const [syncError, setSyncError] = useState<string | null>(null);
    const [installTarget, setInstallTarget] = useState<RegistryServer | null>(null);

    const debounceRef = useRef<ReturnType<typeof setTimeout>>();

    const fetchServers = useCallback(async (s: string, rt: string, tt: string, off: number) => {
        setLoading(true);
        const result = await listAvailable(s, rt, tt, off, PAGE_SIZE);
        setServers(result.servers);
        setTotal(result.total);
        setLoading(false);
    }, []);

    useEffect(() => {
        clearTimeout(debounceRef.current);
        debounceRef.current = setTimeout(() => {
            setOffset(0);
            fetchServers(search, registryType, transportType, 0);
        }, 300);
        return () => clearTimeout(debounceRef.current);
    }, [search, registryType, transportType, fetchServers]);

    useEffect(() => {
        fetchServers(search, registryType, transportType, offset);
    }, [offset]); // eslint-disable-line react-hooks/exhaustive-deps

    const handleSync = async () => {
        setSyncing(true);
        setSyncError(null);
        try {
            await syncRegistry();
            await fetchServers(search, registryType, transportType, 0);
            setOffset(0);
        } catch (e) {
            setSyncError(e instanceof Error ? e.message : 'Sync failed. Check your network connection and try again.');
        } finally {
            setSyncing(false);
        }
    };

    const toggleFilter = (field: 'registryType' | 'transportType', value: string) => {
        if (field === 'registryType') {
            setRegistryType((v) => (v === value ? '' : value));
        } else {
            setTransportType((v) => (v === value ? '' : value));
        }
    };

    const totalPages = Math.ceil(total / PAGE_SIZE);
    const currentPage = Math.floor(offset / PAGE_SIZE) + 1;

    return (
        <div className="h-full flex flex-col">
            <div className="p-5 space-y-3 border-b border-white/6 shrink-0 overflow-hidden">
                <div className="flex gap-3">
                    <div className="relative flex-1 min-w-0">
                        <svg
                            width="16"
                            height="16"
                            viewBox="0 0 24 24"
                            fill="none"
                            stroke="currentColor"
                            strokeWidth="2"
                            className="absolute left-3 top-1/2 -translate-y-1/2 text-slate-500"
                        >
                            <circle cx="11" cy="11" r="8" />
                            <line x1="21" y1="21" x2="16.65" y2="16.65" />
                        </svg>
                        <input
                            type="text"
                            value={search}
                            onChange={(e) => setSearch(e.target.value)}
                            placeholder="Search servers..."
                            className="w-full pl-10 pr-4 py-2.5 text-sm bg-gray-900 border border-white/8 rounded-lg text-slate-200 placeholder-slate-500 focus:outline-none focus:border-[#6366f1] focus:ring-1 focus:ring-[#6366f1]/30"
                        />
                    </div>
                    <button
                        onClick={handleSync}
                        disabled={syncing}
                        className="px-5 py-2.5 text-sm font-medium bg-slate-600 hover:bg-cyan-600 text-white rounded-lg transition-colors disabled:opacity-50 whitespace-nowrap shrink-0"
                    >
                        {syncing ? 'Syncing...' : 'Sync Registry'}
                    </button>
                </div>
                <div className="flex gap-2">
                    {filterChips.map((chip) => {
                        const active =
                            (chip.field === 'registryType' && registryType === chip.value) ||
                            (chip.field === 'transportType' && transportType === chip.value);
                        return (
                            <button
                                key={chip.value}
                                onClick={() => toggleFilter(chip.field, chip.value)}
                                className={`px-3.5 py-1.5 text-sm rounded-full border transition-colors ${
                                    active
                                        ? 'bg-indigo-500/15 border-indigo-500/40 text-indigo-300'
                                        : 'bg-[#1a1f2e] border-white/6 text-[#94a3b8] hover:border-white/12 hover:text-[#e2e8f0]'
                                }`}
                            >
                                {chip.label}
                            </button>
                        );
                    })}
                </div>
            </div>

            {syncError && (
                <div className="mx-5 mt-3 px-4 py-3 bg-red-500/10 border border-red-500/30 rounded-lg flex items-center justify-between shrink-0">
                    <span className="text-sm text-[#f87171]">{syncError}</span>
                    <button
                        onClick={handleSync}
                        className="text-sm text-[#f87171] hover:text-red-200 underline ml-4 whitespace-nowrap"
                    >
                        Retry
                    </button>
                </div>
            )}

            <div className="flex-1 min-h-0 overflow-y-auto p-5">
                {loading ? (
                    <div className="flex items-center justify-center h-32 text-[#94a3b8] text-sm">Loading...</div>
                ) : servers.length === 0 ? (
                    <div className="flex flex-col items-center justify-center h-32 text-[#94a3b8]">
                        <p className="text-sm">No servers found</p>
                        <p className="text-xs mt-1 text-[#64748b]">Try adjusting your search or sync the registry.</p>
                    </div>
                ) : (
                    <div className="space-y-3">
                        {servers.map((server) => (
                            <MarketplaceCard
                                key={server.id}
                                server={server}
                                dockerAvailable={state.dockerAvailable}
                                onInstall={() => setInstallTarget(server)}
                            />
                        ))}
                    </div>
                )}
            </div>

            {totalPages > 1 && (
                <div className="flex items-center justify-between px-5 py-3 bg-[#141820] border-t border-white/6 text-sm text-[#94a3b8] shrink-0">
                    <span>
                        {total} result{total !== 1 ? 's' : ''}
                    </span>
                    <div className="flex items-center gap-2">
                        <button
                            onClick={() => setOffset(Math.max(0, offset - PAGE_SIZE))}
                            disabled={offset === 0}
                            className="px-3 py-1.5 bg-[#1a1f2e] border border-white/6 rounded-lg hover:bg-[#222838] hover:border-white/12 text-[#e2e8f0] disabled:opacity-30 transition-colors"
                        >
                            Prev
                        </button>
                        <span className="text-[#e2e8f0]">
                            {currentPage} / {totalPages}
                        </span>
                        <button
                            onClick={() => setOffset(offset + PAGE_SIZE)}
                            disabled={currentPage >= totalPages}
                            className="px-3 py-1.5 bg-[#1a1f2e] border border-white/6 rounded-lg hover:bg-[#222838] hover:border-white/12 text-[#e2e8f0] disabled:opacity-30 transition-colors"
                        >
                            Next
                        </button>
                    </div>
                </div>
            )}

            {installTarget && <InstallModal server={installTarget} onClose={() => setInstallTarget(null)} />}
        </div>
    );
}
