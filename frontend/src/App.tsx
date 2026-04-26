import { AppProvider, useApp } from './context/AppContext';
import TitleBar from './components/layout/TitleBar';
import TabBar from './components/layout/TabBar';
import StatusBar from './components/layout/StatusBar';
import InstalledTab from './components/installed/InstalledTab';
import MarketplaceTab from './components/available/MarketplaceTab';
import ImportTab from './components/import/ImportTab';
import LogsTab from './components/logs/LogsTab';
import StatsTab from './components/stats/StatsTab';
import SettingsModal from './components/settings/SettingsModal';

function AppContent() {
    const { state } = useApp();

    return (
        <div className="flex flex-col w-screen h-screen bg-slate-700 text-slate-300 overflow-hidden">
            <TitleBar />
            <TabBar />
            <main className="flex-1 min-h-0 overflow-hidden">
                {state.activeTab === 'installed' && <InstalledTab />}
                {state.activeTab === 'marketplace' && <MarketplaceTab />}
                {state.activeTab === 'import' && <ImportTab />}
                {state.activeTab === 'logs' && <LogsTab />}
                {state.activeTab === 'stats' && <StatsTab />}
            </main>
            <StatusBar />
            {state.settingsOpen && <SettingsModal />}
        </div>
    );
}

export default function App() {
    return (
        <AppProvider>
            <AppContent />
        </AppProvider>
    );
}
