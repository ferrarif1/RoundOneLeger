import { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import api from '../api/client';
import { useSession } from '../hooks/useSession';
import { LockClosedIcon } from '@heroicons/react/24/outline';

const Login = () => {
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const navigate = useNavigate();
  const { setToken } = useSession();

  const pickString = (source: any, keys: string[]): string | undefined => {
    if (!source || typeof source !== 'object') {
      return undefined;
    }
    for (const key of keys) {
      const value = source[key];
      if (typeof value === 'string' && value.trim().length > 0) {
        return value;
      }
    }
    return undefined;
  };

  const normalizeAccount = (raw: any): { sdid: string; publicKey: string } | null => {
    if (!raw) {
      return null;
    }
    if (Array.isArray(raw)) {
      for (const item of raw) {
        const normalized = normalizeAccount(item);
        if (normalized) {
          return normalized;
        }
      }
      return null;
    }
    const sdid =
      pickString(raw, ['sdid', 'did', 'identity', 'address', 'id']) ||
      (typeof raw === 'string' ? raw : undefined);
    const publicKey = pickString(raw, ['publicKey', 'public_key', 'pubKey', 'pub_key', 'publickey', 'key']);

    if (sdid && publicKey) {
      return { sdid, publicKey };
    }

    if (raw.account) {
      return normalizeAccount(raw.account);
    }

    if (raw.data) {
      return normalizeAccount(raw.data);
    }

    if (raw.result) {
      return normalizeAccount(raw.result);
    }

    return null;
  };

  const normalizeSignature = (raw: any): { signature: string; sdid?: string; publicKey?: string } | null => {
    if (!raw) {
      return null;
    }
    if (typeof raw === 'string') {
      return { signature: raw };
    }

    const signature = pickString(raw, ['signature', 'sig', 'signedMessage']);

    if (signature) {
      const account = normalizeAccount(raw);
      return {
        signature,
        sdid: account?.sdid,
        publicKey: account?.publicKey
      };
    }

    if (raw.result) {
      const normalized = normalizeSignature(raw.result);
      if (normalized) {
        return normalized;
      }
    }

    if (raw.data) {
      const normalized = normalizeSignature(raw.data);
      if (normalized) {
        return normalized;
      }
    }

    return null;
  };

  const getWallet = () => {
    const anyWindow = window as any;
    const candidates = [
      anyWindow.SDID,
      anyWindow.sdid,
      anyWindow.SDID?.wallet,
      anyWindow.sdid?.wallet,
      anyWindow.SDIDWallet,
      anyWindow.sdidWallet,
      anyWindow.wallets?.SDID,
      anyWindow.wallets?.sdid,
      anyWindow.SDID?.provider,
      anyWindow.sdid?.provider,
      anyWindow.sdidProvider,
      anyWindow.__SDID__
    ];

    for (const candidate of candidates) {
      if (!candidate) {
        continue;
      }
      if (typeof candidate === 'function') {
        return candidate;
      }
      if (candidate.wallet && typeof candidate.wallet === 'object') {
        return candidate.wallet;
      }
      if (candidate.provider && typeof candidate.provider === 'object') {
        return candidate.provider;
      }
      return candidate;
    }

    throw new Error('未检测到 SDID 浏览器插件，请安装或启用扩展。');
  };

  const resolveAccount = async () => {
    const wallet = getWallet();

    if (typeof wallet.getAccount === 'function') {
      const result = await wallet.getAccount();
      const normalized = normalizeAccount(result);
      if (normalized) {
        return normalized;
      }
    }

    if (wallet.account) {
      const normalized = normalizeAccount(wallet.account);
      if (normalized) {
        return normalized;
      }
    }

    if (typeof wallet.request === 'function') {
      const methods = ['sdid_getAccount', 'sdid_requestAccount', 'sdid_accounts'];
      for (const method of methods) {
        try {
          const result = await wallet.request({ method });
          const normalized = normalizeAccount(result);
          if (normalized) {
            return normalized;
          }
        } catch (err) {
          console.warn(`SDID wallet request for ${method} failed`, err);
        }
      }
    }

    throw new Error('无法获取 SDID 账号信息，请确认插件已登录。');
  };

  const signMessage = async (message: string) => {
    const wallet = getWallet();

    if (typeof wallet.signMessage === 'function') {
      const result = await wallet.signMessage(message);
      const normalized = normalizeSignature(result);
      if (normalized) {
        return normalized;
      }
    }

    if (typeof wallet.sign === 'function') {
      const result = await wallet.sign(message);
      const normalized = normalizeSignature(result);
      if (normalized) {
        return normalized;
      }
    }

    if (typeof wallet.request === 'function') {
      const requestPayloads = [
        { method: 'sdid_signMessage', params: { message } },
        { method: 'sdid_sign', params: { message } },
        { method: 'sdid_signMessage', params: [message] }
      ];

      for (const payload of requestPayloads) {
        try {
          const result = await wallet.request(payload);
          const normalized = normalizeSignature(result);
          if (normalized) {
            return normalized;
          }
        } catch (err) {
          console.warn('SDID wallet request sign failure', err);
        }
      }
    }

    throw new Error('未能完成 SDID 签名，请重试。');
  };

  const handleSdidLogin = async () => {
    if (loading) {
      return;
    }
    setLoading(true);

    try {
      const [{ data: nonceResp }, account] = await Promise.all([
        api.post('/auth/request-nonce'),
        resolveAccount()
      ]);
      const message = nonceResp.message || nonceResp.nonce;
      const signatureResult = await signMessage(message);
      const sdid = signatureResult.sdid || account.sdid;
      const publicKey = signatureResult.publicKey || account.publicKey;
      const signature = signatureResult.signature;
      if (!sdid || !publicKey || !signature) {
        throw new Error('SDID 插件未返回完整的账号或签名信息。');
      }
      const { data } = await api.post('/auth/login', {
        sdid,
        nonce: nonceResp.nonce,
        signature,
        public_key: publicKey
      });

      setToken(data.token);
      navigate('/dashboard');
    } catch (err) {
      console.error(err);
      setError(
        err instanceof Error ? err.message : 'SDID 登录失败，请确认插件已开启并在受信任网络中访问。'
      );
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
            <p className="text-sm text-night-200">使用 SDID 钱包进行一键验证，无需注册，身份管理由 SDID 负责。</p>
          </div>
        </div>

        <div className="space-y-5">
          <div className="rounded-2xl bg-night-900/30 px-4 py-3 text-xs text-night-200">
            点击下方按钮后，浏览器会请求 SDID 插件获取账号并对一次性登录消息进行签名。
          </div>
          {error && <p className="rounded-2xl bg-red-100 px-4 py-3 text-sm text-red-500">{error}</p>}
          <button type="button" className="button-primary w-full" onClick={handleSdidLogin} disabled={loading}>
            {loading ? '签名验证中...' : '使用 SDID 一键登录'}
          </button>
        </div>
      </div>
    </div>
  );
};

export default Login;
