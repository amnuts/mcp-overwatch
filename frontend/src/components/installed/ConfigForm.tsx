import { useState, useEffect } from 'react';
import type { EnvVarDef } from '../../types';
import { configureServer } from '../../services/api';

interface Props {
    serverId: string;
    envVarsJson: string;
    userConfigJson: string;
}

export default function ConfigForm({ serverId, envVarsJson, userConfigJson }: Props) {
    const envVars: EnvVarDef[] = (() => {
        try {
            const parsed = JSON.parse(envVarsJson || '[]');
            return Array.isArray(parsed) ? parsed : [];
        } catch {
            return [];
        }
    })();

    const [values, setValues] = useState<Record<string, string>>({});
    const [saving, setSaving] = useState(false);
    const [saved, setSaved] = useState(false);

    useEffect(() => {
        try {
            const existing = JSON.parse(userConfigJson || '{}');
            const initial: Record<string, string> = {};
            for (const v of envVars) {
                initial[v.name] = existing[v.name] ?? v.default ?? '';
            }
            setValues(initial);
        } catch {
            const initial: Record<string, string> = {};
            for (const v of envVars) {
                initial[v.name] = v.default ?? '';
            }
            setValues(initial);
        }
    }, [envVarsJson, userConfigJson]); // eslint-disable-line react-hooks/exhaustive-deps

    if (envVars.length === 0) {
        return <p className="text-sm text-[#64748b] italic">No configuration required.</p>;
    }

    const handleSave = async () => {
        setSaving(true);
        await configureServer(serverId, JSON.stringify(values));
        setSaving(false);
        setSaved(true);
        setTimeout(() => setSaved(false), 2000);
    };

    return (
        <div className="space-y-4">
            {envVars.map((v) => (
                <div key={v.name}>
                    <label className="block text-sm font-medium text-slate-200 mb-1">
                        {v.name}
                        {v.isRequired && <span className="text-[#f87171] ml-0.5">*</span>}
                    </label>
                    {v.description && (
                        <p className="text-xs text-[#64748b] mb-1.5">{v.description}</p>
                    )}
                    <input
                        type={v.isSecret ? 'password' : 'text'}
                        value={values[v.name] ?? ''}
                        onChange={(e) => setValues({ ...values, [v.name]: e.target.value })}
                        placeholder={v.default || undefined}
                        className="w-full px-3 py-2 text-sm bg-[#161b27] border border-white/[0.08] rounded-lg text-[#e2e8f0] placeholder-[#64748b] focus:outline-none focus:border-indigo-500 focus:ring-1 focus:ring-indigo-500/30"
                    />
                </div>
            ))}
            <button
                onClick={handleSave}
                disabled={saving}
                className="px-5 py-2 text-sm font-medium bg-indigo-600 hover:bg-indigo-500 text-white rounded-lg transition-colors disabled:opacity-50"
            >
                {saving ? 'Saving...' : saved ? 'Saved!' : 'Save Configuration'}
            </button>
        </div>
    );
}
