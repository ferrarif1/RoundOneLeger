import { useEffect, useState } from 'react';
import type { AxiosError } from 'axios';

import api from '../api/client';

interface ApprovalItem {
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
}

type SdidRequestOptions = {
  message?: string;
  challenge?: string;
};

type SdidBridge = {
  requestLogin: (options: SdidRequestOptions) => Promise<Record<string, unknown>>;
};

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

const waitForSdidBridge = async (): Promise<SdidBridge> => {
  const immediate = getSdidBridge();
  if (immediate) {
    return immediate;
  }
  return new Promise((resolve, reject) => {
    const timeout = window.setTimeout(() => {
      window.removeEventListener('sdid#initialized', handleInitialized as EventListener);
      window.removeEventListener('sdid#ready', handleInitialized as EventListener);
      document.removeEventListener('sdid#initialized', handleInitialized as EventListener);
      document.removeEventListener('sdid#ready', handleInitialized as EventListener);
      reject(new Error('未检测到 SDID 浏览器插件，请安装或启用扩展。'));
    }, 8000);

    function handleInitialized(event: Event) {
      const detail = (event as CustomEvent<{ provider?: unknown }>).detail;
      const bridge = promoteBridge(detail?.provider) || getSdidBridge();
      if (!bridge) {
        return;
      }
      window.clearTimeout(timeout);
      window.removeEventListener('sdid#initialized', handleInitialized as EventListener);
      window.removeEventListener('sdid#ready', handleInitialized as EventListener);
      document.removeEventListener('sdid#initialized', handleInitialized as EventListener);
      document.removeEventListener('sdid#ready', handleInitialized as EventListener);
      resolve(bridge);
    }

    window.addEventListener('sdid#initialized', handleInitialized as EventListener);
    window.addEventListener('sdid#ready', handleInitialized as EventListener);
    document.addEventListener('sdid#initialized', handleInitialized as EventListener);
    document.addEventListener('sdid#ready', handleInitialized as EventListener);
  });
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

const Approvals = () => {
  const [items, setItems] = useState<ApprovalItem[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [submittingId, setSubmittingId] = useState<string | null>(null);

  const fetchApprovals = async () => {
    try {
      setLoading(true);
      const { data } = await api.get('/api/v1/approvals');
      setItems(Array.isArray(data?.items) ? (data.items as ApprovalItem[]) : []);
      setError(null);
    } catch (err) {
      console.error('Failed to load approvals', err);
      const axiosError = err as AxiosError<{ error?: string }>;
      setError(axiosError.response?.data?.error || axiosError.message || '无法加载审批请求。');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchApprovals();
  }, []);

  const handleApprove = async (item: ApprovalItem) => {
    if (!item.signingChallenge) {
      setError('该审批请求缺少签名挑战，无法完成操作。');
      return;
    }
    setError(null);
    setSubmittingId(item.id);
    try {
      const bridge = await waitForSdidBridge();
      const response = await bridge.requestLogin({
        message: `审批身份 ${item.applicantLabel || item.applicantDid || item.id}`,
        challenge: item.signingChallenge
      });
      await api.post(`/api/v1/approvals/${item.id}/approve`, { response });
      await fetchApprovals();
    } catch (err) {
      console.error('Approval signing failed', err);
      const axiosError = err as AxiosError<{ error?: string }>;
      setError(axiosError.response?.data?.error || axiosError.message || '审批未能完成，请重试。');
    } finally {
      setSubmittingId(null);
    }
  };

  const pending = items.filter((item) => item.status === 'pending');
  const completed = items.filter((item) => item.status !== 'pending');

  return (
    <div className="space-y-6">
      <div>
        <h2 className="section-title">身份审批</h2>
        <p className="mt-1 text-sm text-night-300">
          审核待处理的身份请求，使用 SDID 插件完成管理员签名，成功后用户即可通过登录。
        </p>
      </div>

      {error && (
        <div className="rounded-2xl border border-red-400/60 bg-red-100/40 px-4 py-3 text-sm text-red-600">{error}</div>
      )}

      {loading ? (
        <div className="glass-panel rounded-3xl p-6 text-center text-night-300">正在加载审批列表…</div>
      ) : (
        <div className="space-y-6">
          <section className="glass-panel rounded-3xl border border-ink-200/80 p-5">
            <h3 className="text-lg font-semibold text-night-100">待审批</h3>
            {pending.length === 0 ? (
              <p className="mt-3 text-sm text-night-400">暂无待审批请求。</p>
            ) : (
              <div className="mt-4 space-y-3">
                {pending.map((item) => (
                  <div key={item.id} className="rounded-2xl border border-ink-200/80 bg-white/70 p-4">
                    <div className="flex flex-wrap items-center justify-between gap-3">
                      <div>
                        <h4 className="text-base font-semibold text-night-100">
                          {item.applicantLabel || item.applicantDid || item.id}
                        </h4>
                        <p className="text-xs text-night-400">{item.applicantDid}</p>
                        <p className="mt-2 text-sm text-night-300">
                          角色：{item.applicantRoles && item.applicantRoles.length ? item.applicantRoles.join(', ') : '—'}
                        </p>
                        <p className="text-xs text-night-500">提交时间：{formatTimestamp(item.createdAt)}</p>
                      </div>
                      <button
                        type="button"
                        onClick={() => handleApprove(item)}
                        disabled={submittingId === item.id}
                        className="button-primary"
                      >
                        {submittingId === item.id ? '签名中…' : '签名审批'}
                      </button>
                    </div>
                  </div>
                ))}
              </div>
            )}
          </section>

          <section className="glass-panel rounded-3xl border border-ink-200/80 p-5">
            <h3 className="text-lg font-semibold text-night-100">已完成</h3>
            {completed.length === 0 ? (
              <p className="mt-3 text-sm text-night-400">尚无历史审批记录。</p>
            ) : (
              <div className="mt-4 overflow-x-auto">
                <table className="min-w-full divide-y divide-night-800/40 text-sm">
                  <thead className="text-left text-xs uppercase tracking-wide text-night-500">
                    <tr>
                      <th className="px-3 py-2">身份</th>
                      <th className="px-3 py-2">角色</th>
                      <th className="px-3 py-2">提交时间</th>
                      <th className="px-3 py-2">审批时间</th>
                      <th className="px-3 py-2">审批人</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-night-800/30 text-night-200">
                    {completed.map((item) => (
                      <tr key={item.id}>
                        <td className="px-3 py-2">{item.applicantLabel || item.applicantDid || item.id}</td>
                        <td className="px-3 py-2">
                          {item.applicantRoles && item.applicantRoles.length ? item.applicantRoles.join(', ') : '—'}
                        </td>
                        <td className="px-3 py-2">{formatTimestamp(item.createdAt)}</td>
                        <td className="px-3 py-2">{formatTimestamp(item.approvedAt)}</td>
                        <td className="px-3 py-2">
                          {item.approverLabel || item.approverDid || '—'}
                          {item.approverDid && item.approverLabel ? `（${item.approverDid}）` : ''}
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
          </section>
        </div>
      )}
    </div>
  );
};

export default Approvals;
