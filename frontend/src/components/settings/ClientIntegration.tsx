import { useState, useEffect } from 'react';
import type { ClientInfo } from '../../types';
import { detectClients, registerClaudeDesktop, registerClaudeCode, getSettings } from '../../services/api';

const configSnippets: Record<string, (port: number) => string> = {
    'Claude Desktop': (port) => JSON.stringify({
        "mcp-overwatch": {
            type: "streamable-http",
            url: `http://localhost:${port}/mcp`,
        },
    }, null, 2),
    'Claude Code': (port) => JSON.stringify({
        "mcp-overwatch": {
            type: "http",
            url: `http://localhost:${port}/mcp`,
        },
    }, null, 2),
};

const defaultSnippet = (port: number) => JSON.stringify({
    "mcp-overwatch": {
        type: "http",
        url: `http://localhost:${port}/mcp`,
    },
}, null, 2);

const registerFns: Record<string, () => Promise<void>> = {
    'Claude Desktop': registerClaudeDesktop,
    'Claude Code': registerClaudeCode,
};

export default function ClientIntegration() {
    const [clients, setClients] = useState<ClientInfo[]>([]);
    const [registering, setRegistering] = useState<string | null>(null);
    const [registered, setRegistered] = useState<string | null>(null);
    const [expandedClient, setExpandedClient] = useState<string | null>(null);
    const [proxyPort, setProxyPort] = useState(3100);
    const [copied, setCopied] = useState(false);

    useEffect(() => {
        detectClients().then(setClients);
        getSettings().then((cfg) => setProxyPort(cfg.proxy.port));
    }, []);

    const handleRegister = async (e: React.MouseEvent, clientName: string) => {
        e.stopPropagation();
        const fn = registerFns[clientName];
        if (!fn) return;
        setRegistering(clientName);
        try {
            await fn();
            setRegistered(clientName);
            setTimeout(() => setRegistered(null), 3000);
        } finally {
            setRegistering(null);
        }
    };

    const handleCopy = async (e: React.MouseEvent, clientName: string) => {
        e.stopPropagation();
        const snippet = (configSnippets[clientName] || defaultSnippet)(proxyPort);
        await navigator.clipboard.writeText(snippet);
        setCopied(true);
        setTimeout(() => setCopied(false), 2000);
    };

    return (
        <div className="space-y-4">
            <p className="text-sm text-[#94a3b8]">
                Register MCP Overwatch with detected MCP client applications, or click a client to see the config snippet.
            </p>

            {clients.length === 0 ? (
                <div className="p-4 bg-[#1a1f2e] rounded-xl border border-white/[0.06] text-sm text-[#94a3b8]">
                    No MCP clients detected. Make sure Claude Desktop or another MCP client is installed.
                </div>
            ) : (
                clients.map((client) => {
                    const isExpanded = expandedClient === client.name;
                    const snippet = (configSnippets[client.name] || defaultSnippet)(proxyPort);
                    return (
                        <div
                            key={client.name}
                            className="bg-[#1a1f2e] rounded-xl border border-white/[0.06] overflow-hidden cursor-pointer hover:border-white/[0.12] transition-colors"
                            onClick={() => setExpandedClient(isExpanded ? null : client.name)}
                        >
                            <div className="flex items-center justify-between p-4">
                                <div>
                                    <div className="flex items-center gap-2">
                                        <span className="text-sm font-medium text-[#e2e8f0]">{client.name}</span>
                                        <span className={`px-2 py-0.5 text-xs rounded-full ${
                                            client.installed
                                                ? 'bg-emerald-500/15 text-emerald-400'
                                                : 'bg-white/[0.06] text-[#64748b]'
                                        }`}>
                                            {client.installed ? 'Installed' : 'Not Found'}
                                        </span>
                                    </div>
                                    {client.configPath && (
                                        <p className="text-xs text-[#64748b] font-mono mt-1 truncate max-w-xs">{client.configPath}</p>
                                    )}
                                </div>
                                <div className="flex items-center gap-2">
                                    {client.installed && registerFns[client.name] && (
                                        <button
                                            onClick={(e) => handleRegister(e, client.name)}
                                            disabled={registering !== null}
                                            className="px-4 py-2 text-sm font-medium bg-[#6366f1] hover:bg-[#818cf8] text-white rounded-lg transition-colors disabled:opacity-50"
                                        >
                                            {registering === client.name ? 'Registering...' : registered === client.name ? 'Registered!' : 'Auto-Register'}
                                        </button>
                                    )}
                                    <svg
                                        width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"
                                        className={`text-[#64748b] transition-transform ${isExpanded ? 'rotate-180' : ''}`}
                                    >
                                        <polyline points="6 9 12 15 18 9" />
                                    </svg>
                                </div>
                            </div>
                            {isExpanded && (
                                <div className="px-4 pb-4 border-t border-white/[0.06]">
                                    <div className="flex items-center justify-between mt-3 mb-2">
                                        <span className="text-xs text-[#94a3b8]">
                                            Add this to your client's <code className="text-indigo-400">mcpServers</code> config:
                                        </span>
                                        <button
                                            onClick={(e) => handleCopy(e, client.name)}
                                            className="px-3 py-1 text-xs font-medium bg-[#161b27] hover:bg-[#222838] text-[#94a3b8] hover:text-[#e2e8f0] rounded-md border border-white/[0.06] transition-colors"
                                        >
                                            {copied ? 'Copied!' : 'Copy'}
                                        </button>
                                    </div>
                                    <pre className="p-3 bg-[#161b27] rounded-lg text-xs font-mono text-[#e2e8f0] overflow-x-auto">
                                        {snippet}
                                    </pre>
                                </div>
                            )}
                        </div>
                    );
                })
            )}
        </div>
    );
}
