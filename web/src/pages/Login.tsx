import { useState } from 'react';
import { Link, useNavigate } from 'react-router-dom';
import api from '../api/client';
import { fingerprintHash } from '../utils/fingerprint';
import { useSession } from '../hooks/useSession';
import { LockClosedIcon } from '@heroicons/react/24/outline';

const encoder = new TextEncoder();

const Login = () => {
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [sdidError, setSdidError] = useState<string | null>(null);
  const [sdid, setSdid] = useState('');
  const [showSdidModal, setShowSdidModal] = useState(false);
  const navigate = useNavigate();
  const { setToken } = useSession();

  const signChallenge = async (nonce: string, fingerprint: string) => {
    const anyWindow = window as any;
    const payload = nonce + fingerprint;
    if (anyWindow.SDID && typeof anyWindow.SDID.sign === 'function') {
      const result = await anyWindow.SDID.sign(payload);
      if (typeof result === 'string') {
        return result;
      }
      if (result && typeof result.signature === 'string') {
        return result.signature;
      }
      throw new Error('未能获取 SDID 签名。');
    }

    if (!anyWindow.ledgerPrivateKey) {
      throw new Error('缺少 SDID 私钥');
    }

    const signature = await window.crypto.subtle.sign(
      { name: 'NODE-ED25519' } as AlgorithmIdentifier,
      anyWindow.ledgerPrivateKey,
      encoder.encode(payload)
    );
    return btoa(String.fromCharCode(...new Uint8Array(signature)));
  };

  const confirmSdidLogin = async () => {
    if (loading) {
      return;
    }
    setSdidError(null);
    setLoading(true);

    try {
      const { data: nonceResp } = await api.post('/auth/request-nonce', {
        sdid
      });
      const fp = await fingerprintHash();
      const signatureB64 = await signChallenge(nonceResp.nonce, fp);
      const { data } = await api.post('/auth/login', {
        sdid,
        nonce: nonceResp.nonce,
        fingerprint: fp,
        signature: signatureB64
      });

      setToken(data.token);
      setShowSdidModal(false);
      navigate('/dashboard');
    } catch (err) {
      console.error(err);
      setSdidError('SDID 登录失败，请确认签名插件已开启且当前 IP 在允许列表中。');
    } finally {
      setLoading(false);
    }
  };

  const openSdidModal = () => {
    if (!sdid) {
      setError('请输入设备 SDID。');
      return;
    }
    setError(null);
    setSdidError(null);
    setShowSdidModal(true);
  };

  const closeSdidModal = () => {
    if (!loading) {
      setShowSdidModal(false);
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
            <p className="text-sm text-night-200">仅支持 SDID 一键认证，需在允许的固定 IP 环境内使用。</p>
          </div>
        </div>

        <div className="space-y-5">
          <div>
            <label className="text-sm text-night-200">SDID</label>
            <input
              type="text"
              value={sdid}
              onChange={(e) => {
                setSdid(e.target.value);
                if (error) {
                  setError(null);
                }
              }}
              className="mt-2 w-full rounded-2xl border border-night-600 bg-night-900 px-4 py-3 text-sm focus:border-neon-500 focus:ring-neon-500/40"
              placeholder="例如：device-xxxxxxxx"
              required
            />
          </div>
          <div className="rounded-2xl bg-night-900/30 px-4 py-3 text-xs text-night-200">
            浏览器将弹出 SDID 签名确认窗口；设备需绑定当前 IP，并携带管理员批准的签名才能通过允许清单验证。
          </div>
          {error && <p className="rounded-2xl bg-red-100 px-4 py-3 text-sm text-red-500">{error}</p>}
          <button type="button" className="button-primary w-full" onClick={openSdidModal} disabled={loading}>
            {loading ? '处理中...' : '一键发起 SDID 登录'}
          </button>
        </div>

        <p className="text-xs text-night-300">
          第一次访问？
          <Link className="ml-2 text-neon-500" to="/enroll">
            立即注册设备
          </Link>
        </p>
      </div>

      {showSdidModal && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-night-900/70 backdrop-blur-sm">
          <div className="w-full max-w-md space-y-5 rounded-[28px] bg-white p-8 shadow-2xl">
            <div>
              <h2 className="text-lg font-semibold text-night-700">确认 SDID 签名</h2>
              <p className="mt-2 text-sm text-night-400">设备 {sdid || '（未填写）'} 将执行一次性签名完成登录，请确认管理员签名已通过。</p>
            </div>
            {sdidError && <p className="rounded-2xl bg-red-100 px-4 py-3 text-sm text-red-500">{sdidError}</p>}
            <div className="flex gap-3">
              <button type="button" className="button-primary flex-1" onClick={confirmSdidLogin} disabled={loading}>
                {loading ? '签名中...' : '确认签名并登录'}
              </button>
              <button
                type="button"
                className="flex-1 rounded-2xl border border-night-200 bg-white px-4 py-3 text-sm text-night-400 hover:border-night-300"
                onClick={closeSdidModal}
                disabled={loading}
              >
                取消
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
};

export default Login;
