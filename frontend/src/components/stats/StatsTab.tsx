import { useState, useEffect, useMemo, useCallback } from 'react';
import { useApp } from '../../context/AppContext';
import {
    getStatsSummary,
    getServerStats,
    getToolStats,
    getActivityBuckets,
} from '../../services/api';
import type {
    StatsSummary,
    ServerStat,
    ToolStat,
    ActivityBucket,
    StatsWindow,
} from '../../types';

const WINDOWS: { id: StatsWindow; label: string }[] = [
    { id: '24h', label: 'Last 24h' },
    { id: '7d', label: '7 days' },
    { id: '30d', label: '30 days' },
    { id: 'all', label: 'All time' },
];

function fmtNumber(n: number): string {
    if (n >= 1_000_000) return (n / 1_000_000).toFixed(1) + 'M';
    if (n >= 1_000) return (n / 1_000).toFixed(1) + 'k';
    return n.toLocaleString();
}

function fmtLatency(ms: number): string {
    if (!ms) return '—';
    if (ms < 1) return '<1ms';
    if (ms < 1000) return `${Math.round(ms)}ms`;
    return `${(ms / 1000).toFixed(2)}s`;
}

function fmtRate(r: number): string {
    return `${(r * 100).toFixed(1)}%`;
}

function fmtRelative(ts: string): string {
    if (!ts) return '—';
    const date = new Date(ts);
    const diffMs = Date.now() - date.getTime();
    if (diffMs < 0) return 'just now';
    const sec = Math.floor(diffMs / 1000);
    if (sec < 60) return `${sec}s ago`;
    const min = Math.floor(sec / 60);
    if (min < 60) return `${min}m ago`;
    const hr = Math.floor(min / 60);
    if (hr < 24) return `${hr}h ago`;
    const day = Math.floor(hr / 24);
    return `${day}d ago`;
}

function Tile({ label, value, hint, tone }: { label: string; value: string; hint?: string; tone?: 'normal' | 'error' | 'good' }) {
    const valueColor =
        tone === 'error' ? 'text-rose-300' : tone === 'good' ? 'text-emerald-300' : 'text-white';
    return (
        <div className="bg-[#161b27] border border-white/8 rounded-lg px-4 py-3 flex-1 min-w-[140px]">
            <div className="text-xs text-[#94a3b8] uppercase tracking-wide">{label}</div>
            <div className={`text-2xl font-semibold mt-1 ${valueColor}`}>{value}</div>
            {hint && <div className="text-xs text-[#64748b] mt-1">{hint}</div>}
        </div>
    );
}

function fmtBucketWidth(seconds: number): string {
    if (seconds >= 7 * 86400) return `${seconds / (7 * 86400)}w`;
    if (seconds >= 86400) return `${seconds / 86400}d`;
    if (seconds >= 3600) return `${seconds / 3600}h`;
    return `${seconds}s`;
}

function ActivityChart({ buckets }: { buckets: ActivityBucket[] }) {
    const max = Math.max(1, ...buckets.map((b) => b.calls));
    if (buckets.length === 0) {
        return (
            <div className="text-sm text-[#64748b] py-4 text-center">No activity in this window</div>
        );
    }
    const barWidth = `${100 / buckets.length}%`;
    const granularity = buckets[0].bucket_seconds;
    return (
        <div>
            <div className="flex items-end h-32 gap-px bg-[#0f141d] border border-white/8 rounded-lg p-2">
                {buckets.map((b) => {
                    const callPct = (b.calls / max) * 100;
                    const errorPct = b.calls > 0 ? (b.errors / b.calls) * callPct : 0;
                    const successPct = callPct - errorPct;
                    const tooltip = `${new Date(b.bucket_start).toLocaleString()} (${fmtBucketWidth(b.bucket_seconds)}) — ${b.calls} call(s), ${b.errors} error(s)`;
                    return (
                        <div
                            key={b.bucket_start}
                            style={{ width: barWidth }}
                            className="flex flex-col justify-end h-full"
                            title={tooltip}
                        >
                            {b.calls > 0 ? (
                                <>
                                    <div style={{ height: `${errorPct}%` }} className="bg-rose-500/70" />
                                    <div style={{ height: `${successPct}%` }} className="bg-indigo-500/70" />
                                </>
                            ) : (
                                <div style={{ height: '2px' }} className="bg-white/5" />
                            )}
                        </div>
                    );
                })}
            </div>
            <div className="flex justify-between text-xs text-[#64748b] mt-1 px-2">
                <span>{new Date(buckets[0].bucket_start).toLocaleString()}</span>
                <span>{fmtBucketWidth(granularity)} buckets</span>
                <span>{new Date(buckets[buckets.length - 1].bucket_start).toLocaleString()}</span>
            </div>
        </div>
    );
}

export default function StatsTab() {
    const { state } = useApp();
    const [window, setWindow] = useState<StatsWindow>('24h');
    const [summary, setSummary] = useState<StatsSummary | null>(null);
    const [serverStats, setServerStats] = useState<ServerStat[]>([]);
    const [selectedServer, setSelectedServer] = useState<string | null>(null);
    const [toolStats, setToolStats] = useState<ToolStat[]>([]);
    const [buckets, setBuckets] = useState<ActivityBucket[]>([]);
    const [loading, setLoading] = useState(false);

    const serverNameById = useMemo(() => {
        const map = new Map<string, string>();
        for (const s of state.installedServers) map.set(s.id, s.display_name);
        return map;
    }, [state.installedServers]);

    const reload = useCallback(async () => {
        setLoading(true);
        const [sum, servers] = await Promise.all([
            getStatsSummary(window),
            getServerStats(window),
        ]);
        setSummary(sum);
        setServerStats(servers);
        setLoading(false);
    }, [window]);

    useEffect(() => {
        reload();
    }, [reload]);

    useEffect(() => {
        if (!selectedServer) {
            setToolStats([]);
            setBuckets([]);
            return;
        }
        let cancelled = false;
        (async () => {
            const [tools, hbuckets] = await Promise.all([
                getToolStats(selectedServer, window),
                getActivityBuckets(selectedServer, window),
            ]);
            if (cancelled) return;
            setToolStats(tools);
            setBuckets(hbuckets);
        })();
        return () => {
            cancelled = true;
        };
    }, [selectedServer, window]);

    return (
        <div className="flex flex-col h-full overflow-y-auto">
            <div className="flex items-center justify-between p-5 bg-slate-700 border-b border-white/8 shrink-0">
                <div className="flex items-center gap-2">
                    {WINDOWS.map((w) => (
                        <button
                            key={w.id}
                            onClick={() => setWindow(w.id)}
                            className={`px-3 py-1.5 text-sm rounded-lg transition-colors ${
                                window === w.id
                                    ? 'bg-indigo-500/20 border border-indigo-500/40 text-indigo-200'
                                    : 'bg-slate-600 hover:bg-slate-500 text-slate-200 border border-white/6'
                            }`}
                        >
                            {w.label}
                        </button>
                    ))}
                </div>
                <button
                    onClick={reload}
                    disabled={loading}
                    className="px-3 py-1.5 text-sm bg-slate-600 hover:bg-cyan-600 text-slate-200 border border-white/6 rounded-lg transition-colors disabled:opacity-50"
                >
                    {loading ? 'Loading...' : 'Refresh'}
                </button>
            </div>

            <div className="p-5 space-y-6">
                <section>
                    <h3 className="text-sm font-semibold text-[#94a3b8] uppercase tracking-wide mb-3">Summary</h3>
                    <div className="flex flex-wrap gap-3">
                        <Tile
                            label="Tool calls"
                            value={summary ? fmtNumber(summary.total_calls) : '—'}
                        />
                        <Tile
                            label="Errors"
                            value={summary ? fmtNumber(summary.total_errors) : '—'}
                            tone={summary && summary.total_errors > 0 ? 'error' : 'normal'}
                        />
                        <Tile
                            label="Error rate"
                            value={summary ? fmtRate(summary.error_rate) : '—'}
                            tone={summary && summary.error_rate >= 0.05 ? 'error' : 'normal'}
                        />
                        <Tile
                            label="Avg latency"
                            value={summary ? fmtLatency(summary.avg_latency_ms) : '—'}
                        />
                        <Tile
                            label="Active servers"
                            value={summary ? String(summary.unique_servers) : '—'}
                        />
                        <Tile
                            label="Tools used"
                            value={summary ? String(summary.unique_tools) : '—'}
                        />
                    </div>
                </section>

                <section>
                    <h3 className="text-sm font-semibold text-[#94a3b8] uppercase tracking-wide mb-3">Per-server</h3>
                    {serverStats.length === 0 ? (
                        <div className="bg-[#161b27] border border-white/8 rounded-lg p-6 text-center text-sm text-[#64748b]">
                            No tool calls recorded in this window
                        </div>
                    ) : (
                        <div className="bg-[#161b27] border border-white/8 rounded-lg overflow-hidden">
                            <table className="w-full text-sm">
                                <thead>
                                    <tr className="border-b border-white/8 text-xs uppercase tracking-wide text-[#94a3b8]">
                                        <th className="text-left px-4 py-2 font-medium">Server</th>
                                        <th className="text-right px-4 py-2 font-medium">Calls</th>
                                        <th className="text-right px-4 py-2 font-medium">Errors</th>
                                        <th className="text-right px-4 py-2 font-medium">Error rate</th>
                                        <th className="text-right px-4 py-2 font-medium">Avg</th>
                                        <th className="text-right px-4 py-2 font-medium">p95</th>
                                        <th className="text-right px-4 py-2 font-medium">Max</th>
                                        <th className="text-right px-4 py-2 font-medium">Last</th>
                                    </tr>
                                </thead>
                                <tbody>
                                    {serverStats.map((s) => {
                                        const isSelected = selectedServer === s.server_id;
                                        return (
                                            <tr
                                                key={s.server_id}
                                                onClick={() => setSelectedServer(isSelected ? null : s.server_id)}
                                                className={`border-b border-white/4 cursor-pointer transition-colors ${
                                                    isSelected ? 'bg-indigo-500/10' : 'hover:bg-white/3'
                                                }`}
                                            >
                                                <td className="px-4 py-2 text-[#e2e8f0]">
                                                    {serverNameById.get(s.server_id) || s.server_id}
                                                </td>
                                                <td className="px-4 py-2 text-right text-[#e2e8f0] font-mono">
                                                    {fmtNumber(s.calls)}
                                                </td>
                                                <td className={`px-4 py-2 text-right font-mono ${s.errors > 0 ? 'text-rose-300' : 'text-[#64748b]'}`}>
                                                    {fmtNumber(s.errors)}
                                                </td>
                                                <td className={`px-4 py-2 text-right font-mono ${s.error_rate >= 0.05 ? 'text-rose-300' : 'text-[#64748b]'}`}>
                                                    {fmtRate(s.error_rate)}
                                                </td>
                                                <td className="px-4 py-2 text-right text-[#e2e8f0] font-mono">
                                                    {fmtLatency(s.avg_latency_ms)}
                                                </td>
                                                <td className="px-4 py-2 text-right text-[#e2e8f0] font-mono">
                                                    {fmtLatency(s.p95_latency_ms)}
                                                </td>
                                                <td className="px-4 py-2 text-right text-[#94a3b8] font-mono">
                                                    {fmtLatency(s.max_latency_ms)}
                                                </td>
                                                <td className="px-4 py-2 text-right text-[#94a3b8]">
                                                    {fmtRelative(s.last_activity)}
                                                </td>
                                            </tr>
                                        );
                                    })}
                                </tbody>
                            </table>
                        </div>
                    )}
                </section>

                {selectedServer && (
                    <>
                        <section>
                            <h3 className="text-sm font-semibold text-[#94a3b8] uppercase tracking-wide mb-3">
                                {serverNameById.get(selectedServer) || selectedServer} — hourly activity
                            </h3>
                            <ActivityChart buckets={buckets} />
                        </section>

                        <section>
                            <h3 className="text-sm font-semibold text-[#94a3b8] uppercase tracking-wide mb-3">
                                {serverNameById.get(selectedServer) || selectedServer} — tool breakdown
                            </h3>
                            {toolStats.length === 0 ? (
                                <div className="bg-[#161b27] border border-white/8 rounded-lg p-6 text-center text-sm text-[#64748b]">
                                    No tool data
                                </div>
                            ) : (
                                <div className="bg-[#161b27] border border-white/8 rounded-lg overflow-hidden">
                                    <table className="w-full text-sm">
                                        <thead>
                                            <tr className="border-b border-white/8 text-xs uppercase tracking-wide text-[#94a3b8]">
                                                <th className="text-left px-4 py-2 font-medium">Tool</th>
                                                <th className="text-right px-4 py-2 font-medium">Calls</th>
                                                <th className="text-right px-4 py-2 font-medium">Errors</th>
                                                <th className="text-right px-4 py-2 font-medium">Avg</th>
                                                <th className="text-right px-4 py-2 font-medium">Max</th>
                                                <th className="text-right px-4 py-2 font-medium">Last</th>
                                            </tr>
                                        </thead>
                                        <tbody>
                                            {toolStats.map((t) => (
                                                <tr key={t.tool_name} className="border-b border-white/4 hover:bg-white/3">
                                                    <td className="px-4 py-2 text-[#e2e8f0] font-mono">
                                                        {t.tool_name || '(unnamed)'}
                                                    </td>
                                                    <td className="px-4 py-2 text-right text-[#e2e8f0] font-mono">
                                                        {fmtNumber(t.calls)}
                                                    </td>
                                                    <td className={`px-4 py-2 text-right font-mono ${t.errors > 0 ? 'text-rose-300' : 'text-[#64748b]'}`}>
                                                        {fmtNumber(t.errors)}
                                                    </td>
                                                    <td className="px-4 py-2 text-right text-[#e2e8f0] font-mono">
                                                        {fmtLatency(t.avg_latency_ms)}
                                                    </td>
                                                    <td className="px-4 py-2 text-right text-[#94a3b8] font-mono">
                                                        {fmtLatency(t.max_latency_ms)}
                                                    </td>
                                                    <td className="px-4 py-2 text-right text-[#94a3b8]">
                                                        {fmtRelative(t.last_call)}
                                                    </td>
                                                </tr>
                                            ))}
                                        </tbody>
                                    </table>
                                </div>
                            )}
                        </section>
                    </>
                )}
            </div>
        </div>
    );
}
