import { FormEvent, useCallback, useEffect, useMemo, useState } from 'react';
import type { AxiosError } from 'axios';
import { CheckCircleIcon, ExclamationCircleIcon, PlusIcon, TrashIcon } from '@heroicons/react/24/outline';

import api from '../api/client';
import { useSession } from '../hooks/useSession';

interface UserRecord {
  id: string;
  username: string;
  admin: boolean;
  createdAt?: string;
  updatedAt?: string;
}

interface UserListResponse {
  items?: UserRecord[];
}

interface CreateUserResponse {
  user?: UserRecord;
}

const formatTimestamp = (value?: string) => {
  if (!value) {
    return '—';
  }
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return '—';
  }
  return new Intl.DateTimeFormat('zh-CN', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit'
  }).format(date);
};

const MIN_PASSWORD_LENGTH = 8;

const Users = () => {
  const { username: currentUsername } = useSession();
  const [users, setUsers] = useState<UserRecord[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [status, setStatus] = useState<string | null>(null);
  const [formUsername, setFormUsername] = useState('');
  const [formPassword, setFormPassword] = useState('');
  const [formAdmin, setFormAdmin] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [deletingId, setDeletingId] = useState<string | null>(null);

  const loadUsers = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const { data } = await api.get<UserListResponse>('/api/v1/users');
      setUsers(data.items ?? []);
    } catch (err) {
      const axiosError = err as AxiosError<{ error?: string }>;
      setError(axiosError.response?.data?.error || axiosError.message || '加载用户列表失败，请稍后重试。');
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    loadUsers().catch((err) => console.error(err));
  }, [loadUsers]);

  const resetForm = () => {
    setFormUsername('');
    setFormPassword('');
    setFormAdmin(false);
  };

  const handleSubmit = async (event: FormEvent) => {
    event.preventDefault();
    setStatus(null);
    setError(null);
    if (!formUsername.trim()) {
      setError('请输入用户名。');
      return;
    }
    if (!formPassword || formPassword.length < MIN_PASSWORD_LENGTH) {
      setError(`请输入至少 ${MIN_PASSWORD_LENGTH} 位的登录密码。`);
      return;
    }
    setSubmitting(true);
    try {
      const { data } = await api.post<CreateUserResponse>('/api/v1/users', {
        username: formUsername.trim(),
        password: formPassword,
        admin: formAdmin
      });
      const created = data.user;
      if (created) {
        setUsers((prev) => [...prev, created]);
      } else {
        await loadUsers();
      }
      setStatus(`已成功创建用户 ${formUsername.trim()}。`);
      resetForm();
    } catch (err) {
      const axiosError = err as AxiosError<{ error?: string }>;
      setError(axiosError.response?.data?.error || axiosError.message || '新增用户失败，请稍后重试。');
    } finally {
      setSubmitting(false);
    }
  };

  const handleDelete = async (id: string, username: string) => {
    if (!window.confirm(`确认删除用户 ${username}？该操作不可撤销。`)) {
      return;
    }
    setError(null);
    setStatus(null);
    setDeletingId(id);
    try {
      await api.delete(`/api/v1/users/${id}`);
      setUsers((prev) => prev.filter((user) => user.id !== id));
      setStatus(`已删除用户 ${username}。`);
    } catch (err) {
      const axiosError = err as AxiosError<{ error?: string }>;
      setError(axiosError.response?.data?.error || axiosError.message || '删除用户失败，请稍后再试。');
    } finally {
      setDeletingId(null);
    }
  };

  const sortedUsers = useMemo(() => {
    return [...users].sort((a, b) => (a.createdAt || '').localeCompare(b.createdAt || ''));
  }, [users]);

  return (
    <div className="space-y-8">
      <section className="rounded-[var(--radius-lg)] border border-black/5 bg-white/90 p-6 shadow-[var(--shadow-soft)]">
        <header className="mb-4 flex items-center justify-between">
          <div>
            <h2 className="text-lg font-semibold text-[var(--text)]">新增用户</h2>
            <p className="mt-1 text-xs text-[var(--muted)]">
              仅管理员可创建用户，默认密码需不少于 {MIN_PASSWORD_LENGTH} 位。
            </p>
          </div>
        </header>
        <form onSubmit={handleSubmit} className="grid gap-4 md:grid-cols-3">
          <label className="flex flex-col text-sm text-[var(--text)]">
            <span className="mb-1 text-xs text-[var(--muted)]">用户名</span>
            <input
              value={formUsername}
              onChange={(event) => setFormUsername(event.target.value)}
              placeholder="hzdsz_staff"
              className="rounded-[var(--radius-md)] border border-black/10 bg-white/90 px-3 py-2 text-sm text-[var(--text)] focus:border-[var(--accent)] focus:outline-none focus:ring-2 focus:ring-[var(--accent)]/20"
            />
          </label>
          <label className="flex flex-col text-sm text-[var(--text)]">
            <span className="mb-1 text-xs text-[var(--muted)]">登录密码</span>
            <input
              type="password"
              value={formPassword}
              onChange={(event) => setFormPassword(event.target.value)}
              placeholder="请输入至少 8 位密码"
              className="rounded-[var(--radius-md)] border border-black/10 bg-white/90 px-3 py-2 text-sm text-[var(--text)] focus:border-[var(--accent)] focus:outline-none focus:ring-2 focus:ring-[var(--accent)]/20"
            />
          </label>
          <label className="flex items-center gap-2 text-sm text-[var(--text)]">
            <input
              type="checkbox"
              checked={formAdmin}
              onChange={(event) => setFormAdmin(event.target.checked)}
              className="h-4 w-4 rounded border border-black/10 text-[var(--accent)] focus:ring-[var(--accent)]"
            />
            赋予管理员权限
          </label>
          <div className="md:col-span-3 flex items-center justify-end gap-3 text-sm">
            <button
              type="submit"
              disabled={submitting}
              className="flex items-center gap-2 rounded-full bg-[var(--accent)] px-5 py-2 text-sm font-semibold text-white shadow-[var(--shadow-sm)] transition hover:bg-[var(--accent-2)] disabled:cursor-not-allowed"
            >
              <PlusIcon className="h-4 w-4" />
              {submitting ? '创建中…' : '创建用户'}
            </button>
          </div>
        </form>
        {error && (
          <div className="mt-4 flex items-center gap-2 rounded-[var(--radius-md)] bg-red-50 px-3 py-2 text-sm text-red-600">
            <ExclamationCircleIcon className="h-5 w-5" />
            <span>{error}</span>
          </div>
        )}
        {status && !error && (
          <div className="mt-4 flex items-center gap-2 rounded-[var(--radius-md)] bg-green-50 px-3 py-2 text-sm text-green-600">
            <CheckCircleIcon className="h-5 w-5" />
            <span>{status}</span>
          </div>
        )}
      </section>

      <section className="rounded-[var(--radius-lg)] border border-black/5 bg-white/90 p-6 shadow-[var(--shadow-soft)]">
        <header className="mb-4 flex items-center justify-between">
          <div>
            <h2 className="text-lg font-semibold text-[var(--text)]">用户列表</h2>
            <p className="mt-1 text-xs text-[var(--muted)]">共 {users.length} 个账户</p>
          </div>
        </header>
        {loading ? (
          <div className="rounded-[var(--radius-md)] border border-dashed border-black/10 bg-white/70 p-6 text-center text-sm text-[var(--muted)]">
            正在加载用户信息…
          </div>
        ) : sortedUsers.length ? (
          <div className="overflow-x-auto">
            <table className="min-w-full text-sm text-[var(--text)]">
              <thead className="border-b border-black/5 text-left text-xs uppercase tracking-wide text-[var(--muted)]">
                <tr>
                  <th className="px-3 py-2">用户名</th>
                  <th className="px-3 py-2">角色</th>
                  <th className="px-3 py-2">创建时间</th>
                  <th className="px-3 py-2">最近更新</th>
                  <th className="px-3 py-2 text-right">操作</th>
                </tr>
              </thead>
              <tbody>
                {sortedUsers.map((user) => {
                  const disableDelete = user.username === currentUsername || user.username === 'hzdsz_admin';
                  return (
                    <tr key={user.id} className="border-b border-black/5 last:border-none">
                      <td className="px-3 py-3 font-medium">{user.username}</td>
                      <td className="px-3 py-3 text-[var(--muted)]">{user.admin ? '管理员' : '普通用户'}</td>
                      <td className="px-3 py-3 text-[var(--muted)]">{formatTimestamp(user.createdAt)}</td>
                      <td className="px-3 py-3 text-[var(--muted)]">{formatTimestamp(user.updatedAt)}</td>
                      <td className="px-3 py-3 text-right">
                        <button
                          type="button"
                          onClick={() => handleDelete(user.id, user.username)}
                          disabled={disableDelete || deletingId === user.id}
                          className="inline-flex items-center gap-2 rounded-full border border-red-200 px-3 py-1.5 text-xs text-red-500 transition hover:bg-red-50 disabled:cursor-not-allowed disabled:opacity-60"
                        >
                          <TrashIcon className="h-4 w-4" />
                          删除
                        </button>
                      </td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          </div>
        ) : (
          <div className="rounded-[var(--radius-md)] border border-dashed border-black/10 bg-white/70 p-6 text-center text-sm text-[var(--muted)]">
            暂无用户，请使用上方表单新增。
          </div>
        )}
      </section>
    </div>
  );
};

export default Users;
