import { Window } from '@wailsio/runtime';

export default function TitleBar() {
    return (
        <header
            className="flex items-center justify-between h-9 bg-slate-800 shrink-0 select-none"
            style={{ '--wails-draggable': 'drag' } as React.CSSProperties}
        >
            {/* Left: app branding */}
            <div className="flex items-center gap-2.5 pl-4">
                <div className="w-5 h-5 rounded-md bg-linear-to-br from-indigo-500 to-purple-600 flex items-center justify-center shadow-lg shadow-indigo-500/20">
                    <span className="text-[9px] font-black text-slate-200 leading-none">
                        <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" strokeWidth={2} stroke="currentColor" className="size-4">
                          <path strokeLinecap="round" strokeLinejoin="round" d="M2.036 12.322a1.012 1.012 0 0 1 0-.639C3.423 7.51 7.36 4.5 12 4.5c4.638 0 8.573 3.007 9.963 7.178.07.207.07.431 0 .639C20.577 16.49 16.64 19.5 12 19.5c-4.638 0-8.573-3.007-9.963-7.178Z" />
                          <path strokeLinecap="round" strokeLinejoin="round" d="M15 12a3 3 0 1 1-6 0 3 3 0 0 1 6 0Z" />
                        </svg>
                    </span>
                </div>
                <span className="text-sm font-semibold text-slate-400 tracking-wide">
                    MCP Overwatch
                </span>
            </div>

            {/* Right: window controls */}
            <div
                className="flex items-center h-full gap-0 pr-0"
                style={{ '--wails-draggable': 'no-drag' } as React.CSSProperties}
            >
                <button
                    onClick={() => Window.Minimise()}
                    className="flex items-center justify-center w-12 h-full m-0 text-slate-400 hover:bg-white/6 hover:text-slate-300 transition-colors"
                    aria-label="Minimize"
                >
                    <svg width="10" height="1" viewBox="0 0 10 1">
                        <rect width="10" height="1" fill="currentColor" />
                    </svg>
                </button>
                <button
                    onClick={() => Window.ToggleMaximise()}
                    className="flex items-center justify-center w-12 h-full m-0 text-slate-400 hover:bg-white/6 hover:text-slate-300 transition-colors"
                    aria-label="Maximize"
                >
                    <svg width="10" height="10" viewBox="0 0 10 10" fill="none" stroke="currentColor" strokeWidth="1">
                        <rect x="0.5" y="0.5" width="9" height="9" />
                    </svg>
                </button>
                <button
                    onClick={() => Window.Close()}
                    className="flex items-center justify-center w-12 h-full m=- text-slate-500 hover:bg-red-600 hover:text-white transition-colors"
                    aria-label="Close"
                >
                    <svg width="10" height="10" viewBox="0 0 10 10" stroke="currentColor" strokeWidth="1.5">
                        <line x1="1" y1="1" x2="9" y2="9" />
                        <line x1="9" y1="1" x2="1" y2="9" />
                    </svg>
                </button>
            </div>
        </header>
    );
}
