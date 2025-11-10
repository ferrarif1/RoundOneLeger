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
    <div className="flex min-h-screen items-center justify-center bg-gradient-to-br from-blue-50 to-indigo-100 p-4">
      <div className="w-full max-w-md">
        <div className="rounded-2xl bg-white/90 p-8 shadow-xl backdrop-blur-sm">
          <div className="mb-8 text-center">
            <div className="mx-auto mb-4 flex h-12 w-12 items-center justify-center rounded-full bg-indigo-100">
              <LockClosedIcon className="h-6 w-6 text-indigo-600" />
            </div>
            <h1 className="text-2xl font-bold text-gray-900">台账系统</h1>
            <p className="mt-2 text-sm text-gray-600">登录以继续访问</p>
          </div>

          <form onSubmit={handleSubmit} className="space-y-6">
            {status && (
              <div className="rounded-lg bg-green-50 p-4">
                <div className="flex">
                  <div className="flex-shrink-0">
                    <CheckCircleIcon className="h-5 w-5 text-green-400" />
                  </div>
                  <div className="ml-3">
                    <p className="text-sm text-green-700">{status}</p>
                  </div>
                </div>
              </div>
            )}

            {error && (
              <div className="rounded-lg bg-red-50 p-4">
                <div className="flex">
                  <div className="flex-shrink-0">
                    <ExclamationCircleIcon className="h-5 w-5 text-red-400" />
                  </div>
                  <div className="ml-3">
                    <p className="text-sm text-red-700">{error}</p>
                  </div>
                </div>
              </div>
            )}

            <div>
              <label htmlFor="username" className="block text-sm font-medium text-gray-700">
                用户名
              </label>
              <div className="mt-1">
                <input
                  id="username"
                  name="username"
                  type="text"
                  autoComplete="username"
                  required
                  value={username}
                  onChange={(e) => setUsername(e.target.value)}
                  className="block w-full rounded-md border border-gray-300 px-3 py-2 shadow-sm focus:border-indigo-500 focus:outline-none focus:ring-indigo-500 sm:text-sm"
                />
              </div>
            </div>

            <div>
              <label htmlFor="password" className="block text-sm font-medium text-gray-700">
                密码
              </label>
              <div className="mt-1">
                <input
                  id="password"
                  name="password"
                  type="password"
                  autoComplete="current-password"
                  required
                  value={password}
                  onChange={(e) => setPassword(e.target.value)}
                  className="block w-full rounded-md border border-gray-300 px-3 py-2 shadow-sm focus:border-indigo-500 focus:outline-none focus:ring-indigo-500 sm:text-sm"
                />
              </div>
            </div>

            <div>
              <button
                type="submit"
                disabled={loading}
                className="flex w-full justify-center rounded-md border border-transparent bg-indigo-600 py-2 px-4 text-sm font-medium text-white shadow-sm hover:bg-indigo-700 focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:ring-offset-2 disabled:opacity-50"
              >
                {loading ? '登录中…' : '登录'}
              </button>
            </div>
          </form>
        </div>
      </div>
    </div>
  );
};

export default Login;