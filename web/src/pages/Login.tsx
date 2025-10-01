import { useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { LockClosedIcon } from '@heroicons/react/24/outline';

import api from '../api/client';
import { useSession } from '../hooks/useSession';

type SdidLoginIdentity = {
  did?: string;
  label?: string;
  roles?: string[];
  publicKeyJwk?: Record<string, unknown>;
};

type SdidProof = {
  signatureValue?: string;
};

type SdidAuthentication = {
  canonicalRequest?: string;
  payload?: unknown;
};

type SdidLoginResponse = {
  identity?: SdidLoginIdentity;
  challenge?: string;
  signature?: string;
  proof?: SdidProof;
  authentication?: SdidAuthentication;
  [key: string]: unknown;
};

type SdidRequestOptions = {
  message?: string;
  challenge?: string;
  forcePrompt?: boolean;
};

type SdidBridge = {
  requestLogin: (options: SdidRequestOptions) => Promise<SdidLoginResponse>;
};

declare global {
  interface Window {
    SDID?: SdidBridge;
  }
}

const getSdidBridge = (): SdidBridge | null => {
  if (typeof window === 'undefined') {
    return null;
  }
  const candidate: unknown = (window as any).SDID;
  if (candidate && typeof (candidate as SdidBridge).requestLogin === 'function') {
    return candidate as SdidBridge;
  }
  return null;
};

const waitForSdidBridge = (): Promise<SdidBridge> => {
  if (typeof window === 'undefined') {
    return Promise.reject(new Error('当前环境不支持 SDID 登录。'));
  }
  const immediate = getSdidBridge();
  if (immediate) {
    return Promise.resolve(immediate);
  }
  return new Promise<SdidBridge>((resolve, reject) => {
    const timeout = window.setTimeout(() => {
      window.removeEventListener('sdid#initialized' as any, handler as any);
      reject(new Error('未检测到 SDID 浏览器插件，请安装或启用扩展。'));
    }, 5000);
    const handler = () => {
      const bridge = getSdidBridge();
      if (!bridge) {
        return;
      }
      window.clearTimeout(timeout);
      window.removeEventListener('sdid#initialized' as any, handler as any);
      resolve(bridge);
    };
    window.addEventListener('sdid#initialized' as any, handler as any);
  });
};

const Login = () => {
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [sdidReady, setSdidReady] = useState(() => getSdidBridge() !== null);
  const navigate = useNavigate();
  const { setToken } = useSession();

  useEffect(() => {
    if (getSdidBridge()) {
      setSdidReady(true);
      return;
    }
    const handler = () => setSdidReady(true);
    window.addEventListener('sdid#initialized' as any, handler as any);
    return () => {
      window.removeEventListener('sdid#initialized' as any, handler as any);
    };
  }, []);

  const handleSdidLogin = async (forcePrompt: boolean) => {
    if (loading) {
      return;
    }
    setLoading(true);
    setError(null);
    try {
      const [{ data: nonceResp }, bridge] = await Promise.all([
        api.post<{ nonce: string; message?: string }>('/auth/request-nonce'),
        waitForSdidBridge()
      ]);
      setSdidReady(true);
      const response = await bridge.requestLogin({
        message: nonceResp.message || 'RoundOne Ledger 请求访问',
        challenge: nonceResp.nonce,
        forcePrompt
      });
      const { data } = await api.post('/auth/login', {
        nonce: nonceResp.nonce,
        response
      });
      setToken(data.token);
      navigate('/dashboard');
    } catch (err) {
      console.error(err);
      const serverError = (err as any)?.response?.data?.error;
      if (typeof serverError === 'string' && serverError.trim()) {
        setError(serverError.trim());
      } else {
        const message =
          err instanceof Error
            ? err.message.trim()
            : typeof err === 'string'
            ? err
            : '';
        if (message) {
          const lowered = message.toLowerCase();
          if (lowered.includes('cancel') || message.includes('拒绝') || message.includes('取消')) {
            setError('SDID 登录请求已被取消。');
          } else {
            setError(message);
          }
        } else {
          setError('SDID 登录失败，请确认插件已启用并允许此站点。');
        }
      }
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
            <p className="text-sm text-night-200">
              使用 SDID 钱包进行一键验证，无需注册，身份审批和管理由 SDID 完成。
            </p>
          </div>
        </div>

        <div className="space-y-5">
          <div className="rounded-2xl bg-night-900/30 px-4 py-3 text-xs text-night-200">
            <p>点击下方任意按钮后，浏览器会调用 SDID 插件发起登录并返回签名。</p>
            <p className="mt-2 text-[11px] uppercase tracking-wide text-night-400">
              {sdidReady ? 'SDID 插件已就绪，点击即可登录。' : '正在等待 SDID 插件连接…'}
            </p>
          </div>
          {error && <p className="rounded-2xl bg-red-100 px-4 py-3 text-sm text-red-500">{error}</p>}
          <div className="grid gap-3 sm:grid-cols-2">
            <button
              type="button"
              className="button-primary w-full"
              onClick={() => handleSdidLogin(false)}
              disabled={loading}
            >
              {loading ? '签名验证中…' : '连接 SDID 登录'}
            </button>
            <button
              type="button"
              className="w-full rounded-2xl border border-night-200 bg-white/80 px-4 py-3 text-sm font-medium text-night-500 shadow transition hover:bg-white focus:outline-none focus:ring-2 focus:ring-neon-500/60 disabled:cursor-not-allowed disabled:opacity-60"
              onClick={() => handleSdidLogin(true)}
              disabled={loading}
            >
              {loading ? '处理中…' : '强制弹窗验证'}
            </button>
          </div>
        </div>
      </div>
    </div>
  );
};

export default Login;
