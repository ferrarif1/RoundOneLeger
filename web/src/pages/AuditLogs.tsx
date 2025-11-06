import { useCallback, useEffect, useMemo, useState } from 'react';
import {
  ArrowDownTrayIcon,
  ArrowPathIcon,
  CheckCircleIcon,
  ExclamationTriangleIcon,
  ShieldCheckIcon
} from '@heroicons/react/24/outline';
import api from '../api/client';

interface ApiAuditLog {
  id: string;
  actor?: string;
  action?: string;
  details?: string;
  hash?: string;
  prev_hash?: string;
  created_at?: string;
}

interface AuditLog {
  id: string;
  actor: string;
  action: string;
  details: string;
  hash: string;
  prevHash: string;
  createdAt: string;
}

interface AuditListResponse {
  items?: ApiAuditLog[];
}

type VerificationStatus = 'success' | 'failure' | 'error';

interface VerificationResult {
  status: VerificationStatus;
  checkedAt: string;
}

const formatDateTime = (value: string) => {
  if (!value) {
    return '—';
  }
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return value;
  }
  return new Intl.DateTimeFormat('zh-CN', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit',
    hour12: false
  }).format(date);
};

const mapAuditLog = (entry: ApiAuditLog): AuditLog => ({
  id: entry.id,
  actor: entry.actor?.trim() ?? '',
  action: entry.action?.trim() ?? '',
  details: entry.details?.trim() ?? '',
  hash: entry.hash ?? '',
  prevHash: entry.prev_hash ?? '',
  createdAt: entry.created_at ?? ''
});

const actionButtonClasses = (disabled?: boolean) =>
  [
    'inline-flex items-center gap-2 rounded-full border px-4 py-2 text-sm font-medium transition-all duration-150',
    'shadow-sm bg-white border-[rgba(20,20,20,0.12)] text-[var(--text)]',
    disabled
      ? 'opacity-60 cursor-not-allowed'
      : 'hover:-translate-y-0.5 hover:border-black/60 hover:text-black'
  ].join(' ');

const AuditLogs = () => {
  const [logs, setLogs] = useState<AuditLog[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [verifying, setVerifying] = useState(false);
  const [verification, setVerification] = useState<VerificationResult | null>(null);

  const fetchLogs = useCallback(async () => {
    setLoading(true);
    try {
      const { data } = await api.get<AuditListResponse>('/api/v1/audit');
      const items = (data.items ?? []).map(mapAuditLog);
      items.sort((a, b) => {
        const left = new Date(a.createdAt).getTime();
        const right = new Date(b.createdAt).getTime();
        if (Number.isNaN(left) || Number.isNaN(right)) {
          return 0;
        }
        return right - left;
      });
      setLogs(items);
      setError(null);
    } catch (err) {
      console.error('Failed to load audit logs', err);
      setError('无法加载审计日志，请稍后重试。');
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void fetchLogs();
  }, [fetchLogs]);

  const handleVerify = useCallback(async () => {
    setVerifying(true);
    try {
      const { data } = await api.get<{ verified: boolean }>('/api/v1/audit/verify');
      setVerification({
        status: data.verified ? 'success' : 'failure',
        checkedAt: new Date().toISOString()
      });
    } catch (err) {
      console.error('Failed to verify audit chain', err);
      setVerification({
        status: 'error',
        checkedAt: new Date().toISOString()
      });
    } finally {
      setVerifying(false);
    }
  }, []);

  const handleExport = useCallback(() => {
    if (!logs.length) {
      return;
    }
    const timestamp = new Date().toISOString().replace(/[:.]/g, '-');
    const payload = {
      generatedAt: new Date().toISOString(),
      count: logs.length,
      items: logs.map((log) => ({
        id: log.id,
        actor: log.actor,
        action: log.action,
        details: log.details,
        hash: log.hash,
        prev_hash: log.prevHash,
        created_at: log.createdAt
      }))
    };
    const blob = new Blob([JSON.stringify(payload, null, 2)], {
      type: 'application/json;charset=utf-8'
    });
    const url = URL.createObjectURL(blob);
    const link = document.createElement('a');
    link.href = url;
    link.download = `audit-logs-${timestamp}.json`;
    document.body.appendChild(link);
    link.click();
    document.body.removeChild(link);
    URL.revokeObjectURL(url);
  }, [logs]);

  const verificationBanner = useMemo(() => {
    if (!verification) {
      return null;
    }
    const checkedAt = formatDateTime(verification.checkedAt);
    switch (verification.status) {
      case 'success':
        return (
          <div className="flex items-start gap-3 rounded-2xl border border-green-200 bg-green-50 p-4 text-sm text-green-700">
            <CheckCircleIcon className="h-5 w-5 flex-shrink-0" />
            <div>
              <p className="font-medium">链完整性校验通过</p>
              <p className="mt-1 text-xs text-green-600/80">最后校验时间：{checkedAt}</p>
            </div>
          </div>
        );
      case 'failure':
        return (
          <div className="flex items-start gap-3 rounded-2xl border border-red-200 bg-red-50 p-4 text-sm text-red-600">
            <ExclamationTriangleIcon className="h-5 w-5 flex-shrink-0" />
            <div>
              <p className="font-medium">审计链校验失败，存在篡改或缺失风险</p>
              <p className="mt-1 text-xs text-red-500/80">最后校验时间：{checkedAt}</p>
            </div>
          </div>
        );
      case 'error':
      default:
        return (
          <div className="flex items-start gap-3 rounded-2xl border border-amber-200 bg-amber-50 p-4 text-sm text-amber-700">
            <ExclamationTriangleIcon className="h-5 w-5 flex-shrink-0" />
            <div>
              <p className="font-medium">无法完成链校验，请稍后重试</p>
              <p className="mt-1 text-xs text-amber-600/80">尝试时间：{checkedAt}</p>
            </div>
          </div>
        );
    }
  }, [verification]);

  return (
    <div className="space-y-6">
      <div className="flex flex-col gap-3 md:flex-row md:items-center md:justify-between">
        <div>
          <h2 className="section-title">审计链</h2>
          <p className="mt-1 text-sm text-[rgba(20,20,20,0.55)]">
            每一次登录、台账和配置的改动都会生成不可逆的哈希链，便于事后追踪。
          </p>
        </div>
        <div className="flex flex-wrap gap-2">
          <button
            type="button"
            className={actionButtonClasses(loading)}
            onClick={() => void fetchLogs()}
            disabled={loading}
          >
            <ArrowPathIcon className="h-4 w-4" /> 刷新
          </button>
          <button
            type="button"
            className={actionButtonClasses(verifying || !logs.length)}
            onClick={() => void handleVerify()}
            disabled={verifying || !logs.length}
          >
            <ShieldCheckIcon className="h-4 w-4" />
            {verifying ? '校验中…' : '校验链完整性'}
          </button>
          <button
            type="button"
            className={actionButtonClasses(!logs.length)}
            onClick={handleExport}
            disabled={!logs.length}
          >
            <ArrowDownTrayIcon className="h-4 w-4" /> 导出签名日志
          </button>
        </div>
      </div>

      {error && (
        <div className="rounded-2xl border border-red-200 bg-red-50 p-4 text-sm text-red-600">{error}</div>
      )}

      {verificationBanner}

      <div className="overflow-hidden rounded-2xl border border-[var(--line)] bg-white shadow-[0_22px_44px_rgba(0,0,0,0.08)]">
        <table className="min-w-full divide-y divide-[rgba(20,20,20,0.12)]">
          <thead className="bg-[var(--bg-subtle)] text-xs uppercase tracking-wider text-[rgba(20,20,20,0.45)]">
            <tr>
              <th className="px-6 py-3 text-left">时间</th>
              <th className="px-6 py-3 text-left">操作者</th>
              <th className="px-6 py-3 text-left">动作</th>
              <th className="px-6 py-3 text-left">详情</th>
              <th className="px-6 py-3 text-left">记录哈希</th>
              <th className="px-6 py-3 text-left">前序哈希</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-[rgba(20,20,20,0.12)] text-sm">
            {loading && logs.length === 0 && (
              <tr>
                <td colSpan={6} className="px-6 py-12">
                  <div className="flex flex-col items-center gap-3">
                    <div className="h-3 w-32 animate-pulse rounded-full bg-[rgba(20,20,20,0.08)]" />
                    <div className="h-3 w-48 animate-pulse rounded-full bg-[rgba(20,20,20,0.08)]" />
                    <div className="h-3 w-24 animate-pulse rounded-full bg-[rgba(20,20,20,0.08)]" />
                  </div>
                </td>
              </tr>
            )}

            {!loading && logs.length === 0 && (
              <tr>
                <td colSpan={6} className="px-6 py-12 text-center text-[rgba(20,20,20,0.45)]">
                  暂无审计记录。
                </td>
              </tr>
            )}

            {logs.map((log) => (
              <tr key={log.id} className="transition-colors hover:bg-[var(--bg-subtle)]">
                <td className="px-6 py-4 text-[rgba(20,20,20,0.55)]">{formatDateTime(log.createdAt)}</td>
                <td className="px-6 py-4 text-[var(--text)]">
                  {log.actor || <span className="text-[rgba(20,20,20,0.35)]">系统</span>}
                </td>
                <td className="px-6 py-4 text-[var(--text)]">{log.action || '—'}</td>
                <td className="px-6 py-4 text-[rgba(20,20,20,0.55)]">
                  {log.details ? log.details : <span className="text-[rgba(20,20,20,0.35)]">—</span>}
                </td>
                <td className="px-6 py-4 font-mono text-xs text-[var(--accent)]" title={log.hash}>
                  <span className="break-all">{log.hash}</span>
                </td>
                <td className="px-6 py-4 font-mono text-xs text-[rgba(20,20,20,0.55)]" title={log.prevHash}>
                  <span className="break-all">{log.prevHash || '—'}</span>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
};

export default AuditLogs;
