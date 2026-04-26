import { useState, useEffect, useCallback, useRef, type CSSProperties, type ReactElement } from 'react';
import { List, type ListImperativeAPI } from 'react-window';
import { useApp } from '../../context/AppContext';
import type { LogEntry } from '../../types';

const ROW_HEIGHT = 40;

interface RowProps {
    logs: LogEntry[];
    expandedIdx: number | null;
    onToggle: (idx: number) => void;
}

function LogRow({ index, style, logs, expandedIdx, onToggle }: {
    index: number;
    style: CSSProperties;
    ariaAttributes: { 'aria-posinset': number; 'aria-setsize': number; role: 'listitem' };
} & RowProps): ReactElement | null {
    const entry = logs[index];
    if (!entry) return null;

    const ts = new Date(entry.timestamp);
    const timeStr = ts.toLocaleTimeString(undefined, {
        hour12: false,
        hour: '2-digit',
        minute: '2-digit',
        second: '2-digit',
    });

    const directionIcon = entry.direction === 'in' ? '\u2190' : entry.direction === 'out' ? '\u2192' : '\u00b7';
    const isExpanded = expandedIdx === index;

    return (
        <div style={style} className={`border-b border-white/4 ${index % 2 === 0 ? 'bg-transparent' : 'bg-white/2'}`}>
            <div
                onClick={() => entry.raw && onToggle(isExpanded ? -1 : index)}
                className={`flex items-start gap-3 px-4 pt-1.5 h-full text-sm ${entry.raw ? 'cursor-pointer hover:bg-white/5' : ''}`}
            >
                <span className="text-[#64748b] font-mono w-16 shrink-0">{timeStr}</span>
                <span className="text-indigo-300 font-medium w-24 truncate shrink-0">
                    {entry.server_id ? entry.server_id.substring(0, 8) : 'system'}
                </span>
                <span className="text-[#64748b] w-4 shrink-0 text-center">{directionIcon}</span>
                <span className="text-[#e2e8f0] flex-1 overflow-hidden break-all whitespace-normal line-clamp-2">{entry.summary}</span>
            </div>
        </div>
    );
}

export default function LogsTab() {
    const { state } = useApp();
    const [filter, setFilter] = useState('');
    const [paused, setPaused] = useState(false);
    const [expandedIdx, setExpandedIdx] = useState<number | null>(null);
    const listRef = useRef<ListImperativeAPI | null>(null);
    const [containerHeight, setContainerHeight] = useState(400);

    const measuredRef = useCallback((node: HTMLDivElement | null) => {
        if (!node) return;
        setContainerHeight(node.getBoundingClientRect().height);
        const ro = new ResizeObserver((entries) => {
            for (const entry of entries) {
                setContainerHeight(entry.contentRect.height);
            }
        });
        ro.observe(node);
    }, []);

    const filtered = filter
        ? state.logs.filter(
              (l) =>
                  l.summary.toLowerCase().includes(filter.toLowerCase()) ||
                  l.server_id.toLowerCase().includes(filter.toLowerCase()),
          )
        : state.logs;

    useEffect(() => {
        if (!paused && listRef.current && filtered.length > 0) {
            listRef.current.scrollToRow({ index: filtered.length - 1, align: 'end' });
        }
    }, [filtered.length, paused]);

    const handleToggle = useCallback((idx: number) => {
        setExpandedIdx((prev) => (prev === idx ? null : idx));
    }, []);

    const expandedEntry = expandedIdx !== null ? filtered[expandedIdx] : null;

    return (
        <div className="flex flex-col h-full">
            <div className="flex items-center gap-3 p-5 space-y-3 bg-slate-700 border-b border-white/8 shrink-0">
                <div className="relative flex-1">
                    <svg
                        width="16"
                        height="16"
                        viewBox="0 0 24 24"
                        fill="none"
                        stroke="currentColor"
                        strokeWidth="2"
                        className="absolute left-3 top-1/2 -translate-y-1/2 text-[#64748b]"
                    >
                        <circle cx="11" cy="11" r="8" />
                        <line x1="21" y1="21" x2="16.65" y2="16.65" />
                    </svg>
                    <input
                        type="text"
                        value={filter}
                        onChange={(e) => setFilter(e.target.value)}
                        placeholder="Filter logs..."
                        className="w-full pl-10 pr-4 py-2.5 text-sm bg-[#161b27] border border-white/8 rounded-lg text-[#e2e8f0] placeholder-[#64748b] focus:outline-none focus:border-[#6366f1] focus:ring-1 focus:ring-[#6366f1]/30"
                    />
                </div>
                <button
                    onClick={() => setPaused(!paused)}
                    className={`px-4 py-2.5 text-sm font-medium rounded-lg transition-colors disabled:opacity-50 whitespace-nowrap shrink-0 ${
                        paused
                            ? 'bg-amber-500/15 hover:bg-amber-500/45 border border-amber-500/30 text-amber-300'
                            : 'bg-slate-600 hover:bg-cyan-600 text-slate-200 border border-white/6'
                    }`}
                >
                    {paused ? 'Resume' : 'Pause'}
                </button>
                <span className="text-sm text-[#64748b] whitespace-nowrap">{filtered.length} entries</span>
            </div>

            <div ref={measuredRef} className="flex-1 min-h-0 overflow-hidden">
                {filtered.length === 0 ? (
                    <div className="flex items-center justify-center h-full text-[#94a3b8] text-sm">
                        {state.logs.length === 0 ? 'No log entries yet' : 'No matching entries'}
                    </div>
                ) : (
                    <List
                        listRef={listRef}
                        rowCount={filtered.length}
                        rowHeight={ROW_HEIGHT}
                        rowComponent={LogRow}
                        rowProps={{ logs: filtered, expandedIdx, onToggle: handleToggle }}
                        style={{ height: containerHeight }}
                    />
                )}
            </div>

            {expandedEntry?.raw && (
                <div className="border-t border-white/6 max-h-48 overflow-y-auto shrink-0 bg-[#1a1f2e]">
                    <div className="flex items-center justify-between px-4 py-2">
                        <span className="text-sm text-[#e2e8f0] font-medium">Raw Payload</span>
                        <button
                            onClick={() => setExpandedIdx(null)}
                            className="text-sm text-[#94a3b8] hover:text-[#e2e8f0]"
                        >
                            Close
                        </button>
                    </div>
                    <pre className="px-4 py-2 text-xs text-[#94a3b8] font-mono whitespace-pre-wrap">
                        {(() => {
                            try {
                                return JSON.stringify(JSON.parse(expandedEntry.raw!), null, 2);
                            } catch {
                                return expandedEntry.raw;
                            }
                        })()}
                    </pre>
                </div>
            )}
        </div>
    );
}
