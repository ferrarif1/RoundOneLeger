import { FormEvent, useEffect, useState } from 'react';
import api from '../api/client';
import { ShieldCheckIcon } from '@heroicons/react/24/outline';
import { useSession } from '../hooks/useSession';

interface AllowRule {
  id: string;
  label?: string;
  cidr: string;
  description?: string;
  created_at: string;
}

const IPAllowlist = () => {
  const { admin } = useSession();
  const [rules, setRules] = useState<AllowRule[]>([]);
  const [cidr, setCidr] = useState('');
  const [description, setDescription] = useState('');
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    (async () => {
      try {
        const { data } = await api.get<{ items: AllowRule[] }>('/api/v1/ip-allowlist');
        setRules(data.items ?? []);
        setError(null);
      } catch (err) {
        console.error('Failed to load IP allowlist', err);
        setError('无法加载白名单，请稍后刷新页面。');
      } finally {
        setLoading(false);
      }
    })();
  }, []);

  const handleCreate = async (event: FormEvent) => {
    event.preventDefault();
    if (!admin) {
      return;
    }
    const { data } = await api.post<AllowRule>('/api/v1/ip-allowlist', { cidr, description });
    setRules((prev) => [data, ...prev]);
    setCidr('');
    setDescription('');
  };

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h2 className="section-title">IP 白名单</h2>
      </div>
      <form
        onSubmit={handleCreate}
        className="grid gap-4 rounded-2xl border border-[var(--line)] bg-white p-6 shadow-[0_20px_36px_rgba(0,0,0,0.08)] md:grid-cols-[2fr,2fr,auto]"
      >
        <div>
          <label className="text-xs uppercase tracking-wider text-[rgba(20,20,20,0.45)]">CIDR</label>
          <input
            value={cidr}
            onChange={(e) => setCidr(e.target.value)}
            placeholder="192.168.1.0/24"
            required
            disabled={!admin}
            className="mt-2 w-full border border-[var(--line)] bg-white px-4 py-3 text-sm disabled:cursor-not-allowed disabled:bg-[var(--bg-subtle)] disabled:text-[rgba(20,20,20,0.35)]"
          />
        </div>
        <div>
          <label className="text-xs uppercase tracking-wider text-[rgba(20,20,20,0.45)]">备注</label>
          <input
            value={description}
            onChange={(e) => setDescription(e.target.value)}
            placeholder="总部办公室"
            disabled={!admin}
            className="mt-2 w-full border border-[var(--line)] bg-white px-4 py-3 text-sm disabled:cursor-not-allowed disabled:bg-[var(--bg-subtle)] disabled:text-[rgba(20,20,20,0.35)]"
          />
        </div>
        <button type="submit" className="button-primary self-end" disabled={!admin}>
          {admin ? '添加' : '仅管理员可配置'}
        </button>
      </form>

      {!admin && (
        <div className="rounded-2xl border border-[var(--line)] bg-[var(--bg-subtle)] p-4 text-sm text-[rgba(20,20,20,0.6)]">
          当前账号仅可查看白名单。请联系管理员调整访问策略。
        </div>
      )}

      {error && (
        <div className="rounded-2xl border border-red-200 bg-red-50 p-4 text-sm text-red-600">{error}</div>
      )}

      <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-3">
        {loading &&
          Array.from({ length: 3 }).map((_, index) => (
            <div key={index} className="h-28 animate-pulse rounded-2xl border border-[var(--line)] bg-white" />
          ))}
        {!loading && rules.length === 0 && (
          <div className="rounded-2xl border border-[var(--line)] bg-white p-5 text-sm text-[rgba(20,20,20,0.6)]">
            暂无白名单记录。
          </div>
        )}
        {!loading &&
          rules.map((rule) => (
            <div
              key={rule.id}
              className="rounded-2xl border border-[var(--line)] bg-white p-5 shadow-[0_20px_36px_rgba(0,0,0,0.08)]"
            >
              <div className="flex items-center gap-3">
                <div className="rounded-xl border border-[rgba(20,20,20,0.12)] bg-[var(--bg-subtle)] p-2">
                  <ShieldCheckIcon className="h-6 w-6 text-[var(--accent)]" />
                </div>
                <div>
                  <p className="text-sm font-semibold text-[var(--text)]">{rule.cidr}</p>
                  <p className="text-xs text-[rgba(20,20,20,0.55)]">
                    {rule.description || rule.label || '无描述'}
                  </p>
                </div>
              </div>
              <p className="mt-4 text-xs text-[rgba(20,20,20,0.45)]">
                添加于 {new Date(rule.created_at).toLocaleString()}
              </p>
            </div>
          ))}
      </div>
    </div>
  );
};

export default IPAllowlist;
