import { useEffect, useState } from 'react';
import api from '../api/client';
import { ArrowTrendingUpIcon } from '@heroicons/react/24/outline';

interface AuditLog {
  id: string;
  actor: string;
  action: string;
  prev_hash: string;
  record_hash: string;
  created_at: string;
}

const AuditLogs = () => {
  const [logs, setLogs] = useState<AuditLog[]>([]);

  useEffect(() => {
    (async () => {
      const { data } = await api.get('/audit-logs');
      setLogs(data);
    })();
  }, []);

  return (
    <div className="space-y-6">
      <div className="flex flex-col gap-3 md:flex-row md:items-center md:justify-between">
        <h2 className="section-title">审计链</h2>
        <button className="button-primary flex items-center gap-2">
          <ArrowTrendingUpIcon className="h-4 w-4" /> 导出签名日志
        </button>
      </div>

      <div className="overflow-hidden rounded-3xl border border-ink-200 bg-white shadow-glow">
        <table className="min-w-full divide-y divide-ink-200">
          <thead className="bg-ink-50 text-xs uppercase tracking-wider text-night-300">
            <tr>
              <th className="px-6 py-3 text-left">时间</th>
              <th className="px-6 py-3 text-left">操作者</th>
              <th className="px-6 py-3 text-left">动作</th>
              <th className="px-6 py-3 text-left">记录哈希</th>
              <th className="px-6 py-3 text-left">前序哈希</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-ink-200 text-sm">
            {logs.map((log) => (
              <tr key={log.id} className="transition-colors hover:bg-ink-50">
                <td className="px-6 py-4 text-night-300">
                  {new Date(log.created_at).toLocaleString()}
                </td>
                <td className="px-6 py-4 text-night-200">{log.actor}</td>
                <td className="px-6 py-4 text-night-200">{log.action}</td>
                <td className="px-6 py-4 text-[11px] text-neon-500">{log.record_hash}</td>
                <td className="px-6 py-4 text-[11px] text-night-400">{log.prev_hash}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
};

export default AuditLogs;
