import { useState, useEffect } from 'react';
import type { RuntimeInfo } from '../../types';
import { getRuntimes, downloadRuntime, deleteRuntime } from '../../services/api';

function formatSize(bytes: number): string {
    if (bytes === 0) return '';
    if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(0)} KB`;
    if (bytes < 1024 * 1024 * 1024) return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
    return `${(bytes / (1024 * 1024 * 1024)).toFixed(2)} GB`;
}

export default function RuntimeManager() {
    const [runtimes, setRuntimes] = useState<RuntimeInfo[]>([]);
    const [downloading, setDownloading] = useState<string | null>(null);
    const [progress, setProgress] = useState(0);

    useEffect(() => {
        getRuntimes().then(setRuntimes);
    }, []);

    const handleDownload = async (id: string) => {
        setDownloading(id);
        setProgress(0);
        const timer = setInterval(() => {
            setProgress((p) => Math.min(p + 20, 90));
        }, 500);

        try {
            await downloadRuntime(id);
            setProgress(100);
            const updated = await getRuntimes();
            setRuntimes(updated);
        } finally {
            clearInterval(timer);
            setDownloading(null);
            setProgress(0);
        }
    };

    const handleDelete = async (id: string) => {
        if (!confirm('Remove this runtime? Servers that depend on it will not start until it is re-downloaded.')) return;
        await deleteRuntime(id);
        const updated = await getRuntimes();
        setRuntimes(updated);
    };

    const defaultRuntimes = [
        { id: 'node', label: 'Node.js', description: 'Required for npm-based MCP servers' },
        { id: 'python', label: 'Python', description: 'Required for pip/pypi-based MCP servers' },
    ];

    return (
        <div className="space-y-4">
            <p className="text-sm text-[#94a3b8]">
                Manage portable runtime installations used to run MCP servers.
            </p>
            {defaultRuntimes.map((rt) => {
                const installed = runtimes.find((r) => r.id === rt.id);
                const isDownloading = downloading === rt.id;

                return (
                    <div key={rt.id} className="p-4 bg-[#1a1f2e] rounded-xl border border-white/[0.06]">
                        {installed ? (
                            <div className="space-y-3">
                                <div className="flex items-start justify-between">
                                    <div>
                                        <div className="flex items-center gap-2">
                                            <span className="text-sm font-medium text-[#e2e8f0]">{rt.label}</span>
                                            <span className="px-2 py-0.5 text-xs bg-emerald-500/15 text-emerald-400 rounded-full">
                                                {installed.version}
                                            </span>
                                        </div>
                                        <p className="text-xs text-[#94a3b8] mt-1">{rt.description}</p>
                                    </div>
                                </div>
                                <div className="grid grid-cols-2 gap-2 text-xs">
                                    <div className="p-2 rounded-lg bg-[#161b27]">
                                        <span className="text-[#64748b]">Installed</span>
                                        <p className="text-[#94a3b8] mt-0.5">{new Date(installed.installed_at).toLocaleDateString()}</p>
                                    </div>
                                    {installed.size_bytes > 0 && (
                                        <div className="p-2 rounded-lg bg-[#161b27]">
                                            <span className="text-[#64748b]">Size</span>
                                            <p className="text-[#94a3b8] mt-0.5">{formatSize(installed.size_bytes)}</p>
                                        </div>
                                    )}
                                </div>
                                <p className="text-xs text-[#64748b] font-mono truncate">{installed.path}</p>
                                <div className="flex items-center gap-2">
                                    {isDownloading ? (
                                        <div className="w-24">
                                            <div className="h-2 bg-[#161b27] rounded-full overflow-hidden">
                                                <div
                                                    className="h-full bg-[#6366f1] rounded-full transition-all duration-300"
                                                    style={{ width: `${progress}%` }}
                                                />
                                            </div>
                                        </div>
                                    ) : (
                                        <>
                                            <button
                                                onClick={() => handleDownload(rt.id)}
                                                className="px-3 py-1.5 text-xs font-medium rounded-lg bg-[#161b27] hover:bg-[#222838] text-[#94a3b8] border border-white/[0.08] transition-colors"
                                            >
                                                Reinstall
                                            </button>
                                            <button
                                                onClick={() => handleDelete(rt.id)}
                                                className="px-3 py-1.5 text-xs font-medium rounded-lg bg-red-500/10 hover:bg-red-500/20 text-[#f87171] border border-red-500/20 transition-colors"
                                            >
                                                Remove
                                            </button>
                                        </>
                                    )}
                                </div>
                            </div>
                        ) : (
                            <div className="flex items-center justify-between">
                                <div>
                                    <span className="text-sm font-medium text-[#e2e8f0]">{rt.label}</span>
                                    <p className="text-xs text-[#94a3b8] mt-1">{rt.description}</p>
                                </div>
                                <div className="flex items-center gap-2">
                                    {isDownloading ? (
                                        <div className="w-24">
                                            <div className="h-2 bg-[#161b27] rounded-full overflow-hidden">
                                                <div
                                                    className="h-full bg-[#6366f1] rounded-full transition-all duration-300"
                                                    style={{ width: `${progress}%` }}
                                                />
                                            </div>
                                        </div>
                                    ) : (
                                        <button
                                            onClick={() => handleDownload(rt.id)}
                                            className="px-4 py-2 text-sm font-medium rounded-lg bg-[#6366f1] hover:bg-[#818cf8] text-white transition-colors"
                                        >
                                            Download
                                        </button>
                                    )}
                                </div>
                            </div>
                        )}
                    </div>
                );
            })}
        </div>
    );
}
