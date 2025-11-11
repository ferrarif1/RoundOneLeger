import { FormEvent, useEffect, useState } from 'react';
import api from '../api/client';
import { PencilSquareIcon, ShieldCheckIcon, TrashIcon } from '@heroicons/react/24/outline';

interface AllowRule {
  id: string;
  label?: string;
  cidr: string;
  description?: string;
  created_at: string;
  updated_at: string;
}

const IPAllowlist = () => {
  const [rules, setRules] = useState<AllowRule[]>([]);
  const [label, setLabel] = useState('');
  const [cidr, setCidr] = useState('');
  const [description, setDescription] = useState('');
  const [editingId, setEditingId] = useState<string | null>(null);
  const [editingFields, setEditingFields] = useState({ label: '', cidr: '', description: '' });

  useEffect(() => {
    (async () => {
      const { data } = await api.get('/api/v1/ip-allowlist');
      setRules(Array.isArray(data.items) ? data.items : []);
    })();
  }, []);

  const handleCreate = async (event: FormEvent) => {
    event.preventDefault();
    const { data } = await api.post('/api/v1/ip-allowlist', { label, cidr, description });
    setRules((prev) => [data, ...prev]);
    setLabel('');
    setCidr('');
    setDescription('');
  };

  const handleStartEdit = (rule: AllowRule) => {
    setEditingId(rule.id);
    setEditingFields({
      label: rule.label ?? '',
      cidr: rule.cidr,
      description: rule.description ?? '',
    });
  };

  const resetEditing = () => {
    setEditingId(null);
    setEditingFields({ label: '', cidr: '', description: '' });
  };

  const handleUpdate = async (event: FormEvent) => {
    event.preventDefault();
    if (!editingId) {
      return;
    }
    const { data } = await api.put(`/api/v1/ip-allowlist/${editingId}`, {
      label: editingFields.label,
      cidr: editingFields.cidr,
      description: editingFields.description,
    });
    setRules((prev) => prev.map((rule) => (rule.id === data.id ? data : rule)));
    resetEditing();
  };

  const handleDelete = async (id: string) => {
    if (!window.confirm('确定要删除该白名单条目吗？')) {
      return;
    }
    await api.delete(`/api/v1/ip-allowlist/${id}`);
    setRules((prev) => prev.filter((rule) => rule.id !== id));
    if (editingId === id) {
      resetEditing();
    }
  };

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h2 className="section-title">IP 白名单</h2>
      </div>
      <form
        onSubmit={handleCreate}
        className="grid gap-4 rounded-2xl border border-[var(--line)] bg-white p-6 shadow-[0_20px_36px_rgba(0,0,0,0.08)] md:grid-cols-[1.5fr,1.5fr,1.5fr,auto]"
      >
        <div>
          <label className="text-xs uppercase tracking-wider text-[rgba(20,20,20,0.45)]">名称</label>
          <input
            value={label}
            onChange={(e) => setLabel(e.target.value)}
            placeholder="办公网段"
            className="mt-2 w-full border border-[var(--line)] bg-white px-4 py-3 text-sm"
          />
        </div>
        <div>
          <label className="text-xs uppercase tracking-wider text-[rgba(20,20,20,0.45)]">CIDR</label>
          <input
            value={cidr}
            onChange={(e) => setCidr(e.target.value)}
            placeholder="192.168.1.0/24"
            required
            className="mt-2 w-full border border-[var(--line)] bg-white px-4 py-3 text-sm"
          />
        </div>
        <div>
          <label className="text-xs uppercase tracking-wider text-[rgba(20,20,20,0.45)]">备注</label>
          <input
            value={description}
            onChange={(e) => setDescription(e.target.value)}
            placeholder="总部办公室"
            className="mt-2 w-full border border-[var(--line)] bg-white px-4 py-3 text-sm"
          />
        </div>
        <button type="submit" className="button-primary self-end">
          添加
        </button>
      </form>

      <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-3">
        {rules.map((rule) => (
          <div
            key={rule.id}
            className="rounded-2xl border border-[var(--line)] bg-white p-5 shadow-[0_20px_36px_rgba(0,0,0,0.08)]"
          >
            <div className="flex items-start justify-between gap-3">
              <div className="flex items-center gap-3">
                <div className="rounded-xl border border-[rgba(20,20,20,0.12)] bg-[var(--bg-subtle)] p-2">
                  <ShieldCheckIcon className="h-6 w-6 text-[var(--accent)]" />
                </div>
                <div>
                  <p className="text-sm font-semibold text-[var(--text)]">{rule.label || rule.cidr}</p>
                  <p className="text-xs text-[rgba(20,20,20,0.55)]">{rule.cidr}</p>
                </div>
              </div>
              <div className="flex items-center gap-2">
                <button
                  type="button"
                  onClick={() => handleStartEdit(rule)}
                  className="flex items-center gap-1 rounded-full border border-transparent px-2 py-1 text-xs font-medium text-[var(--accent)] transition hover:border-[var(--accent)]"
                >
                  <PencilSquareIcon className="h-4 w-4" /> 编辑
                </button>
                <button
                  type="button"
                  onClick={() => handleDelete(rule.id)}
                  className="flex items-center gap-1 rounded-full border border-transparent px-2 py-1 text-xs font-medium text-red-500 transition hover:border-red-200"
                >
                  <TrashIcon className="h-4 w-4" /> 删除
                </button>
              </div>
            </div>

            {editingId === rule.id ? (
              <form onSubmit={handleUpdate} className="mt-4 space-y-3">
                <div>
                  <label className="text-xs uppercase tracking-wider text-[rgba(20,20,20,0.45)]">名称</label>
                  <input
                    value={editingFields.label}
                    onChange={(e) => setEditingFields((prev) => ({ ...prev, label: e.target.value }))}
                    placeholder="办公网段"
                    className="mt-2 w-full border border-[var(--line)] bg-white px-4 py-3 text-sm"
                  />
                </div>
                <div>
                  <label className="text-xs uppercase tracking-wider text-[rgba(20,20,20,0.45)]">CIDR</label>
                  <input
                    value={editingFields.cidr}
                    onChange={(e) => setEditingFields((prev) => ({ ...prev, cidr: e.target.value }))}
                    placeholder="192.168.1.0/24"
                    required
                    className="mt-2 w-full border border-[var(--line)] bg-white px-4 py-3 text-sm"
                  />
                </div>
                <div>
                  <label className="text-xs uppercase tracking-wider text-[rgba(20,20,20,0.45)]">备注</label>
                  <input
                    value={editingFields.description}
                    onChange={(e) => setEditingFields((prev) => ({ ...prev, description: e.target.value }))}
                    placeholder="总部办公室"
                    className="mt-2 w-full border border-[var(--line)] bg-white px-4 py-3 text-sm"
                  />
                </div>
                <div className="flex items-center justify-end gap-3 pt-2">
                  <button
                    type="button"
                    onClick={resetEditing}
                    className="text-sm font-medium text-[rgba(20,20,20,0.55)] hover:text-[var(--text)]"
                  >
                    取消
                  </button>
                  <button type="submit" className="button-primary px-6 py-2 text-sm">
                    保存
                  </button>
                </div>
              </form>
            ) : (
              <>
                <p className="mt-4 text-xs text-[rgba(20,20,20,0.45)]">{rule.description || '无描述'}</p>
                <p className="mt-2 text-xs text-[rgba(20,20,20,0.45)]">
                  添加于 {new Date(rule.created_at).toLocaleString()}
                </p>
                <p className="text-xs text-[rgba(20,20,20,0.45)]">
                  最近更新 {new Date(rule.updated_at).toLocaleString()}
                </p>
              </>
            )}
          </div>
        ))}
      </div>
    </div>
  );
};

export default IPAllowlist;
