import { FormEvent, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import type { AxiosError } from 'axios';
import { CheckCircleIcon, ExclamationCircleIcon, LockClosedIcon } from '@heroicons/react/24/outline';

import api from '../api/client';
import { useSession } from '../hooks/useSession';
import '../styles/login.css';

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
    <div className="auth-shell roleger-ledger-root">
      <div className="auth-panel glass-panel">
        <header className="auth-header">
          <div className="auth-icon">
            <LockClosedIcon />
          </div>
          <div>
            <p className="auth-eyebrow">Roledger Ledger</p>
            <h1>登录台账系统</h1>
            <p className="auth-subtext">使用分配的账号密码完成认证</p>
          </div>
        </header>

        <form onSubmit={handleSubmit} className="auth-form">
          {status && (
            <div className="auth-alert auth-alert--success" role="status" aria-live="polite">
              <CheckCircleIcon />
              <p>{status}</p>
            </div>
          )}

          {error && (
            <div className="auth-alert auth-alert--error" role="alert" aria-live="assertive">
              <ExclamationCircleIcon />
              <p>{error}</p>
            </div>
          )}

          <label className="auth-field">
            <span>用户名</span>
            <input
              id="username"
              name="username"
              type="text"
              autoComplete="username"
              required
              value={username}
              onChange={(e) => setUsername(e.target.value)}
            />
          </label>

          <label className="auth-field">
            <span>密码</span>
            <input
              id="password"
              name="password"
              type="password"
              autoComplete="current-password"
              required
              value={password}
              onChange={(e) => setPassword(e.target.value)}
            />
          </label>

          <button
            type="submit"
            disabled={loading}
            className="button-primary auth-submit"
            aria-busy={loading ? true : undefined}
          >
            {loading ? '登录中…' : '登录'}
          </button>
        </form>
        <footer className="auth-meta">
          <p>仅限授权用户访问，所有操作都会被记录以保障合规。</p>
          <span>Roledger Ledger · 内部版本</span>
        </footer>
      </div>
    </div>
  );
};

export default Login;