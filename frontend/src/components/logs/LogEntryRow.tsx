import type { CSSProperties } from 'react';
import type { LogEntry } from '../../types';

interface Props {
    entry: LogEntry;
    style: CSSProperties;
    expanded: boolean;
    onToggle: () => void;
}

const directionIcons: Record<string, string> = {
    in: '\u2190',   // left arrow
    out: '\u2192',  // right arrow
};

export default function LogEntryRow({ entry, style, expanded, onToggle }: Props) {
    const ts = new Date(entry.timestamp);
    const timeStr = ts.toLocaleTimeString(undefined, { hour12: false, hour: '2-digit', minute: '2-digit', second: '2-digit' });

    return (
        <div style={style} className="border-b border-gray-800/50">
            <div
                onClick={entry.raw ? onToggle : undefined}
                className={`flex items-center gap-2 px-4 py-1 text-xs ${entry.raw ? 'cursor-pointer hover:bg-gray-800/50' : ''}`}
            >
                <span className="text-gray-500 font-mono w-16 flex-shrink-0">{timeStr}</span>
                <span className="text-indigo-400 font-medium w-24 truncate flex-shrink-0">
                    {entry.server_id ? entry.server_id.substring(0, 8) : 'system'}
                </span>
                <span className="text-gray-500 w-4 flex-shrink-0 text-center">
                    {directionIcons[entry.direction] ?? '\u00b7'}
                </span>
                <span className="text-gray-300 flex-1 overflow-hidden break-all whitespace-normal line-clamp-2">{entry.summary}</span>
            </div>
            {expanded && entry.raw && (
                <div className="px-4 pb-2">
                    <pre className="text-[11px] text-gray-400 bg-gray-900 rounded p-2 overflow-x-auto font-mono whitespace-pre-wrap">
                        {(() => {
                            try {
                                return JSON.stringify(JSON.parse(entry.raw), null, 2);
                            } catch {
                                return entry.raw;
                            }
                        })()}
                    </pre>
                </div>
            )}
        </div>
    );
}
