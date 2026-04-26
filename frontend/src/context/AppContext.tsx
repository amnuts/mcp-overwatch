import React, { createContext, useContext, useReducer, useCallback, useEffect } from 'react';
import type { InstalledServer, LogEntry, TabId } from '../types';
import { useWailsEvent } from '../hooks/useWailsEvent';
import { listInstalled, getRecentLogs, checkDocker } from '../services/api';

// ── State ───────────────────────────────────────────────────────────────

interface AppState {
    installedServers: InstalledServer[];
    logs: LogEntry[];
    activeTab: TabId;
    settingsOpen: boolean;
    selectedServerId: string | null;
    dockerAvailable: boolean;
}

const initialState: AppState = {
    installedServers: [],
    logs: [],
    activeTab: 'installed',
    settingsOpen: false,
    selectedServerId: null,
    dockerAvailable: false,
};

// ── Actions ─────────────────────────────────────────────────────────────

type Action =
    | { type: 'SET_INSTALLED'; payload: InstalledServer[] }
    | { type: 'UPDATE_SERVER_STATUS'; payload: { id: string; status: string; is_active?: boolean } }
    | { type: 'SET_LOGS'; payload: LogEntry[] }
    | { type: 'ADD_LOG'; payload: LogEntry }
    | { type: 'SET_TAB'; payload: TabId }
    | { type: 'SET_SETTINGS_OPEN'; payload: boolean }
    | { type: 'SET_SELECTED_SERVER'; payload: string | null }
    | { type: 'SET_DOCKER_AVAILABLE'; payload: boolean };

function reducer(state: AppState, action: Action): AppState {
    switch (action.type) {
        case 'SET_INSTALLED':
            return { ...state, installedServers: action.payload };
        case 'UPDATE_SERVER_STATUS':
            return {
                ...state,
                installedServers: state.installedServers.map((s) =>
                    s.id === action.payload.id
                        ? {
                              ...s,
                              status: action.payload.status as InstalledServer['status'],
                              ...(action.payload.is_active !== undefined && { is_active: action.payload.is_active }),
                          }
                        : s,
                ),
            };
        case 'SET_LOGS':
            return { ...state, logs: action.payload };
        case 'ADD_LOG':
            return { ...state, logs: [...state.logs, action.payload] };
        case 'SET_TAB':
            return { ...state, activeTab: action.payload };
        case 'SET_SETTINGS_OPEN':
            return { ...state, settingsOpen: action.payload };
        case 'SET_SELECTED_SERVER':
            return { ...state, selectedServerId: action.payload };
        case 'SET_DOCKER_AVAILABLE':
            return { ...state, dockerAvailable: action.payload };
        default:
            return state;
    }
}

// ── Context ─────────────────────────────────────────────────────────────

interface AppContextValue {
    state: AppState;
    dispatch: React.Dispatch<Action>;
    refreshInstalled: () => Promise<void>;
}

const AppContext = createContext<AppContextValue | null>(null);

export function useApp(): AppContextValue {
    const ctx = useContext(AppContext);
    if (!ctx) throw new Error('useApp must be used inside AppProvider');
    return ctx;
}

// ── Provider ────────────────────────────────────────────────────────────

export function AppProvider({ children }: { children: React.ReactNode }) {
    const [state, dispatch] = useReducer(reducer, initialState);

    const refreshInstalled = useCallback(async () => {
        const servers = await listInstalled();
        dispatch({ type: 'SET_INSTALLED', payload: servers });
    }, []);

    // Load initial data
    useEffect(() => {
        refreshInstalled();
        getRecentLogs(500).then((logs) => dispatch({ type: 'SET_LOGS', payload: logs }));
        checkDocker().then((available) => dispatch({ type: 'SET_DOCKER_AVAILABLE', payload: available }));
    }, [refreshInstalled]);

    // Subscribe to real-time events from the Go backend
    const onServerStatus = useCallback((data: { id: string; status: string; is_active?: boolean }) => {
        dispatch({ type: 'UPDATE_SERVER_STATUS', payload: data });
    }, []);

    const onLogEntry = useCallback((data: LogEntry) => {
        dispatch({ type: 'ADD_LOG', payload: data });
    }, []);

    useWailsEvent('server:status', onServerStatus);
    useWailsEvent('log:entry', onLogEntry);

    const value: AppContextValue = { state, dispatch, refreshInstalled };

    return <AppContext.Provider value={value}>{children}</AppContext.Provider>;
}
