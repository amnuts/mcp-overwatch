import { useApp } from '../../context/AppContext';

export default function StatusBar() {
    const { state } = useApp();

    const activeCount = state.installedServers.filter((s) => s.status === 'running').length;
    const totalCount = state.installedServers.length;

    return (
        <div className="flex items-center justify-between h-7 bg-slate-800 pl-4 pr-4 text-xs text-slate-500">
            <div className="flex items-center gap-4 mb-1">
                <span className="flex items-center gap-1.5">
                    <span className="inline-block w-1.5 h-1.5 rounded-full bg-emerald-400 shadow-[0_0_4px] shadow-emerald-400/60" />
                    <span className="text-slate-400">Proxy :3100</span>
                </span>
                <span className="text-slate-500">
                    {activeCount}/{totalCount} servers active
                </span>
            </div>
            <div className="flex items-center gap-4 mb-1">
                <span>{state.logs.length} log entries</span>
            </div>
        </div>
    );
}
