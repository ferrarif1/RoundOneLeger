import { FormEvent, useEffect, useState } from 'react';
import { Link, useNavigate } from 'react-router-dom';
import api from '../api/client';
import { fingerprintHash } from '../utils/fingerprint';
import { useSession } from '../hooks/useSession';
import { LockClosedIcon } from '@heroicons/react/24/outline';

const encoder = new TextEncoder();

const Login = () => {
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [mode, setMode] = useState<'password' | 'sdid'>('password');
  const [username, setUsername] = useState('');
  const [password, setPassword] = useState('');
  const [sdid, setSdid] = useState('');
  const navigate = useNavigate();
  const { setToken } = useSession();

  useEffect(() => {
    setError(null);
    setLoading(false);
  }, [mode]);

  const handlePasswordLogin = async (event: FormEvent) => {
    event.preventDefault();
    setLoading(true);
    setError(null);

    try {
      const { data } = await api.post('/auth/login-password', {
        username,
        password
      });
      setToken(data.token);
      navigate('/dashboard');
    } catch (err) {
      console.error(err);
      setError('账号或密码错误，请重试。');
    } finally {
      setLoading(false);
    }
  };

  const handleSDIDLogin = async (event: FormEvent) => {
    event.preventDefault();
    setLoading(true);
    setError(null);

    try {
      const { data: nonceResp } = await api.post('/auth/request-nonce', {
        username,
        sdid
      });
      const fp = await fingerprintHash();
      if (!(window as any).ledgerPrivateKey) {
        throw new Error('缺少私钥');
      }
      const signature = await window.crypto.subtle.sign(
        { name: 'NODE-ED25519' } as AlgorithmIdentifier,
        (window as any).ledgerPrivateKey,
        encoder.encode(nonceResp.nonce + fp)
      );
      const signatureB64 = btoa(String.fromCharCode(...new Uint8Array(signature)));

      const { data } = await api.post('/auth/login', {
        username,
        sdid,
        nonce: nonceResp.nonce,
        fingerprint: fp,
        signature: signatureB64
      });

      setToken(data.token);
      navigate('/dashboard');
    } catch (err) {
      console.error(err);
      setError('认证失败，请确认 SDID 与签名信息。');
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="flex min-h-screen items-center justify-center px-6 py-16">
      <div className="w-full max-w-md space-y-6 rounded-[32px] border border-white bg-white/90 p-10 shadow-glow">
        <div className="flex items-center gap-3">
          <div className="rounded-2xl bg-neon-500/15 p-3 text-neon-500">
            <LockClosedIcon className="h-7 w-7" />
          </div>
          <div>
            <h1 className="text-2xl font-semibold text-night-50">登录控制台</h1>
            <p className="text-sm text-night-200">延续样例中柔和的卡片式质感与留白</p>
          </div>
        </div>
        <div className="rounded-2xl bg-night-900/30 p-2">
          <div className="grid grid-cols-2 gap-2 rounded-2xl bg-night-900/40 p-1">
            <button
              type="button"
              onClick={() => setMode('password')}
              className={`rounded-xl px-4 py-2 text-sm transition ${
                mode === 'password'
                  ? 'bg-night-900 text-night-50 shadow-sm'
                  : 'text-night-300 hover:text-night-100'
              }`}
            >
              账号密码登录
            </button>
            <button
              type="button"
              onClick={() => setMode('sdid')}
              className={`rounded-xl px-4 py-2 text-sm transition ${
                mode === 'sdid'
                  ? 'bg-night-900 text-night-50 shadow-sm'
                  : 'text-night-300 hover:text-night-100'
              }`}
            >
              SDID 登录
            </button>
          </div>
        </div>
        {mode === 'password' ? (
          <form onSubmit={handlePasswordLogin} className="space-y-5">
            <div>
              <label className="text-sm text-night-200">账号</label>
              <input
                type="text"
                value={username}
                onChange={(e) => setUsername(e.target.value)}
                className="mt-2 w-full rounded-2xl border border-night-600 bg-night-900 px-4 py-3 text-sm focus:border-neon-500 focus:ring-neon-500/40"
                placeholder="admin"
                required
              />
            </div>
            <div>
              <label className="text-sm text-night-200">密码</label>
              <input
                type="password"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                className="mt-2 w-full rounded-2xl border border-night-600 bg-night-900 px-4 py-3 text-sm focus:border-neon-500 focus:ring-neon-500/40"
                placeholder="请输入密码"
                required
              />
            </div>
            {error && <p className="rounded-2xl bg-red-100 px-4 py-3 text-sm text-red-500">{error}</p>}
            <button type="submit" className="button-primary w-full" disabled={loading}>
              {loading ? '登录中...' : '登录'}
            </button>
          </form>
        ) : (
          <form onSubmit={handleSDIDLogin} className="space-y-5">
            <div>
              <label className="text-sm text-night-200">账号</label>
              <input
                type="text"
                value={username}
                onChange={(e) => setUsername(e.target.value)}
                className="mt-2 w-full rounded-2xl border border-night-600 bg-night-900 px-4 py-3 text-sm focus:border-neon-500 focus:ring-neon-500/40"
                placeholder="admin"
                required
              />
            </div>
            <div>
              <label className="text-sm text-night-200">SDID</label>
              <input
                type="text"
                value={sdid}
                onChange={(e) => setSdid(e.target.value)}
                className="mt-2 w-full rounded-2xl border border-night-600 bg-night-900 px-4 py-3 text-sm focus:border-neon-500 focus:ring-neon-500/40"
                placeholder="例如：device-xxxxxxxx"
                required
              />
            </div>
            {error && <p className="rounded-2xl bg-red-100 px-4 py-3 text-sm text-red-500">{error}</p>}
            <button type="submit" className="button-primary w-full" disabled={loading}>
              {loading ? '登录中...' : '登录'}
            </button>
          </form>
        )}
        <p className="text-xs text-night-300">
          第一次访问？
          <Link className="ml-2 text-neon-500" to="/enroll">
            立即注册设备
          </Link>
        </p>
      </div>
    </div>
  );
};

export default Login;
