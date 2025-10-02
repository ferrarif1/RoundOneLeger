import { useEffect, useMemo, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import type { AxiosError } from 'axios';
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

type ApprovalResponse = {
  id: string;
  applicantDid?: string;
  applicantLabel?: string;
  applicantRoles?: string[];
  status: string;
  createdAt?: string;
  approvedAt?: string;
  approverDid?: string;
  approverLabel?: string;
  approverRoles?: string[];
  signingChallenge?: string;
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
  mode?: 'webcrypto' | 'skipped';
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

const decodeBase64 = (value: string): Uint8Array => {
  const normalized = normalizeBase64(value);
  if (!normalized) {
    return new Uint8Array();
  }
  if (typeof atob === 'function') {
    const binary = atob(normalized);
    const bytes = new Uint8Array(binary.length);
    for (let i = 0; i < binary.length; i += 1) {
      bytes[i] = binary.charCodeAt(i);
    }
    return bytes;
  }
  const bufferCtor = (globalThis as Record<string, unknown>).Buffer as
    | undefined
    | {
        from?: (input: string, encoding: string) => Uint8Array | number[];
      };
  if (bufferCtor?.from) {
    return new Uint8Array(bufferCtor.from(normalized, 'base64'));
  }
  throw new Error('Base64 decoding is not supported in this environment.');
};

const decodeSignature = (value: string): Uint8Array => decodeBase64(value);

const encodeText = (value: string): Uint8Array => {
  if (typeof TextEncoder !== 'undefined') {
    return new TextEncoder().encode(value);
  }
  const bufferCtor = (globalThis as Record<string, any>).Buffer as
    | undefined
    | { from?: (input: string, encoding: string) => Uint8Array | number[] };
  if (bufferCtor?.from) {
    return new Uint8Array(bufferCtor.from(value, 'utf8'));
  }
  throw new Error('TextEncoder not supported');
};

const importPublicKey = async (
  identity: SdidLoginIdentity,
  subtle: SubtleCrypto
): Promise<CryptoKey> => {
  if (!identity?.publicKeyJwk) {
    throw new Error('SDID 未返回公钥。');
  }
  const jwk = identity.publicKeyJwk as JsonWebKey;
  try {
    return await subtle.importKey(
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
  const identity = response.identity;
  if (!identity) {
    return { message: 'SDID 未返回身份信息。', success: false, mode: 'skipped' };
  }

  const signatureValue = response.proof?.signatureValue || response.signature;
  if (!signatureValue || !signatureValue.trim()) {
    return { message: 'SDID 返回数据缺少签名。', success: false, mode: 'skipped' };
  }

  let signedData: string;
  try {
    signedData = resolveSignedPayload(response);
  } catch (error) {
    return {
      message: error instanceof Error ? error.message : '无法解析签名内容。',
      success: false,
      mode: 'skipped'
    };
  }

  const subtle =
    typeof window !== 'undefined'
      ? window.crypto?.subtle || (window.crypto as Crypto & { webkitSubtle?: SubtleCrypto })?.webkitSubtle
      : undefined;

  if (!subtle) {
    return {
      message: '当前页面非安全来源，浏览器无法在本地验证签名，系统已默认信任该响应。',
      success: true,
      mode: 'skipped'
    };
  }

  try {
    const publicKey = await importPublicKey(identity, subtle);
    const signatureBytes = decodeSignature(signatureValue);
    const verified = await subtle.verify(
      { name: 'ECDSA', hash: { name: 'SHA-256' } },
      publicKey,
      signatureBytes,
      encodeText(signedData)
    );
    if (verified) {
      return { message: '签名验证通过。', success: true, mode: 'webcrypto' };
    }
    return {
      message: '签名验证失败，将提交服务器复核。',
      success: false,
      mode: 'skipped'
    };
  } catch (error) {
    console.error('WebCrypto 验证失败', error);
    return {
      message: '无法完成本地签名验证，将提交服务器复核。',
      success: false,
      mode: 'skipped'
    };
  }
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

const formatRoles = (roles: string[] | undefined): string => {
  if (!roles || !roles.length) {
    return '—';
  }
  return roles.filter((role) => role && role.trim()).join(', ') || '—';
};

const shortenValue = (value: string, head = 12, tail = 8): string => {
  if (!value) {
    return '—';
  }
  const trimmed = value.trim();
  if (trimmed.length <= head + tail + 1) {
    return trimmed;
  }
  return `${trimmed.slice(0, head)}…${trimmed.slice(-tail)}`;
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
  const [loggedInLabel, setLoggedInLabel] = useState<string | null>(null);
  const [loginComplete, setLoginComplete] = useState(false);
  const [sdidReady, setSdidReady] = useState(() => getSdidBridge() !== null);
  const [approvalStatus, setApprovalStatus] = useState<string | null>(null);
  const [approvalRecord, setApprovalRecord] = useState<ApprovalResponse | null>(null);
  const [submittingApproval, setSubmittingApproval] = useState(false);
  const [lastNonce, setLastNonce] = useState<string | null>(null);

  const identitySummary = useMemo(() => {
    if (!identityResponse?.identity) {
      return [] as Array<{ label: string; value: string }>;
    }
    const signature =
      identityResponse.proof?.signatureValue ||
      identityResponse.signature ||
      '';
    const canonical = identityResponse.authentication?.canonicalRequest || '';
    return [
      { label: 'DID', value: identityResponse.identity.did || '—' },
      { label: '显示名称', value: identityResponse.identity.label || '—' },
      { label: '角色', value: formatRoles(identityResponse.identity.roles) },
      { label: 'Challenge', value: identityResponse.challenge || '—' },
      { label: 'Canonical', value: canonical || '—' },
      { label: '签名', value: signature || '—' }
    ];
  }, [identityResponse]);

  const connectButtonLabel = loggedInLabel
    ? `${loggedInLabel} 已登陆`
    : loading
    ? '签名验证中…'
    : sdidReady
    ? '连接 SDID 登录'
    : '等待 SDID 插件…';

  const verificationBadge = verification?.mode === 'webcrypto'
    ? 'WebCrypto'
    : verification?.mode === 'skipped'
    ? verification.success
      ? '已信任'
      : '等待服务器'
    : null;

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
    setLoginComplete(false);
    setLoggedInLabel(null);
    setApprovalStatus(null);
    setApprovalRecord(null);
    setSubmittingApproval(false);
    setLastNonce(null);
    setStatus({ message: '正在请求授权…', tone: 'info' });

    try {
      const [bridge] = await Promise.all([
        waitForSdidBridge()
      ]);

      let challenge = createChallenge();
      let requestMessage = 'RoundOneLeger 请求访问';

      try {
        const { data } = await api.post('/auth/request-nonce');
        if (data?.nonce && typeof data.nonce === 'string') {
          challenge = data.nonce.trim() || challenge;
        }
        if (data?.message && typeof data.message === 'string') {
          requestMessage = data.message.trim() || requestMessage;
        }
      } catch (nonceError) {
        console.warn('Unable to fetch server nonce, falling back to local challenge.', nonceError);
      }

      setStatus({ message: '请在 SDID 插件中确认登录请求…', tone: 'info' });

      const response = await bridge.requestLogin({
        message: requestMessage,
        challenge
      });

      const loginResponse: SdidLoginResponse = {
        ...response,
        challenge: response.challenge || challenge
      };

      setIdentityResponse(loginResponse);
      setLastNonce(challenge);

      setStatus({ message: '正在验证签名…', tone: 'info' });
      const verificationResult = await verifySdidResponse(loginResponse);
      setVerification(verificationResult);

      setStatus({ message: '正在向服务器提交登录数据…', tone: 'info' });

      const { data } = await api.post('/auth/login', {
        nonce: challenge,
        response: loginResponse
      });

      const label = formatIdentityLabel(loginResponse.identity);
      setStatus({ message: `已连接身份：${label}`, tone: 'success' });
      setLoggedInLabel(label);
      setLoginComplete(true);
      setToken(data.token);
    } catch (error) {
      console.error('SDID login failed', error);
      const axiosError = error as AxiosError<{ error?: string; status?: string; approval?: ApprovalResponse }>;
      const serverCode = axiosError?.response?.data?.error;
      let message: string;
      if (axiosError.response?.status === 403 && serverCode === 'identity_not_approved') {
        message = '当前身份尚未通过管理员认证，请提交审批请求后等待管理员签名。';
        setApprovalStatus(axiosError.response.data?.status ?? 'pending');
        setApprovalRecord(axiosError.response.data?.approval ?? null);
      } else {
        message =
          serverCode ||
          (error instanceof Error
            ? error.message
            : 'SDID 登录失败，请确认插件已启用并允许此站点。');
      }

      setStatus({ message, tone: 'error' });
      setVerification((previous) => ({
        message:
          previous?.success && message
            ? `${previous.message.replace(/。$/, '')}，但服务器会话创建失败：${message}`
            : message || 'SDID 登录失败。',
        success: false,
        mode: 'skipped'
      }));
    } finally {
      setLoading(false);
    }
  };

  const handleSubmitApproval = async () => {
    if (!identityResponse) {
      setStatus({ message: '请先完成一次 SDID 签名再提交审批请求。', tone: 'error' });
      return;
    }
    if (!lastNonce) {
      setStatus({ message: '缺少登录挑战，无法提交审批。', tone: 'error' });
      return;
    }
    setSubmittingApproval(true);
    try {
      const { data } = await api.post('/auth/approvals', {
        nonce: lastNonce,
        response: identityResponse
      });
      if (typeof data?.status === 'string') {
        setApprovalStatus(data.status);
      }
      if (data?.approval) {
        setApprovalRecord(data.approval as ApprovalResponse);
      }
      setStatus({ message: '审批请求已提交，请等待管理员签名。', tone: 'info' });
    } catch (error) {
      console.error('Unable to submit approval request', error);
      const axiosError = error as AxiosError<{ error?: string; status?: string; approval?: ApprovalResponse }>;
      const message =
        axiosError.response?.data?.error ||
        axiosError.message ||
        '审批请求提交失败，请稍后重试。';
      setStatus({ message, tone: 'error' });
      if (axiosError.response?.data?.status) {
        setApprovalStatus(axiosError.response.data.status);
      }
      if (axiosError.response?.data?.approval) {
        setApprovalRecord(axiosError.response.data.approval as ApprovalResponse);
      }
    } finally {
      setSubmittingApproval(false);
    }
  };

  const formatTimestamp = (value?: string) => {
    if (!value) return '—';
    try {
      const date = new Date(value);
      if (Number.isNaN(date.getTime())) {
        return '—';
      }
      return date.toLocaleString();
    } catch {
      return value;
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
          <div className="flex w-full flex-col gap-3 md:w-auto md:flex-row md:items-center">
            <button
              type="button"
              className="button-primary w-full md:w-auto"
              onClick={handleSdidLogin}
              disabled={loading || Boolean(loggedInLabel)}
            >
              {connectButtonLabel}
            </button>
            {loginComplete && (
              <button
                type="button"
                className="w-full rounded-2xl border border-night-700 px-6 py-3 text-sm font-medium text-night-100 transition hover:bg-night-900/40 md:w-auto"
                onClick={() => navigate('/dashboard')}
              >
                进入控制台
              </button>
            )}
          </div>
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
              <div
                className={`rounded-xl px-4 py-3 text-sm ${
                  verification.success
                    ? 'bg-emerald-100 text-emerald-700'
                    : 'bg-red-100 text-red-600'
                }`}
              >
                <div className="flex items-center justify-between gap-3">
                  <span>{verification.message}</span>
                  {verificationBadge && (
                    <span
                      className={`whitespace-nowrap rounded-full px-2 py-0.5 text-xs font-semibold ${
                        verification.success
                          ? 'bg-emerald-200/70 text-emerald-900'
                          : 'bg-red-200/70 text-red-700'
                      }`}
                    >
                      {verificationBadge}
                    </span>
                  )}
                </div>
              </div>
            )}
          </section>
        </div>

        {approvalStatus && (
          <section className="rounded-2xl bg-night-900/20 p-5">
            <h2 className="text-sm font-medium text-night-100">身份审批</h2>
            {approvalStatus === 'approved' ? (
              <p className="mt-2 rounded-xl bg-emerald-100/70 px-4 py-3 text-sm text-emerald-700">
                身份已通过管理员认证，可重新登录以获取会话。
              </p>
            ) : approvalStatus === 'pending' ? (
              <p className="mt-2 rounded-xl bg-amber-100/70 px-4 py-3 text-sm text-amber-700">
                审批请求已提交，等待管理员签名确认。
              </p>
            ) : (
              <p className="mt-2 rounded-xl bg-red-100/70 px-4 py-3 text-sm text-red-600">
                当前身份尚未提交审批请求，请点击下方按钮申请管理员认证。
              </p>
            )}
            {approvalStatus !== 'approved' && (
              <button
                type="button"
                onClick={handleSubmitApproval}
                disabled={submittingApproval || approvalStatus === 'pending'}
                className="mt-4 button-primary"
              >
                {approvalStatus === 'pending' ? '审批处理中' : submittingApproval ? '正在提交…' : '提交审批请求'}
              </button>
            )}
            {approvalRecord && (
              <dl className="mt-4 grid gap-3 text-sm text-night-200 sm:grid-cols-2">
                <div>
                  <dt className="text-xs uppercase tracking-wide text-night-500">提交时间</dt>
                  <dd className="mt-1 break-words text-night-100">{formatTimestamp(approvalRecord.createdAt)}</dd>
                </div>
                {approvalRecord.approvedAt && (
                  <div>
                    <dt className="text-xs uppercase tracking-wide text-night-500">批准时间</dt>
                    <dd className="mt-1 break-words text-night-100">{formatTimestamp(approvalRecord.approvedAt)}</dd>
                  </div>
                )}
                {approvalRecord.approverLabel && (
                  <div>
                    <dt className="text-xs uppercase tracking-wide text-night-500">审批人</dt>
                    <dd className="mt-1 break-words text-night-100">
                      {approvalRecord.approverLabel}
                      {approvalRecord.approverDid ? `（${approvalRecord.approverDid}）` : ''}
                    </dd>
                  </div>
                )}
                {approvalRecord.applicantRoles && approvalRecord.applicantRoles.length > 0 && (
                  <div>
                    <dt className="text-xs uppercase tracking-wide text-night-500">申请角色</dt>
                    <dd className="mt-1 break-words text-night-100">{formatRoles(approvalRecord.applicantRoles)}</dd>
                  </div>
                )}
              </dl>
            )}
          </section>
        )}

        <section className="rounded-2xl bg-night-900/20 p-5">
          <h2 className="text-sm font-medium text-night-100">SDID 返回数据</h2>
          {identityResponse ? (
            <>
              <dl className="mt-3 grid gap-3 sm:grid-cols-2">
                {identitySummary.map((item) => (
                  <div key={item.label} className="rounded-xl bg-night-950/40 p-3">
                    <dt className="text-xs font-semibold uppercase tracking-wide text-night-400">
                      {item.label}
                    </dt>
                    <dd
                      className="mt-1 break-all text-sm text-night-100"
                      title={item.value}
                    >
                      {item.label === '签名' || item.label === 'Canonical'
                        ? shortenValue(item.value)
                        : item.value}
                    </dd>
                  </div>
                ))}
              </dl>
              <pre className="mt-4 max-h-72 overflow-auto rounded-xl bg-night-950/60 p-4 text-xs text-night-200">
                {JSON.stringify(identityResponse, null, 2)}
              </pre>
            </>
          ) : (
            <pre className="mt-3 max-h-72 overflow-auto rounded-xl bg-night-950/60 p-4 text-xs text-night-200">
              等待登录…
            </pre>
          )}
        </section>
      </div>
    </div>
  );
};

export default Login;
