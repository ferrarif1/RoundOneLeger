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
  identityId?: string;
  forcePrompt?: boolean;
};

type SdidBridge = {
  requestLogin: (options: SdidRequestOptions) => Promise<SdidLoginResponse>;
};

type SdidInitDetail = {
  provider?: unknown;
  SDID?: unknown;
  sdid?: unknown;
};

type StatusTone = 'info' | 'success' | 'error';

type StatusState = {
  message: string;
  tone: StatusTone;
};

type VerificationState = {
  message: string;
  success: boolean;
} | null;

declare global {
  interface Window {
    SDID?: SdidBridge;
    sdid?: SdidBridge;
    wallet?: { sdid?: SdidBridge };
  }
}

const isBridge = (candidate: unknown): candidate is SdidBridge => {
  return Boolean(candidate && typeof (candidate as SdidBridge).requestLogin === 'function');
};

const promoteBridge = (candidate: unknown): SdidBridge | null => {
  if (!isBridge(candidate)) {
    return null;
  }
  try {
    window.SDID = candidate;
  } catch (error) {
    console.warn('Unable to promote SDID provider', error);
  }
  return candidate;
};

const bridgeFromDetail = (detail: SdidInitDetail | null | undefined): SdidBridge | null => {
  if (!detail || typeof detail !== 'object') {
    return null;
  }
  return promoteBridge(detail.provider || detail.SDID || detail.sdid);
};

const getSdidBridge = (): SdidBridge | null => {
  if (typeof window === 'undefined') {
    return null;
  }
  return (
    promoteBridge(window.SDID) ||
    promoteBridge(window.sdid) ||
    promoteBridge(window.wallet?.sdid)
  );
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
    const dispose = () => {
      window.removeEventListener('sdid#initialized', handleInitialized as EventListener);
      window.removeEventListener('sdid#ready', handleInitialized as EventListener);
      document.removeEventListener('sdid#initialized', handleInitialized as EventListener);
      document.removeEventListener('sdid#ready', handleInitialized as EventListener);
    };

    const timeout = window.setTimeout(() => {
      dispose();
      reject(new Error('未检测到 SDID 浏览器插件，请安装或启用扩展。'));
    }, 8000);

    function handleInitialized(event: Event) {
      const detail = (event as CustomEvent<SdidInitDetail>).detail;
      const bridge = bridgeFromDetail(detail) || getSdidBridge();
      if (!bridge) {
        return;
      }
      window.clearTimeout(timeout);
      dispose();
      resolve(bridge);
    }

    window.addEventListener('sdid#initialized', handleInitialized as EventListener);
    window.addEventListener('sdid#ready', handleInitialized as EventListener);
    document.addEventListener('sdid#initialized', handleInitialized as EventListener);
    document.addEventListener('sdid#ready', handleInitialized as EventListener);
  });
};

const isPlainObject = (value: unknown): value is Record<string, unknown> => {
  return typeof value === 'object' && value !== null && !Array.isArray(value);
};

const canonicalizeJson = (value: unknown): string => {
  if (value === null) {
    return 'null';
  }
  if (typeof value === 'string') {
    return JSON.stringify(value);
  }
  if (typeof value === 'number') {
    if (!Number.isFinite(value)) {
      return 'null';
    }
    return String(value);
  }
  if (typeof value === 'boolean') {
    return value ? 'true' : 'false';
  }
  if (Array.isArray(value)) {
    return `[${value.map((item) => canonicalizeJson(item)).join(',')}]`;
  }
  if (isPlainObject(value)) {
    const entries = Object.entries(value)
      .filter(([, v]) => v !== undefined)
      .sort(([a], [b]) => a.localeCompare(b));
    return `{${entries
      .map(([key, item]) => `${JSON.stringify(key)}:${canonicalizeJson(item)}`)
      .join(',')}}`;
  }
  return JSON.stringify(value ?? null);
};

const normalizeBase64 = (value: string): string => {
  const compact = value.replace(/\s+/g, '').replace(/-/g, '+').replace(/_/g, '/');
  const padding = compact.length % 4;
  if (padding === 0) {
    return compact;
  }
  return compact + '='.repeat(4 - padding);
};

const decodeSignature = (value: string): Uint8Array => {
  const normalized = normalizeBase64(value);
  const binary = atob(normalized);
  const bytes = new Uint8Array(binary.length);
  for (let i = 0; i < binary.length; i += 1) {
    bytes[i] = binary.charCodeAt(i);
  }
  return bytes;
};

const importPublicKey = async (identity: SdidLoginIdentity): Promise<CryptoKey> => {
  if (!identity?.publicKeyJwk) {
    throw new Error('SDID 未返回公钥。');
  }
  const jwk = identity.publicKeyJwk as JsonWebKey;
  try {
    return await crypto.subtle.importKey(
      'jwk',
      jwk,
      { name: 'ECDSA', namedCurve: 'P-256' },
      false,
      ['verify']
    );
  } catch (error) {
    console.error('Unable to import SDID JWK', error);
    throw new Error('无法导入 SDID 公钥。');
  }
};

const resolveSignedPayload = (response: SdidLoginResponse): string => {
  const authentication = response.authentication;
  if (authentication) {
    const { canonicalRequest = '', payload } = authentication;
    let signed = canonicalRequest.trim();
    if (payload !== undefined) {
      try {
        const canonical = canonicalizeJson(payload);
        if (signed && canonical && signed !== canonical) {
          throw new Error('authentication_mismatch');
        }
        if (!signed) {
          signed = canonical;
        }
      } catch (error) {
        console.error('Unable to canonicalize authentication payload', error);
        throw new Error('无法解析 SDID 返回的认证信息。');
      }
    }
    if (signed) {
      return signed;
    }
  }
  if (response.challenge && response.challenge.trim()) {
    return response.challenge.trim();
  }
  throw new Error('缺少用于验证的 challenge。');
};

const verifySdidResponse = async (
  response: SdidLoginResponse
): Promise<VerificationState> => {
  if (typeof window === 'undefined' || !window.crypto?.subtle) {
    return { message: '当前浏览器不支持 WebCrypto，无法验证签名。', success: false };
  }
  const identity = response.identity;
  if (!identity) {
    return { message: 'SDID 未返回身份信息。', success: false };
  }
  const signatureValue = response.proof?.signatureValue || response.signature;
  if (!signatureValue || !signatureValue.trim()) {
    return { message: 'SDID 返回数据缺少签名。', success: false };
  }
  let signedData: string;
  try {
    signedData = resolveSignedPayload(response);
  } catch (error) {
    return { message: error instanceof Error ? error.message : '无法解析签名内容。', success: false };
  }

  try {
    const publicKey = await importPublicKey(identity);
    const signatureBytes = decodeSignature(signatureValue);
    const encoder = new TextEncoder();
    const verified = await crypto.subtle.verify(
      { name: 'ECDSA', hash: { name: 'SHA-256' } },
      publicKey,
      signatureBytes,
      encoder.encode(signedData)
    );
    if (!verified) {
      return { message: '签名验证失败，请重新尝试。', success: false };
    }
  } catch (error) {
    const message =
      error instanceof Error && error.message
        ? error.message
        : '验证 SDID 签名时出现错误。';
    return { message, success: false };
  }

  return { message: '签名验证通过。', success: true };
};

const formatIdentityLabel = (identity: SdidLoginIdentity | undefined): string => {
  if (!identity) {
    return '未知身份';
  }
  const label = identity.label?.trim();
  if (label) {
    return label;
  }
  return identity.did?.trim() || '未知身份';
};

const createChallenge = (): string => {
  const buffer = new Uint32Array(1);
  const cryptoObj: Crypto | undefined =
    typeof window !== 'undefined' && window.crypto
      ? window.crypto
      : (globalThis as { crypto?: Crypto }).crypto;
  if (cryptoObj?.getRandomValues) {
    cryptoObj.getRandomValues(buffer);
    return `demo:${Date.now().toString(16)}:${buffer[0].toString(16)}`;
  }
  const fallback = Math.floor(Math.random() * 0xffffffff).toString(16);
  return `demo:${Date.now().toString(16)}:${fallback}`;
};

const Login = () => {
  const navigate = useNavigate();
  const { setToken } = useSession();

  const [loading, setLoading] = useState(false);
  const [status, setStatus] = useState<StatusState>({
    message: '正在等待 SDID 插件…',
    tone: 'info'
  });
  const [verification, setVerification] = useState<VerificationState>(null);
  const [identityResponse, setIdentityResponse] = useState<SdidLoginResponse | null>(null);
  const [sdidReady, setSdidReady] = useState(() => getSdidBridge() !== null);

  useEffect(() => {
    if (sdidReady) {
      setStatus({ message: 'SDID 插件已就绪，点击按钮开始登录。', tone: 'info' });
      return;
    }
    const handleInitialized = (event: Event) => {
      const detail = (event as CustomEvent<SdidInitDetail>).detail;
      if (bridgeFromDetail(detail) || getSdidBridge()) {
        setSdidReady(true);
        setStatus({ message: 'SDID 插件已就绪，点击按钮开始登录。', tone: 'info' });
      }
    };
    window.addEventListener('sdid#initialized', handleInitialized as EventListener);
    window.addEventListener('sdid#ready', handleInitialized as EventListener);
    document.addEventListener('sdid#initialized', handleInitialized as EventListener);
    document.addEventListener('sdid#ready', handleInitialized as EventListener);
    return () => {
      window.removeEventListener('sdid#initialized', handleInitialized as EventListener);
      window.removeEventListener('sdid#ready', handleInitialized as EventListener);
      document.removeEventListener('sdid#initialized', handleInitialized as EventListener);
      document.removeEventListener('sdid#ready', handleInitialized as EventListener);
    };
  }, [sdidReady]);

  const handleSdidLogin = async () => {
    if (loading) {
      return;
    }
    setLoading(true);
    setVerification(null);
    setIdentityResponse(null);
    setStatus({ message: '正在请求授权…', tone: 'info' });

    try {
      const [bridge] = await Promise.all([
        waitForSdidBridge()
      ]);

      const challenge = createChallenge();

      setStatus({ message: '请在 SDID 插件中确认登录请求…', tone: 'info' });

      const response = await bridge.requestLogin({
        message: 'RoundOne Ledger 请求访问',
        challenge
      });

      const loginResponse: SdidLoginResponse = {
        ...response,
        challenge: response.challenge || challenge
      };

      setStatus({ message: '正在验证签名…', tone: 'info' });
      const verificationResult = await verifySdidResponse(loginResponse);
      setVerification(verificationResult);
      if (!verificationResult?.success) {
        setStatus({
          message: verificationResult?.message || '签名验证失败，请重试。',
          tone: 'error'
        });
        return;
      }

      setIdentityResponse(loginResponse);

      const { data } = await api.post('/auth/login', {
        nonce: challenge,
        response: loginResponse
      });

      const label = formatIdentityLabel(loginResponse.identity);
      setStatus({ message: `已连接身份：${label}`, tone: 'success' });
      setToken(data.token);
      navigate('/dashboard');
    } catch (error) {
      console.error('SDID login failed', error);
      const message =
        (error as any)?.response?.data?.error ||
        (error instanceof Error ? error.message : 'SDID 登录失败，请确认插件已启用并允许此站点。');
      setStatus({ message, tone: 'error' });
      setVerification({ message: message || 'SDID 登录失败。', success: false });
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="flex min-h-screen items-center justify-center px-6 py-16">
      <div className="w-full max-w-3xl space-y-6 rounded-[32px] border border-white bg-white/90 p-10 shadow-glow">
        <div className="flex flex-col gap-6 md:flex-row md:items-center md:justify-between">
          <div className="flex items-center gap-3">
            <div className="rounded-2xl bg-neon-500/15 p-3 text-neon-500">
              <LockClosedIcon className="h-7 w-7" />
            </div>
            <div>
              <h1 className="text-2xl font-semibold text-night-50">登录控制台</h1>
              <p className="text-sm text-night-200">
                使用 SDID 钱包进行一键验证，无需注册，身份审批与管理全部在 SDID 扩展中完成。
              </p>
            </div>
          </div>
          <button
            type="button"
            className="button-primary w-full md:w-auto"
            onClick={handleSdidLogin}
            disabled={loading}
          >
            {loading ? '签名验证中…' : '连接 SDID 登录'}
          </button>
        </div>

        <div className="grid gap-6 md:grid-cols-2">
          <section className="space-y-3 rounded-2xl bg-night-900/20 p-5">
            <h2 className="text-sm font-medium text-night-100">流程说明</h2>
            <ol className="list-decimal space-y-2 pl-5 text-sm text-night-300">
              <li>页面生成随机 challenge，并请求 SDID 插件发起登录签名。</li>
              <li>插件弹出确认窗口，用户选择身份并授权签名。</li>
              <li>扩展返回 DID、公钥以及签名，页面使用 WebCrypto 验证签名。</li>
              <li>验证通过后将响应提交到服务端换取会话。</li>
            </ol>
          </section>

          <section className="space-y-3 rounded-2xl bg-night-900/20 p-5">
            <h2 className="text-sm font-medium text-night-100">当前状态</h2>
            <p
              className={`rounded-xl px-4 py-3 text-sm ${
                status.tone === 'success'
                  ? 'bg-emerald-100 text-emerald-700'
                  : status.tone === 'error'
                  ? 'bg-red-100 text-red-600'
                  : 'bg-night-900/40 text-night-200'
              }`}
            >
              {status.message}
            </p>
            {verification && (
              <p
                className={`rounded-xl px-4 py-3 text-sm ${
                  verification.success
                    ? 'bg-emerald-100 text-emerald-700'
                    : 'bg-red-100 text-red-600'
                }`}
              >
                {verification.message}
              </p>
            )}
          </section>
        </div>

        <section className="rounded-2xl bg-night-900/20 p-5">
          <h2 className="text-sm font-medium text-night-100">SDID 返回数据</h2>
          <pre className="mt-3 max-h-72 overflow-auto rounded-xl bg-night-950/60 p-4 text-xs text-night-200">
            {identityResponse ? JSON.stringify(identityResponse, null, 2) : '等待登录…'}
          </pre>
        </section>
      </div>
    </div>
  );
};

export default Login;
