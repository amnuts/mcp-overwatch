import { useState } from 'react';
import { useApp } from '../../context/AppContext';
import { importFromGitHub, importFromLocal, browseDirectory } from '../../services/api';

export default function ImportTab() {
    const { dispatch, refreshInstalled } = useApp();

    // GitHub import state
    const [gitUrl, setGitUrl] = useState('');
    const [gitLoading, setGitLoading] = useState(false);
    const [gitError, setGitError] = useState('');
    const [gitSuccess, setGitSuccess] = useState('');

    // Local import state
    const [localDir, setLocalDir] = useState('');
    const [copyToOverwatch, setCopyToOverwatch] = useState(false);
    const [localLoading, setLocalLoading] = useState(false);
    const [localError, setLocalError] = useState('');
    const [localSuccess, setLocalSuccess] = useState('');

    const refreshAndNavigate = async () => {
        await refreshInstalled();
        dispatch({ type: 'SET_TAB', payload: 'installed' });
    };

    const handleGitImport = async () => {
        setGitError('');
        setGitSuccess('');
        setGitLoading(true);
        try {
            const result = await importFromGitHub(gitUrl);
            if (result) {
                setGitSuccess(`Successfully imported "${result.display_name}"`);
                setGitUrl('');
                setTimeout(() => refreshAndNavigate(), 1500);
            }
        } catch (e: any) {
            setGitError(e?.message || String(e));
        } finally {
            setGitLoading(false);
        }
    };

    const handleBrowse = async () => {
        try {
            const dir = await browseDirectory();
            if (dir) setLocalDir(dir);
        } catch (e: any) {
            setLocalError(e?.message || String(e));
        }
    };

    const handleLocalImport = async () => {
        setLocalError('');
        setLocalSuccess('');
        setLocalLoading(true);
        try {
            const result = await importFromLocal(localDir, copyToOverwatch);
            if (result) {
                setLocalSuccess(`Successfully imported "${result.display_name}"`);
                setLocalDir('');
                setTimeout(() => refreshAndNavigate(), 1500);
            }
        } catch (e: any) {
            setLocalError(e?.message || String(e));
        } finally {
            setLocalLoading(false);
        }
    };

    return (
        <div className="h-full overflow-y-auto p-6">
            <div className="max-w-3xl mx-auto space-y-6">
                <div>
                    <h2 className="text-lg font-semibold text-white mb-1">Import MCP Server</h2>
                    <p className="text-sm text-[#94a3b8]">
                        Import an MCP server from a GitHub repository or a local directory.
                        The server must contain a package.json, pyproject.toml, or Go module with MCP SDK dependencies.
                    </p>
                </div>

                {/* GitHub Import */}
                <div className="bg-[#1e2536] border border-white/[0.08] rounded-xl p-5 space-y-4">
                    <div className="flex items-center gap-3">
                        <svg width="20" height="20" viewBox="0 0 24 24" fill="currentColor" className="text-white shrink-0">
                            <path d="M12 0C5.37 0 0 5.37 0 12c0 5.31 3.435 9.795 8.205 11.385.6.105.825-.255.825-.57 0-.285-.015-1.23-.015-2.235-3.015.555-3.795-.735-4.035-1.41-.135-.345-.72-1.41-1.23-1.695-.42-.225-1.02-.78-.015-.795.945-.015 1.62.87 1.845 1.23 1.08 1.815 2.805 1.305 3.495.99.105-.78.42-1.305.765-1.605-2.67-.3-5.46-1.335-5.46-5.925 0-1.305.465-2.385 1.23-3.225-.12-.3-.54-1.53.12-3.18 0 0 1.005-.315 3.3 1.23.96-.27 1.98-.405 3-.405s2.04.135 3 .405c2.295-1.56 3.3-1.23 3.3-1.23.66 1.65.24 2.88.12 3.18.765.84 1.23 1.905 1.23 3.225 0 4.605-2.805 5.625-5.475 5.925.435.375.81 1.095.81 2.22 0 1.605-.015 2.895-.015 3.3 0 .315.225.69.825.57A12.02 12.02 0 0024 12c0-6.63-5.37-12-12-12z" />
                        </svg>
                        <h3 className="text-base font-medium text-white">From GitHub</h3>
                    </div>
                    <p className="text-sm text-[#94a3b8]">
                        Clone a public GitHub repository containing an MCP server.
                    </p>
                    <div className="flex gap-3">
                        <input
                            type="text"
                            value={gitUrl}
                            onChange={(e) => { setGitUrl(e.target.value); setGitError(''); }}
                            placeholder="https://github.com/owner/repo"
                            className="flex-1 px-4 py-2.5 text-sm bg-[#161b27] border border-white/[0.08] rounded-lg text-[#e2e8f0] placeholder-[#64748b] focus:outline-none focus:border-[#6366f1] focus:ring-1 focus:ring-[#6366f1]/30"
                            disabled={gitLoading}
                            onKeyDown={(e) => e.key === 'Enter' && gitUrl.trim() && handleGitImport()}
                        />
                        <button
                            onClick={handleGitImport}
                            disabled={gitLoading || !gitUrl.trim()}
                            className="px-5 py-2.5 text-sm font-medium rounded-lg bg-indigo-600 hover:bg-indigo-500 text-white transition-colors disabled:opacity-50 disabled:cursor-not-allowed whitespace-nowrap shrink-0"
                        >
                            {gitLoading ? (
                                <span className="flex items-center gap-2">
                                    <svg className="animate-spin h-4 w-4" viewBox="0 0 24 24">
                                        <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" fill="none" />
                                        <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z" />
                                    </svg>
                                    Cloning...
                                </span>
                            ) : 'Clone & Import'}
                        </button>
                    </div>
                    {gitError && (
                        <div className="flex items-start gap-2 p-3 bg-red-500/10 border border-red-500/20 rounded-lg">
                            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" className="text-red-400 shrink-0 mt-0.5">
                                <circle cx="12" cy="12" r="10" />
                                <line x1="15" y1="9" x2="9" y2="15" />
                                <line x1="9" y1="9" x2="15" y2="15" />
                            </svg>
                            <span className="text-sm text-red-300">{gitError}</span>
                        </div>
                    )}
                    {gitSuccess && (
                        <div className="flex items-start gap-2 p-3 bg-emerald-500/10 border border-emerald-500/20 rounded-lg">
                            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" className="text-emerald-400 shrink-0 mt-0.5">
                                <path d="M22 11.08V12a10 10 0 11-5.93-9.14" />
                                <polyline points="22 4 12 14.01 9 11.01" />
                            </svg>
                            <span className="text-sm text-emerald-300">{gitSuccess}</span>
                        </div>
                    )}
                </div>

                {/* Local Directory Import */}
                <div className="bg-[#1e2536] border border-white/[0.08] rounded-xl p-5 space-y-4">
                    <div className="flex items-center gap-3">
                        <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" className="text-white shrink-0">
                            <path d="M22 19a2 2 0 01-2 2H4a2 2 0 01-2-2V5a2 2 0 012-2h5l2 3h9a2 2 0 012 2z" />
                        </svg>
                        <h3 className="text-base font-medium text-white">From Local Directory</h3>
                    </div>
                    <p className="text-sm text-[#94a3b8]">
                        Import an MCP server from a directory on your machine.
                    </p>
                    <div className="flex gap-3">
                        <input
                            type="text"
                            value={localDir}
                            onChange={(e) => { setLocalDir(e.target.value); setLocalError(''); }}
                            placeholder="/path/to/mcp-server"
                            className="flex-1 px-4 py-2.5 text-sm bg-[#161b27] border border-white/[0.08] rounded-lg text-[#e2e8f0] placeholder-[#64748b] focus:outline-none focus:border-[#6366f1] focus:ring-1 focus:ring-[#6366f1]/30"
                            disabled={localLoading}
                        />
                        <button
                            onClick={handleBrowse}
                            disabled={localLoading}
                            className="px-4 py-2.5 text-sm font-medium rounded-lg bg-slate-600 hover:bg-slate-500 text-slate-200 border border-white/[0.06] transition-colors disabled:opacity-50 whitespace-nowrap shrink-0"
                        >
                            Browse...
                        </button>
                    </div>
                    <label className="flex items-center gap-3 cursor-pointer">
                        <div
                            onClick={() => setCopyToOverwatch(!copyToOverwatch)}
                            className={`relative w-10 h-5 rounded-full transition-colors ${copyToOverwatch ? 'bg-indigo-600' : 'bg-slate-600'}`}
                        >
                            <div className={`absolute top-0.5 left-0.5 w-4 h-4 bg-white rounded-full transition-transform ${copyToOverwatch ? 'translate-x-5' : ''}`} />
                        </div>
                        <span className="text-sm text-[#e2e8f0]">Copy into MCP Overwatch</span>
                        <span className="text-xs text-[#64748b]">
                            {copyToOverwatch ? '(files will be copied)' : '(server runs from original location)'}
                        </span>
                    </label>
                    <button
                        onClick={handleLocalImport}
                        disabled={localLoading || !localDir.trim()}
                        className="px-5 py-2.5 text-sm font-medium rounded-lg bg-indigo-600 hover:bg-indigo-500 text-white transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
                    >
                        {localLoading ? (
                            <span className="flex items-center gap-2">
                                <svg className="animate-spin h-4 w-4" viewBox="0 0 24 24">
                                    <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" fill="none" />
                                    <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z" />
                                </svg>
                                Importing...
                            </span>
                        ) : 'Import'}
                    </button>
                    {localError && (
                        <div className="flex items-start gap-2 p-3 bg-red-500/10 border border-red-500/20 rounded-lg">
                            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" className="text-red-400 shrink-0 mt-0.5">
                                <circle cx="12" cy="12" r="10" />
                                <line x1="15" y1="9" x2="9" y2="15" />
                                <line x1="9" y1="9" x2="15" y2="15" />
                            </svg>
                            <span className="text-sm text-red-300">{localError}</span>
                        </div>
                    )}
                    {localSuccess && (
                        <div className="flex items-start gap-2 p-3 bg-emerald-500/10 border border-emerald-500/20 rounded-lg">
                            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" className="text-emerald-400 shrink-0 mt-0.5">
                                <path d="M22 11.08V12a10 10 0 11-5.93-9.14" />
                                <polyline points="22 4 12 14.01 9 11.01" />
                            </svg>
                            <span className="text-sm text-emerald-300">{localSuccess}</span>
                        </div>
                    )}
                </div>
            </div>
        </div>
    );
}
