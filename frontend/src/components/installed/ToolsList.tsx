import { useState } from 'react';

interface Tool {
    name: string;
    description?: string;
    inputSchema?: unknown;
}

interface Props {
    toolsJson: string;
}

export default function ToolsList({ toolsJson }: Props) {
    const [expanded, setExpanded] = useState(false);

    const tools: Tool[] = (() => {
        try {
            const parsed = JSON.parse(toolsJson || '[]');
            return Array.isArray(parsed) ? parsed : [];
        } catch {
            return [];
        }
    })();

    if (tools.length === 0) {
        return <p className="text-sm text-[#64748b] italic">No tools discovered yet. Start the server to discover tools.</p>;
    }

    return (
        <div>
            <button
                onClick={() => setExpanded(!expanded)}
                className="flex items-center gap-2 text-sm font-medium text-[#94a3b8] hover:text-[#e2e8f0] transition-colors"
            >
                <svg
                    width="12"
                    height="12"
                    viewBox="0 0 12 12"
                    fill="currentColor"
                    className={`transition-transform ${expanded ? 'rotate-90' : ''}`}
                >
                    <path d="M4 2l4 4-4 4z" />
                </svg>
                {tools.length} Tool{tools.length !== 1 ? 's' : ''}
            </button>
            {expanded && (
                <div className="mt-3 space-y-2 pl-5 border-l-2 border-indigo-500/20">
                    {tools.map((tool) => (
                        <div key={tool.name} className="text-sm">
                            <span className="font-mono text-indigo-400">{tool.name}</span>
                            {tool.description && (
                                <span className="text-[#94a3b8] ml-2">- {tool.description}</span>
                            )}
                        </div>
                    ))}
                </div>
            )}
        </div>
    );
}
