import { FormEvent, useEffect, useState } from 'react';
import api from '../api/client';
import { ShieldCheckIcon } from '@heroicons/react/24/outline';

interface AllowRule {
  id: string;
  cidr: string;
  description: string;
  created_at: string;
}

const IPAllowlist = () => {
  const [rules, setRules] = useState<AllowRule[]>([]);
  const [cidr, setCidr] = useState('');
  const [description, setDescription] = useState('');

  useEffect(() => {
    (async () => {
      const { data } = await api.get('/ip-allowlists');
      setRules(data);
    })();
  }, []);

  const handleCreate = async (event: FormEvent) => {
    event.preventDefault();
    const { data } = await api.post('/ip-allowlists', { cidr, description });
    setRules((prev) => [data, ...prev]);
    setCidr('');
    setDescription('');
  };

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h2 className="section-title">IP 白名单</h2>
      </div>
      <form onSubmit={handleCreate} className="grid gap-4 rounded-3xl border border-ink-200 bg-white p-6 shadow-glow md:grid-cols-[2fr,2fr,auto]">
        <div>
          <label className="text-xs uppercase tracking-wider text-night-300">CIDR</label>
          <input
            value={cidr}
            onChange={(e) => setCidr(e.target.value)}
            placeholder="192.168.1.0/24"
            required
            className="mt-2 w-full rounded-2xl border border-ink-200 bg-white px-4 py-3 text-sm focus:border-neon-500 focus:ring-neon-500/30"
          />
        </div>
        <div>
          <label className="text-xs uppercase tracking-wider text-night-300">备注</label>
          <input
            value={description}
            onChange={(e) => setDescription(e.target.value)}
            placeholder="总部办公室"
            className="mt-2 w-full rounded-2xl border border-ink-200 bg-white px-4 py-3 text-sm focus:border-neon-500 focus:ring-neon-500/30"
          />
        </div>
        <button type="submit" className="button-primary self-end">
          添加
        </button>
      </form>

      <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-3">
        {rules.map((rule) => (
          <div key={rule.id} className="rounded-3xl border border-white bg-white/90 p-5 shadow-glow">
            <div className="flex items-center gap-3">
              <ShieldCheckIcon className="h-6 w-6 text-neon-500" />
              <div>
                <p className="text-sm font-semibold text-night-100">{rule.cidr}</p>
                <p className="text-xs text-night-300">{rule.description || '无描述'}</p>
              </div>
            </div>
            <p className="mt-4 text-xs text-night-400">
              添加于 {new Date(rule.created_at).toLocaleString()}
            </p>
          </div>
        ))}
      </div>
    </div>
  );
};

export default IPAllowlist;
