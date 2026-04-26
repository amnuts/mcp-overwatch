interface Props {
    raw: string;
}

export default function JsonRpcDetail({ raw }: Props) {
    let formatted: string;
    try {
        const parsed = JSON.parse(raw);
        formatted = JSON.stringify(parsed, null, 2);
    } catch {
        formatted = raw;
    }

    return (
        <div className="px-4 pb-2">
            <pre className="text-[11px] text-gray-400 bg-gray-950 border border-gray-800 rounded-lg p-3 overflow-x-auto font-mono whitespace-pre-wrap max-h-64 overflow-y-auto">
                {formatted}
            </pre>
        </div>
    );
}
