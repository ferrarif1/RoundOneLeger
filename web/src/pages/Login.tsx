import { FormEvent, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import type { AxiosError } from 'axios';
import { CheckCircleIcon, ExclamationCircleIcon, LockClosedIcon } from '@heroicons/react/24/outline';

import api from '../api/client';
import { useSession } from '../hooks/useSession';

interface PasswordLoginResponse {
  token: string;
  username: string;
  admin?: boolean;
  issuedAt?: string;
  expiresAt?: string;
}

const Login = () => {
  const navigate = useNavigate();
  const { setToken } = useSession();
  const [username, setUsername] = useState('hzdsz_admin');
  const [password, setPassword] = useState('');
  const [status, setStatus] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);

  const handleSubmit = async (event: FormEvent) => {
    event.preventDefault();
    setError(null);
    setStatus(null);
    if (!username.trim()) {
      setError('请输入用户名。');
      return;
    }
    if (!password) {
      setError('请输入密码。');
      return;
    }
    setLoading(true);
    try {
      const { data } = await api.post<PasswordLoginResponse>('/auth/password-login', {
        username: username.trim(),
        password
      });
      setToken(data.token, data.username, Boolean(data.admin));
      setStatus('登录成功，正在跳转…');
      setTimeout(() => navigate('/dashboard', { replace: true }), 400);
    } catch (err) {
      const axiosError = err as AxiosError<{ error?: string }>;
      const message = axiosError.response?.data?.error || axiosError.message || '登录失败，请稍后重试。';
      if (message === 'invalid_credentials') {
        setError('用户名或密码错误。');
      } else {
        setError(message);
      }
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="flex min-h-screen items-center justify-center bg-[var(--bg)] px-4 py-12 text-[var(--text)]">
      <div className="w-full max-w-lg rounded-[var(--radius-xl)] border border-black/5 bg-white/95 p-10 shadow-[var(--shadow-soft)]">
        <div className="mb-8 flex items-center gap-3">
          <div className="rounded-2xl border border-black/10 bg-black/90 p-3 text-white">
            <LockClosedIcon className="h-6 w-6" />
          </div>
          <div>
            <h1 className="text-2xl font-semibold tracking-tight">RoundOneLeger 控制台</h1>
            <p className="mt-1 text-sm text-[var(--muted)]">使用管理员账户登录以管理台账与用户。</p>
          </div>
        </div>

        <div className="mb-6 rounded-[var(--radius-lg)] bg-[var(--bg-subtle)]/60 px-4 py-3 text-sm text-[var(--muted)]">
          默认管理员账号：<strong className="font-semibold text-[var(--text)]">hzdsz_admin</strong>，密码：
          <strong className="font-semibold text-[var(--text)]">Hzdsz@2025#</strong>。
        </div>

        <form onSubmit={handleSubmit} className="space-y-5">
          <label className="flex flex-col gap-2 text-sm">
            <span className="text-xs text-[var(--muted)]">用户名</span>
            <input
              value={username}
              onChange={(event) => setUsername(event.target.value)}
              autoComplete="username"
              className="rounded-[var(--radius-md)] border border-black/10 bg-white/90 px-4 py-2 text-sm text-[var(--text)] focus:border-[var(--accent)] focus:outline-none focus:ring-2 focus:ring-[var(--accent)]/20"
              placeholder="请输入用户名"
            />
          </label>
          <label className="flex flex-col gap-2 text-sm">
            <span className="text-xs text-[var(--muted)]">密码</span>
            <input
              type="password"
              value={password}
              onChange={(event) => setPassword(event.target.value)}
              autoComplete="current-password"
              className="rounded-[var(--radius-md)] border border-black/10 bg-white/90 px-4 py-2 text-sm text-[var(--text)] focus:border-[var(--accent)] focus:outline-none focus:ring-2 focus:ring-[var(--accent)]/20"
              placeholder="请输入密码"
            />
          </label>
          <button
            type="submit"
            disabled={loading}
            className="w-full rounded-full bg-black px-5 py-2.5 text-sm font-semibold text-white shadow-[var(--shadow-soft)] transition hover:bg-black/90 disabled:cursor-not-allowed"
          >
            {loading ? '登录中…' : '登录'}
          </button>
        </form>

        {error && (
          <div className="mt-6 flex items-center gap-2 rounded-[var(--radius-lg)] bg-red-50 px-4 py-3 text-sm text-red-600">
            <ExclamationCircleIcon className="h-5 w-5" />
            <span>{error}</span>
          </div>
        )}
        {status && !error && (
          <div className="mt-6 flex items-center gap-2 rounded-[var(--radius-lg)] bg-green-50 px-4 py-3 text-sm text-green-600">
            <CheckCircleIcon className="h-5 w-5" />
            <span>{status}</span>
          </div>
        )}
      </div>
    </div>
  );
};

export default Login;
