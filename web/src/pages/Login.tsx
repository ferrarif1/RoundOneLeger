import { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import api from '../api/client';
import { useSession } from '../hooks/useSession';
import { LockClosedIcon } from '@heroicons/react/24/outline';

type RawWallet = {
  requestAccount?: () => Promise<any>;
  requestAccounts?: () => Promise<any>;
  getAccount?: () => Promise<any>;
  signMessage?: (message: string) => Promise<any>;
  sign?: (message: string) => Promise<any>;
  signData?: (message: string) => Promise<any>;
};

type SdidWallet = {
  requestAccount: () => Promise<any>;
  signMessage: (message: string) => Promise<any>;
};

const pickAccount = (value: any): { sdid: string; publicKey: string } | null => {
  if (!value) {
    return null;
  }

  if (Array.isArray(value)) {
    return pickAccount(value[0]);
  }

  if (typeof value === 'object') {
    const sdidValue = typeof value.sdid === 'string' ? value.sdid : typeof value.did === 'string' ? value.did : undefined;
    const pubKeyValue =
      typeof value.publicKey === 'string'
        ? value.publicKey
        : typeof value.pubKey === 'string'
        ? value.pubKey
        : typeof value.public_key === 'string'
        ? value.public_key
        : undefined;

    if (sdidValue && pubKeyValue) {
      return { sdid: sdidValue, publicKey: pubKeyValue };
    }

    if (value.account) {
      return pickAccount(value.account);
    }
  }

  return null;
};

const normalizeSignature = (value: any): { signature: string; sdid?: string; publicKey?: string } | null => {
  if (!value) {
    return null;
  }

  if (typeof value === 'string') {
    return { signature: value };
  }

  const signatureValue =
    typeof value.signature === 'string'
      ? value.signature
      : typeof value.sig === 'string'
      ? value.sig
      : typeof value.signedMessage === 'string'
      ? value.signedMessage
      : undefined;

  if (signatureValue) {
    return {
      signature: signatureValue,
      sdid: value.sdid,
      publicKey: value.publicKey
    };
  }

  if (value.result) {
    return normalizeSignature(value.result);
  }

  if (value.data) {
    return normalizeSignature(value.data);
  }

  return null;
};

const getWallet = (): SdidWallet => {
  const anyWindow = window as any;
  const raw: RawWallet | undefined =
    anyWindow.sdid?.wallet ||
    anyWindow.sdidWallet ||
    anyWindow.sdWallet?.wallet ||
    anyWindow.sdWallet ||
    anyWindow.SDID?.wallet ||
    anyWindow.SDID ||
    anyWindow.SDWallet;

  if (!raw) {
    throw new Error('未检测到 SDID 浏览器插件，请安装或启用扩展。');
  }

  const requestAccount =
    typeof raw.requestAccount === 'function'
      ? raw.requestAccount.bind(raw)
      : typeof raw.requestAccounts === 'function'
      ? raw.requestAccounts.bind(raw)
      : typeof raw.getAccount === 'function'
      ? raw.getAccount.bind(raw)
      : null;

  if (!requestAccount) {
    throw new Error('SDID 插件缺少 requestAccount 能力。');
  }

  const signMessage =
    typeof raw.signMessage === 'function'
      ? raw.signMessage.bind(raw)
      : typeof raw.sign === 'function'
      ? raw.sign.bind(raw)
      : typeof raw.signData === 'function'
      ? raw.signData.bind(raw)
      : null;

  if (!signMessage) {
    throw new Error('SDID 插件缺少签名能力。');
  }

  return { requestAccount, signMessage };
};

const Login = () => {
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const navigate = useNavigate();
  const { setToken } = useSession();

  const handleSdidLogin = async () => {
    if (loading) {
      return;
    }
    setLoading(true);

    try {
      const wallet = getWallet();
      const [{ data: nonceResp }, accountResult] = await Promise.all([
        api.post('/auth/request-nonce'),
        wallet.requestAccount()
      ]);
      const account = pickAccount(accountResult);
      if (!account) {
        throw new Error('无法获取 SDID 账号信息，请确认插件已登录。');
      }

      const message = nonceResp.message || nonceResp.nonce;
      const signaturePayload = await wallet.signMessage(message);
      const signatureResult = normalizeSignature(signaturePayload);
      const sdid = signatureResult?.sdid || account.sdid;
      const publicKey = signatureResult?.publicKey || account.publicKey;
      const signature = signatureResult?.signature;
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
