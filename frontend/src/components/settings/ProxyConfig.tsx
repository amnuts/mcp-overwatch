import type { AppConfig } from '../../types';

interface Props {
    config: AppConfig;
    onChange: (config: AppConfig) => void;
}

export default function ProxyConfig({ config, onChange }: Props) {
    return (
        <div className="space-y-5">
            <div>
                <label className="block text-sm font-medium text-[#e2e8f0] mb-1">Proxy Port</label>
                <p className="text-xs text-[#94a3b8] mb-2">
                    The port on which the unified MCP proxy listens for client connections.
                </p>
                <input
                    type="number"
                    value={config.proxy.port}
                    onChange={(e) =>
                        onChange({
                            ...config,
                            proxy: { ...config.proxy, port: parseInt(e.target.value) || 8200 },
                        })
                    }
                    className="w-32 px-3 py-2 text-sm bg-[#161b27] border border-white/[0.08] rounded-lg text-[#e2e8f0] focus:outline-none focus:border-[#6366f1] focus:ring-1 focus:ring-[#6366f1]/30"
                />
            </div>

            <div>
                <label className="block text-sm font-medium text-[#e2e8f0] mb-1">Namespace Format</label>
                <p className="text-xs text-[#94a3b8] mb-2">
                    How tool names are namespaced to avoid conflicts between servers.
                </p>
                <select
                    value={config.proxy.namespace_format}
                    onChange={(e) =>
                        onChange({
                            ...config,
                            proxy: { ...config.proxy, namespace_format: e.target.value },
                        })
                    }
                    className="w-72 px-3 py-2 text-sm bg-[#161b27] border border-white/[0.08] rounded-lg text-[#e2e8f0] focus:outline-none focus:border-[#6366f1]"
                >
                    <option value="{server}__{tool}">{'{server}__{tool}'} (recommended)</option>
                    <option value="{tool}">{'{tool}'} (no namespace)</option>
                </select>
            </div>
        </div>
    );
}
