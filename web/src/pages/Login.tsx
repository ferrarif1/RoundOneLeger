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
      // 保存token到localStorage
      localStorage.setItem('ledger.token', data.token);
      localStorage.setItem('ledger.username', data.username);
      localStorage.setItem('ledger.admin', data.admin ? 'true' : 'false');
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
    <div className="flex min-h-screen flex-col bg-[var(--bg)] text-[var(--text)]">
      <div className="flex flex-1 flex-col lg:flex-row">
        <section className="flex flex-col justify-between gap-10 border-b border-[var(--line)]/50 bg-[var(--bg-subtle)] px-6 py-10 text-sm leading-6 shadow-[0_18px_34px_rgba(0,0,0,0.04)] lg:w-[380px] lg:border-b-0 lg:border-r lg:px-10 lg:py-14 lg:shadow-none">
          <div>
            <p className="text-xs uppercase tracking-[0.28em] text-[rgba(20,20,20,0.45)]">HZDSZ LEDGER</p>
            <h1 className="mt-4 text-3xl font-semibold text-[var(--text)]">欢迎回来</h1>
            <p className="mt-4 text-[var(--muted)]">
              集成文档、表格与审批的合规台账工作区，帮助团队高效完成跨部门协作与监管对接。
            </p>
          </div>
          <div className="grid gap-4 text-[rgba(20,20,20,0.65)]">
            <div className="rounded-[var(--radius-md)] border border-[var(--line)]/60 bg-white/70 p-4 shadow-[var(--shadow-sm)]">
              <p className="font-medium text-[var(--text)]">实时协同</p>
              <p className="mt-2">多人在线共建台账与文档，自动记录每次更新。</p>
            </div>
            <div className="rounded-[var(--radius-md)] border border-[var(--line)]/60 bg-white/70 p-4 shadow-[var(--shadow-sm)]">
              <p className="font-medium text-[var(--text)]">安全可追溯</p>
              <p className="mt-2">链式审计日志覆盖导入、审批、发布等关键动作。</p>
            </div>
          </div>
        </section>
        <section className="flex flex-1 items-center justify-center px-6 py-10 lg:px-16">
          <div className="w-full max-w-lg rounded-[var(--radius-lg)] border border-[var(--line)] bg-white/95 p-8 shadow-[var(--shadow-soft)] sm:p-10">
            <div className="mb-8 space-y-3">
              <div className="flex items-center gap-3">
                <div className="flex h-11 w-11 items-center justify-center rounded-full border border-[var(--line)] bg-[var(--bg-subtle)]">
                  <LockClosedIcon className="h-5 w-5 text-[var(--text)]" />
                </div>
                <div>
                  <p className="text-xs uppercase tracking-[0.24em] text-[rgba(20,20,20,0.45)]">Account Access</p>
                  <h2 className="text-xl font-semibold text-[var(--text)]">登录您的工作台</h2>
                </div>
              </div>
              <p className="text-sm text-[var(--muted)]">默认管理员账号为 hzdsz_admin，可在登录后修改密码。</p>
            </div>

            <form onSubmit={handleSubmit} className="space-y-6">
              {status && (
                <div className="flex items-center gap-3 rounded-[var(--radius-md)] border border-emerald-200 bg-emerald-50/90 px-4 py-3 text-sm text-emerald-700">
                  <CheckCircleIcon className="h-5 w-5 text-emerald-500" />
                  <span>{status}</span>
                </div>
              )}

              {error && (
                <div className="flex items-center gap-3 rounded-[var(--radius-md)] border border-red-200 bg-red-50/90 px-4 py-3 text-sm text-red-700">
                  <ExclamationCircleIcon className="h-5 w-5 text-red-500" />
                  <span>{error}</span>
                </div>
              )}

              <div className="space-y-2">
                <label htmlFor="username" className="text-sm font-medium tracking-[0.02em] text-[var(--text)]">
                  用户名
                </label>
                <input
                  id="username"
                  name="username"
                  type="text"
                  autoComplete="username"
                  required
                  value={username}
                  onChange={(e) => setUsername(e.target.value)}
                  className="w-full rounded-[var(--radius-md)] border border-[var(--line)] bg-white/90 px-4 py-3 text-sm text-[var(--text)] shadow-sm transition focus:border-[var(--line-strong)] focus:outline-none focus:ring-0"
                />
              </div>

              <div className="space-y-2">
                <label htmlFor="password" className="text-sm font-medium tracking-[0.02em] text-[var(--text)]">
                  密码
                </label>
                <input
                  id="password"
                  name="password"
                  type="password"
                  autoComplete="current-password"
                  required
                  value={password}
                  onChange={(e) => setPassword(e.target.value)}
                  className="w-full rounded-[var(--radius-md)] border border-[var(--line)] bg-white/90 px-4 py-3 text-sm text-[var(--text)] shadow-sm transition focus:border-[var(--line-strong)] focus:outline-none focus:ring-0"
                />
              </div>

              <button type="submit" disabled={loading} className="button-primary w-full justify-center py-3">
                {loading ? '登录中…' : '登录'}
              </button>
            </form>
          </div>
        </section>
      </div>
    </div>
  );
};

export default Login;