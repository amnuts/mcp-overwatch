import { useState } from 'react';
import type { RegistryServer } from '../../types';
import { installServer } from '../../services/api';
import { useApp } from '../../context/AppContext';

interface Props {
    server: RegistryServer;
    onClose: () => void;
}

export default function InstallModal({ server, onClose }: Props) {
    const { refreshInstalled, dispatch } = useApp();
    const [installing, setInstalling] = useState(false);
    const [progress, setProgress] = useState(0);
    const [error, setError] = useState<string | null>(null);

    const handleInstall = async () => {
        setInstalling(true);
        setError(null);

        const timer = setInterval(() => {
            setProgress((p) => Math.min(p + 15, 90));
        }, 300);

        try {
            await installServer(server.id);
            setProgress(100);
            await refreshInstalled();
            setTimeout(() => {
                dispatch({ type: 'SET_TAB', payload: 'installed' });
                onClose();
            }, 500);
        } catch (e) {
            setError(e instanceof Error ? e.message : 'Installation failed');
        } finally {
            clearInterval(timer);
            setInstalling(false);
        }
    };

    return (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/70 backdrop-blur-sm" onClick={onClose}>
            <div
                onClick={(e) => e.stopPropagation()}
                className="w-full max-w-md mx-4 bg-[#1a1f2e] rounded-xl border border-white/[0.08] shadow-2xl"
            >
                <div className="p-6">
                    <h2 className="text-lg font-semibold text-[#e2e8f0]">Install {server.display_name}</h2>
                    <p className="text-sm text-[#94a3b8] mt-1">{server.description || 'No description'}</p>

                    <div className="mt-5 space-y-2.5 text-sm text-[#e2e8f0]">
                        <div className="flex justify-between">
                            <span className="text-[#94a3b8]">Package</span>
                            <span className="font-mono">{server.package_identifier}</span>
                        </div>
                        <div className="flex justify-between">
                            <span className="text-[#94a3b8]">Type</span>
                            <span>{server.registry_type}</span>
                        </div>
                        <div className="flex justify-between">
                            <span className="text-[#94a3b8]">Version</span>
                            <span>{server.version || 'latest'}</span>
                        </div>
                    </div>

                    {installing && (
                        <div className="mt-5">
                            <div className="h-2 bg-[#161b27] rounded-full overflow-hidden">
                                <div
                                    className="h-full bg-[#6366f1] rounded-full transition-all duration-300"
                                    style={{ width: `${progress}%` }}
                                />
                            </div>
                            <p className="text-xs text-[#94a3b8] mt-1.5">
                                {progress < 100 ? 'Installing...' : 'Done!'}
                            </p>
                        </div>
                    )}

                    {error && (
                        <div className="mt-4 px-3 py-2.5 bg-red-500/10 border border-red-500/30 rounded-lg text-sm text-[#f87171]">
                            {error}
                        </div>
                    )}
                </div>

                <div className="flex justify-end gap-3 px-6 py-4 border-t border-white/[0.06]">
                    <button
                        onClick={onClose}
                        disabled={installing}
                        className="px-5 py-2 text-sm font-medium text-[#94a3b8] hover:text-[#e2e8f0] bg-[#161b27] hover:bg-[#222838] border border-white/[0.08] rounded-lg transition-colors disabled:opacity-50"
                    >
                        Cancel
                    </button>
                    <button
                        onClick={handleInstall}
                        disabled={installing}
                        className="px-5 py-2 text-sm font-medium bg-[#6366f1] hover:bg-[#818cf8] text-white rounded-lg transition-colors disabled:opacity-50"
                    >
                        {installing ? 'Installing...' : 'Install'}
                    </button>
                </div>
            </div>
        </div>
    );
}
