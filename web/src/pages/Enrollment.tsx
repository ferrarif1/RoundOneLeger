import { FormEvent, useState } from 'react';
import api from '../api/client';
import { fingerprintHash } from '../utils/fingerprint';
import { ArrowPathIcon, FingerPrintIcon } from '@heroicons/react/24/outline';

const encoder = new TextEncoder();

const Enrollment = () => {
  const [deviceName, setDeviceName] = useState('');
  const [step, setStep] = useState<'collect' | 'verify' | 'done'>('collect');
  const [nonce, setNonce] = useState('');
  const [fingerprint, setFingerprint] = useState('');
  const [deviceId, setDeviceId] = useState('');
  const [adminSignature, setAdminSignature] = useState('');
  const [error, setError] = useState<string | null>(null);

  const exportPublicKey = async (): Promise<string> => {
    const anyWindow = window as any;
    const key: CryptoKey | undefined = anyWindow.ledgerPublicKey || anyWindow.ledgerKeyPair?.publicKey;
    if (!key) {
      throw new Error('缺少公钥');
    }
    const raw = await window.crypto.subtle.exportKey('raw', key);
    const bytes = new Uint8Array(raw);
    const base64 = btoa(String.fromCharCode(...bytes));
    return base64.replace(/=+$/g, '');
  };

  const handleCollect = async () => {
    try {
      setError(null);
      const fp = await fingerprintHash();
      setFingerprint(fp);
      const publicKey = await exportPublicKey();
      const { data } = await api.post('/auth/enroll-request', {
        device_name: deviceName,
        public_key: publicKey
      });
      setNonce(data.nonce);
      setDeviceId(data.device_id);
      setStep('verify');
    } catch (err) {
      console.error(err);
      setError('无法生成公钥或请求注册，请稍后重试。');
    }
  };

  const handleComplete = async (event: FormEvent) => {
    event.preventDefault();
    try {
      setError(null);
      const trimmedAdminSignature = adminSignature.trim();
      if (!trimmedAdminSignature) {
        setError('请粘贴管理员签名。');
        return;
      }
      if (!(window as any).ledgerPrivateKey) {
        throw new Error('缺少私钥');
      }
      const signature = await window.crypto.subtle.sign(
        { name: 'NODE-ED25519' } as AlgorithmIdentifier,
        (window as any).ledgerPrivateKey,
        encoder.encode(nonce + fingerprint)
      );
      const signatureB64 = btoa(String.fromCharCode(...new Uint8Array(signature)));
      await api.post('/auth/enroll-complete', {
        device_id: deviceId,
        nonce,
        fingerprint,
        signature: signatureB64,
        admin_signature: trimmedAdminSignature
      });
      setStep('done');
    } catch (err) {
      console.error(err);
      setError('注册流程失败，请确认浏览器支持私钥签名。');
    }
  };

  return (
    <div className="flex min-h-screen items-center justify-center px-6 py-16">
      <div className="w-full max-w-2xl space-y-6 rounded-[32px] border border-white bg-white/90 p-10 shadow-glow">
        <div className="flex flex-col gap-3 md:flex-row md:items-center md:justify-between">
          <div>
            <h1 className="text-2xl font-semibold text-night-50">设备注册</h1>
            <p className="text-sm text-night-200">采集浏览器指纹并绑定可信终端。</p>
            <p className="mt-2 text-xs text-night-300">当前访问 IP 将自动写入设备绑定，如需变更请在新的固定地址重新注册。</p>
          </div>
          <div className="flex items-center gap-2 rounded-full border border-ink-200 bg-white px-4 py-2 text-xs text-night-300 shadow-sm">
            <FingerPrintIcon className="h-4 w-4 text-neon-500" />
            通过 HMAC-SHA256 保护的浏览器指纹
          </div>
        </div>

        {error && <p className="rounded-2xl bg-red-100 px-4 py-3 text-sm text-red-500">{error}</p>}

        {step === 'collect' && (
          <div className="space-y-6">
            <div>
              <label className="text-sm text-night-200">设备名称</label>
              <input
                type="text"
                value={deviceName}
                onChange={(e) => setDeviceName(e.target.value)}
                className="mt-2 w-full rounded-2xl border border-ink-200 bg-white px-4 py-3 text-sm focus:border-neon-500 focus:ring-neon-500/40"
                placeholder="例如：办公终端"
                required
              />
            </div>
            <button onClick={handleCollect} className="button-primary w-full">
              采集指纹并请求注册
            </button>
          </div>
        )}

        {step === 'verify' && (
          <form onSubmit={handleComplete} className="space-y-6">
            <div className="rounded-2xl border border-ink-200 bg-white p-5 text-sm text-night-200 shadow-sm">
              <p className="font-medium text-night-100">签名挑战</p>
              <p className="mt-2 break-all text-xs text-night-300">{nonce}</p>
              <p className="mt-3 text-xs text-night-300">设备 SDID</p>
              <p className="break-all text-xs text-night-300">{deviceId}</p>
              <p className="mt-3 text-xs text-night-300">指纹哈希</p>
              <p className="break-all text-xs text-neon-500">{fingerprint}</p>
            </div>
            <div>
              <label className="text-sm text-night-200">管理员批准签名</label>
              <textarea
                value={adminSignature}
                onChange={(e) => setAdminSignature(e.target.value)}
                className="mt-2 h-28 w-full rounded-2xl border border-ink-200 bg-white px-4 py-3 text-sm focus:border-neon-500 focus:ring-neon-500/40"
                placeholder="请粘贴管理员使用私钥对设备 SDID 和公钥的签名（Base64）"
                required
              />
              <p className="mt-2 text-xs text-night-300">签名内容需包含设备标识与公钥，后续登录将复核该签名。</p>
            </div>
            <button type="submit" className="button-primary w-full">
              使用本地私钥签名并完成注册
            </button>
          </form>
        )}

        {step === 'done' && (
          <div className="space-y-4 text-center">
            <div className="mx-auto flex h-20 w-20 items-center justify-center rounded-full bg-neon-500/20">
              <ArrowPathIcon className="h-10 w-10 text-neon-500" />
            </div>
            <h2 className="text-xl font-semibold text-night-50">设备注册完成</h2>
            <p className="text-sm text-night-200">
              您现在可以返回并使用该设备登录后台控制台。
            </p>
          </div>
        )}
      </div>
    </div>
  );
};

export default Enrollment;
