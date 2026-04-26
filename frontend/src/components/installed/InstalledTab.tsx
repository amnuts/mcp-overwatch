import { useApp } from '../../context/AppContext';
import ServerCard from './ServerCard';
import ServerDetail from './ServerDetail';

export default function InstalledTab() {
    const { state, dispatch } = useApp();
    const selected = state.installedServers.find((s) => s.id === state.selectedServerId) ?? null;

    return (
        <div className="flex h-full">
            <div className={`overflow-y-auto p-5 ${selected ? 'w-[45%] border-r border-white/6 space-y-3' : 'w-full'}`}>
                {state.installedServers.length === 0 ? (
                    <div className="flex flex-col items-center justify-center h-full text-slate-400">
                        <svg width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1" className="mb-4">
                            <rect x="2" y="3" width="20" height="14" rx="2" />
                            <line x1="8" y1="21" x2="16" y2="21" />
                            <line x1="12" y1="17" x2="12" y2="21" />
                        </svg>
                        <p className="text-sm font-medium">No servers installed</p>
                        <p className="text-sm mt-1">
                            <a href="#" onClick={(e) => {
                                e.preventDefault();
                                dispatch({ type: 'SET_TAB', payload: 'marketplace' });
                            }} className="text-[#6366f1] hover:text-[#818cf8] cursor-pointer">Browse the Marketplace</a> to find and install MCP servers.
                        </p>
                    </div>
                ) : selected ? (
                    <div className="space-y-3">
                        {state.installedServers.map((server) => (
                            <ServerCard
                                key={server.id}
                                server={server}
                                selected={server.id === state.selectedServerId}
                                onSelect={() =>
                                    dispatch({
                                        type: 'SET_SELECTED_SERVER',
                                        payload: server.id === state.selectedServerId ? null : server.id,
                                    })
                                }
                            />
                        ))}
                    </div>
                ) : (
                    <div className="grid grid-cols-1 lg:grid-cols-2 xl:grid-cols-3 gap-4">
                        {state.installedServers.map((server) => (
                            <ServerCard
                                key={server.id}
                                server={server}
                                selected={server.id === state.selectedServerId}
                                onSelect={() =>
                                    dispatch({
                                        type: 'SET_SELECTED_SERVER',
                                        payload: server.id === state.selectedServerId ? null : server.id,
                                    })
                                }
                            />
                        ))}
                    </div>
                )}
            </div>
            {selected && (
                <div className="w-[55%]">
                    <ServerDetail server={selected} />
                </div>
            )}
        </div>
    );
}
