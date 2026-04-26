import { useState, useEffect } from 'react';
import { useApp } from '../../context/AppContext';
import type { AppConfig } from '../../types';
import { getSettings, saveSettings, getAppInfo } from '../../services/api';
import ProxyConfig from './ProxyConfig';
import RuntimeManager from './RuntimeManager';
import ClientIntegration from './ClientIntegration';

type SettingsTab = 'proxy' | 'runtimes' | 'clients' | 'sync' | 'app';

const tabs: { id: SettingsTab; label: string }[] = [
    { id: 'proxy', label: 'Proxy' },
    { id: 'runtimes', label: 'Runtimes' },
    { id: 'clients', label: 'Clients' },
    { id: 'sync', label: 'Sync' },
    { id: 'app', label: 'App' },
];

export default function SettingsModal() {
    const { dispatch } = useApp();
    const [activeTab, setActiveTab] = useState<SettingsTab>('proxy');
    const [config, setConfig] = useState<AppConfig | null>(null);
    const [saving, setSaving] = useState(false);
    const [saved, setSaved] = useState(false);
    const [appInfo, setAppInfo] = useState<{ name: string; version: string } | null>(null);

    useEffect(() => {
        getSettings().then(setConfig);
        getAppInfo().then(setAppInfo);
    }, []);

    const handleSave = async () => {
        if (!config) return;
        setSaving(true);
        await saveSettings(config);
        setSaving(false);
        setSaved(true);
        setTimeout(() => setSaved(false), 2000);
    };

    const handleClose = () => {
        dispatch({ type: 'SET_SETTINGS_OPEN', payload: false });
    };

    if (!config) return null;

    return (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/70 backdrop-blur-sm" onClick={handleClose}>
            <div
                onClick={(e) => e.stopPropagation()}
                className="w-full max-w-2xl mx-4 bg-[#1a1f2e] rounded-xl border border-white/[0.08] shadow-2xl max-h-[80vh] flex flex-col"
            >
                <div className="flex items-center justify-between px-6 py-4 border-b border-white/[0.06]">
                    <h2 className="text-lg font-semibold text-[#e2e8f0]">Settings</h2>
                    <button
                        onClick={handleClose}
                        className="w-8 h-8 flex items-center justify-center rounded-lg text-[#64748b] hover:text-[#e2e8f0] hover:bg-white/[0.08] transition-colors"
                        aria-label="Close"
                    >
                        <svg width="14" height="14" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round">
                            <line x1="4" y1="4" x2="12" y2="12" />
                            <line x1="12" y1="4" x2="4" y2="12" />
                        </svg>
                    </button>
                </div>

                <div className="flex flex-1 overflow-hidden">
                    <div className="w-40 border-r border-white/[0.06] py-3 px-2">
                        {tabs.map((tab) => (
                            <button
                                key={tab.id}
                                onClick={() => setActiveTab(tab.id)}
                                className={`w-full text-left px-4 py-2.5 text-sm rounded-lg transition-colors mb-0.5 ${
                                    activeTab === tab.id
                                        ? 'text-indigo-300 bg-indigo-500/15'
                                        : 'text-[#94a3b8] hover:text-[#e2e8f0] hover:bg-white/[0.05]'
                                }`}
                            >
                                {tab.label}
                            </button>
                        ))}
                    </div>

                    <div className="flex-1 overflow-y-auto p-6">
                        {activeTab === 'proxy' && <ProxyConfig config={config} onChange={setConfig} />}
                        {activeTab === 'runtimes' && <RuntimeManager />}
                        {activeTab === 'clients' && <ClientIntegration />}
                        {activeTab === 'sync' && (
                            <div className="space-y-5">
                                <div>
                                    <label className="block text-sm font-medium text-[#e2e8f0] mb-1">Sync Interval (hours)</label>
                                    <p className="text-xs text-[#94a3b8] mb-2">
                                        How often to check the MCP registry for updates.
                                    </p>
                                    <input
                                        type="number"
                                        value={config.sync.interval_hours}
                                        onChange={(e) =>
                                            setConfig({
                                                ...config,
                                                sync: { ...config.sync, interval_hours: parseInt(e.target.value) || 24 },
                                            })
                                        }
                                        className="w-32 px-3 py-2 text-sm bg-[#161b27] border border-white/[0.08] rounded-lg text-[#e2e8f0] focus:outline-none focus:border-[#6366f1] focus:ring-1 focus:ring-[#6366f1]/30"
                                    />
                                </div>
                                {config.sync.last_sync && (
                                    <div className="text-sm text-[#94a3b8]">
                                        Last synced: {new Date(config.sync.last_sync).toLocaleString()}
                                    </div>
                                )}
                            </div>
                        )}
                        {activeTab === 'app' && (
                            <div className="space-y-5">
                                <label className="flex items-center gap-3 cursor-pointer">
                                    <input
                                        type="checkbox"
                                        checked={config.app.start_on_boot}
                                        onChange={(e) =>
                                            setConfig({
                                                ...config,
                                                app: { ...config.app, start_on_boot: e.target.checked },
                                            })
                                        }
                                        className="w-4 h-4 rounded border-white/[0.08] bg-[#161b27] text-[#6366f1] focus:ring-[#6366f1]/30"
                                    />
                                    <div>
                                        <span className="text-sm text-[#e2e8f0]">Start on boot</span>
                                        <p className="text-xs text-[#94a3b8]">Launch MCP Overwatch when the system starts.</p>
                                    </div>
                                </label>
                                <div>
                                    <label className="block text-sm font-medium text-[#e2e8f0] mb-1">Log Retention (days)</label>
                                    <input
                                        type="number"
                                        value={config.logging.retention_days}
                                        onChange={(e) =>
                                            setConfig({
                                                ...config,
                                                logging: { ...config.logging, retention_days: parseInt(e.target.value) || 7 },
                                            })
                                        }
                                        className="w-32 px-3 py-2 text-sm bg-[#161b27] border border-white/[0.08] rounded-lg text-[#e2e8f0] focus:outline-none focus:border-[#6366f1] focus:ring-1 focus:ring-[#6366f1]/30"
                                    />
                                </div>
                                <div>
                                    <label className="block text-sm font-medium text-[#e2e8f0] mb-1">Stats Retention (days)</label>
                                    <input
                                        type="number"
                                        value={config.stats.retention_days}
                                        onChange={(e) =>
                                            setConfig({
                                                ...config,
                                                stats: { ...config.stats, retention_days: parseInt(e.target.value) || 30 },
                                            })
                                        }
                                        className="w-32 px-3 py-2 text-sm bg-[#161b27] border border-white/[0.08] rounded-lg text-[#e2e8f0] focus:outline-none focus:border-[#6366f1] focus:ring-1 focus:ring-[#6366f1]/30"
                                    />
                                </div>
                                {appInfo && (
                                    <div className="pt-4 mt-4 border-t border-white/[0.06] text-xs text-[#64748b]">
                                        {appInfo.name} · v{appInfo.version}
                                    </div>
                                )}
                            </div>
                        )}
                    </div>
                </div>

                <div className="flex justify-end gap-3 px-6 py-4 border-t border-white/[0.06]">
                    <button
                        onClick={handleClose}
                        className="px-5 py-2 text-sm font-medium text-[#94a3b8] hover:text-[#e2e8f0] bg-[#161b27] hover:bg-[#222838] border border-white/[0.08] rounded-lg transition-colors"
                    >
                        Cancel
                    </button>
                    <button
                        onClick={handleSave}
                        disabled={saving}
                        className="px-5 py-2 text-sm font-medium bg-[#6366f1] hover:bg-[#818cf8] text-white rounded-lg transition-colors disabled:opacity-50"
                    >
                        {saving ? 'Saving...' : saved ? 'Saved!' : 'Save'}
                    </button>
                </div>
            </div>
        </div>
    );
}
