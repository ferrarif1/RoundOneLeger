import { FormEvent, useState } from 'react';
import { Link, useNavigate } from 'react-router-dom';
import api from '../api/client';
import { fingerprintHash } from '../utils/fingerprint';
import { useSession } from '../hooks/useSession';
import { LockClosedIcon } from '@heroicons/react/24/outline';

const encoder = new TextEncoder();

const Login = () => {
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [email, setEmail] = useState('');
  const navigate = useNavigate();
  const { setToken } = useSession();

  const handleSubmit = async (event: FormEvent) => {
    event.preventDefault();
    setLoading(true);
    setError(null);

    try {
      const { data: nonceResp } = await api.post('/auth/request-nonce', { email });
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
        email,
        fingerprint: fp,
        signature: signatureB64
      });

      setToken(data.token);
      navigate('/dashboard');
    } catch (err) {
      console.error(err);
      setError('认证失败，请确认指纹和私钥签名。');
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
        <form onSubmit={handleSubmit} className="space-y-5">
          <div>
            <label className="text-sm text-night-200">邮箱</label>
            <input
              type="email"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              className="mt-2 w-full rounded-2xl border border-night-600 bg-night-900 px-4 py-3 text-sm focus:border-neon-500 focus:ring-neon-500/40"
              placeholder="admin@ledger"
              required
            />
          </div>
          {error && <p className="rounded-2xl bg-red-100 px-4 py-3 text-sm text-red-500">{error}</p>}
          <button type="submit" className="button-primary w-full" disabled={loading}>
            {loading ? '登录中...' : '登录'}
          </button>
        </form>
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
