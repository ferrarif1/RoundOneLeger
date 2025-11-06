import { FormEvent, useEffect, useState } from 'react';
import api from '../api/client';
import { PencilSquareIcon, ShieldCheckIcon, TrashIcon } from '@heroicons/react/24/outline';
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
  const [creating, setCreating] = useState(false);
  const [editingId, setEditingId] = useState<string | null>(null);
  const [editCidr, setEditCidr] = useState('');
  const [editDescription, setEditDescription] = useState('');
  const [updatingId, setUpdatingId] = useState<string | null>(null);
  const [deletingId, setDeletingId] = useState<string | null>(null);

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
    if (!cidr.trim()) {
      setError('请输入有效的 CIDR 或 IP 地址。');
      return;
    }
    setCreating(true);
    try {
      const { data } = await api.post<AllowRule>('/api/v1/ip-allowlist', { cidr: cidr.trim(), description: description.trim() });
      setRules((prev) => [data, ...prev.filter((rule) => rule.id !== data.id)]);
      setCidr('');
      setDescription('');
      setError(null);
    } catch (err) {
      console.error('Failed to create allowlist rule', err);
      setError('添加白名单失败，请确认格式是否正确。');
    } finally {
      setCreating(false);
    }
  };

  const beginEdit = (rule: AllowRule) => {
    setEditingId(rule.id);
    setEditCidr(rule.cidr);
    setEditDescription(rule.description ?? '');
  };

  const cancelEdit = () => {
    setEditingId(null);
    setEditCidr('');
    setEditDescription('');
  };

  const handleUpdate = async (id: string) => {
    if (!admin) {
      return;
    }
    if (!editCidr.trim()) {
      setError('请输入有效的 CIDR 或 IP 地址。');
      return;
    }
    setUpdatingId(id);
    try {
      const { data } = await api.put<AllowRule>(`/api/v1/ip-allowlist/${id}`, {
        cidr: editCidr.trim(),
        description: editDescription.trim()
      });
      setRules((prev) => prev.map((rule) => (rule.id === id ? data : rule)));
      cancelEdit();
      setError(null);
    } catch (err) {
      console.error('Failed to update allowlist rule', err);
      setError('更新白名单失败，请检查输入格式。');
    } finally {
      setUpdatingId(null);
    }
  };

  const handleDelete = async (id: string) => {
    if (!admin) {
      return;
    }
    if (!window.confirm('确定要移除此白名单规则吗？')) {
      return;
    }
    setDeletingId(id);
    try {
      await api.delete(`/api/v1/ip-allowlist/${id}`);
      setRules((prev) => prev.filter((rule) => rule.id !== id));
      if (editingId === id) {
        cancelEdit();
      }
      setError(null);
    } catch (err) {
      console.error('Failed to delete allowlist rule', err);
      setError('删除白名单失败，请稍后重试。');
    } finally {
      setDeletingId(null);
    }
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
        <button type="submit" className="button-primary self-end" disabled={!admin || creating}>
          {!admin ? '仅管理员可配置' : creating ? '添加中…' : '添加'}
        </button>
      </form>

      <p className="text-xs text-[rgba(20,20,20,0.55)]">
        支持录入单个 IP 或 CIDR 段，可使用 <code className="rounded bg-[var(--bg-subtle)] px-1 py-0.5">0.0.0.0/24</code> 允许任意来源访问。
      </p>

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
          rules.map((rule) => {
            const isEditing = editingId === rule.id;
            return (
              <div
                key={rule.id}
                className="flex flex-col gap-4 rounded-2xl border border-[var(--line)] bg-white p-5 shadow-[0_20px_36px_rgba(0,0,0,0.08)]"
              >
                <div className="flex items-start justify-between gap-3">
                  <div className="flex items-center gap-3">
                    <div className="rounded-xl border border-[rgba(20,20,20,0.12)] bg-[var(--bg-subtle)] p-2">
                      <ShieldCheckIcon className="h-6 w-6 text-[var(--accent)]" />
                    </div>
                    {isEditing ? (
                      <form
                        onSubmit={(event) => {
                          event.preventDefault();
                          handleUpdate(rule.id);
                        }}
                        className="space-y-3"
                      >
                        <div className="space-y-2">
                          <label className="text-xs uppercase tracking-wider text-[rgba(20,20,20,0.45)]">CIDR</label>
                          <input
                            value={editCidr}
                            onChange={(e) => setEditCidr(e.target.value)}
                            required
                            className="w-full border border-[var(--line)] bg-white px-4 py-2 text-sm"
                          />
                        </div>
                        <div className="space-y-2">
                          <label className="text-xs uppercase tracking-wider text-[rgba(20,20,20,0.45)]">备注</label>
                          <input
                            value={editDescription}
                            onChange={(e) => setEditDescription(e.target.value)}
                            className="w-full border border-[var(--line)] bg-white px-4 py-2 text-sm"
                            placeholder="例如：总部办公室"
                          />
                        </div>
                        <div className="flex flex-wrap gap-2">
                          <button
                            type="submit"
                            className="button-primary"
                            disabled={updatingId === rule.id}
                          >
                            {updatingId === rule.id ? '保存中…' : '保存'}
                          </button>
                          <button
                            type="button"
                            onClick={cancelEdit}
                            className="inline-flex items-center gap-1 rounded-[var(--radius-sm)] border border-[var(--line)] px-3 py-2 text-xs text-[var(--text)] transition hover:border-[var(--line-strong)]"
                            disabled={updatingId === rule.id}
                          >
                            取消
                          </button>
                        </div>
                      </form>
                    ) : (
                      <div>
                        <p className="text-sm font-semibold text-[var(--text)]">{rule.cidr}</p>
                        <p className="text-xs text-[rgba(20,20,20,0.55)]">
                          {rule.description || rule.label || '无描述'}
                        </p>
                      </div>
                    )}
                  </div>
                  {admin && !isEditing && (
                    <div className="flex gap-2">
                      <button
                        type="button"
                        onClick={() => beginEdit(rule)}
                        className="inline-flex items-center gap-1 rounded-[var(--radius-sm)] border border-[var(--line)] px-3 py-2 text-xs text-[var(--text)] transition hover:border-[var(--line-strong)]"
                      >
                        <PencilSquareIcon className="h-4 w-4" /> 编辑
                      </button>
                      <button
                        type="button"
                        onClick={() => handleDelete(rule.id)}
                        className="inline-flex items-center gap-1 rounded-[var(--radius-sm)] border border-red-200 px-3 py-2 text-xs text-red-600 transition hover:border-red-300"
                        disabled={deletingId === rule.id}
                      >
                        <TrashIcon className="h-4 w-4" />
                        {deletingId === rule.id ? '删除中…' : '删除'}
                      </button>
                    </div>
                  )}
                </div>
                {!isEditing && (
                  <p className="text-xs text-[rgba(20,20,20,0.45)]">
                    添加于 {new Date(rule.created_at).toLocaleString()}
                  </p>
                )}
              </div>
            );
          })}
      </div>
    </div>
  );
};

export default IPAllowlist;
