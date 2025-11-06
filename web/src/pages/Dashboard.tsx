import { useEffect, useMemo, useState } from 'react';
import {
  BoltIcon,
  IdentificationIcon,
  RectangleGroupIcon,
  ServerStackIcon,
  ShieldCheckIcon,
  UserGroupIcon
} from '@heroicons/react/24/outline';

import api from '../api/client';

interface OverviewResponse {
  overview: {
    generated_at: string;
    systems: number;
    personnel: number;
    ip_assets: number;
    workspaces: number;
    sheets: number;
    documents: number;
    folders: number;
    users: number;
    allowlist: number;
    logins_last_24_h?: number;
    logins_last_24h?: number;
  };
}

interface Overview {
  generatedAt: string;
  systems: number;
  personnel: number;
  ipAssets: number;
  workspaces: number;
  sheets: number;
  documents: number;
  folders: number;
  users: number;
  allowlist: number;
  loginsLast24h: number;
}

const formatNumber = (value: number) =>
  new Intl.NumberFormat('zh-CN', { maximumFractionDigits: 0 }).format(value);

const Dashboard = () => {
  const [overview, setOverview] = useState<Overview | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let mounted = true;
    const fetchOverview = async () => {
      try {
        const { data } = await api.get<OverviewResponse>('/api/v1/overview');
        if (!mounted) {
          return;
        }
        const payload = data.overview;
        setOverview({
          generatedAt: payload.generated_at,
          systems: payload.systems,
          personnel: payload.personnel,
          ipAssets: payload.ip_assets,
          workspaces: payload.workspaces,
          sheets: payload.sheets,
          documents: payload.documents,
          folders: payload.folders,
          users: payload.users,
          allowlist: payload.allowlist,
          loginsLast24h: payload.logins_last_24_h ?? payload.logins_last_24h ?? 0
        });
        setError(null);
      } catch (err) {
        console.error('Failed to load overview', err);
        if (mounted) {
          setError('无法加载概览数据，请稍后重试。');
        }
      } finally {
        if (mounted) {
          setLoading(false);
        }
      }
    };

    fetchOverview();
    return () => {
      mounted = false;
    };
  }, []);

  const headlineMetrics = useMemo(() => {
    if (!overview) {
      return [];
    }
    return [
      {
        title: '系统资产',
        value: formatNumber(overview.systems),
        detail: `${formatNumber(overview.workspaces)} 个协作空间持续跟踪`,
        icon: ServerStackIcon
      },
      {
        title: '人员档案',
        value: formatNumber(overview.personnel),
        detail: `${formatNumber(overview.sheets)} 张台账表格沉淀交接信息`,
        icon: UserGroupIcon
      },
      {
        title: 'IP 资产',
        value: formatNumber(overview.ipAssets),
        detail: `${formatNumber(overview.allowlist)} 条白名单策略护航访问`,
        icon: ShieldCheckIcon
      }
    ];
  }, [overview]);

  const secondaryMetrics = useMemo(() => {
    if (!overview) {
      return [];
    }
    return [
      {
        title: '协作空间',
        value: formatNumber(overview.workspaces),
        detail: `表格 ${formatNumber(overview.sheets)} · 文档 ${formatNumber(
          overview.documents
        )} · 文件夹 ${formatNumber(overview.folders)}`,
        icon: RectangleGroupIcon
      },
      {
        title: '最近 24 小时登录',
        value: formatNumber(overview.loginsLast24h),
        detail: '实时掌握后台访问频次，快速识别异常。',
        icon: BoltIcon
      },
      {
        title: '后台账号',
        value: formatNumber(overview.users),
        detail: '具备系统操作权限的管理员与协作者数量。',
        icon: IdentificationIcon
      }
    ];
  }, [overview]);

  const generatedAtText = useMemo(() => {
    if (!overview?.generatedAt) {
      return '';
    }
    const date = new Date(overview.generatedAt);
    if (Number.isNaN(date.getTime())) {
      return overview.generatedAt;
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
  }, [overview]);

  const renderPlaceholderCards = (count: number) => (
    <>
      {Array.from({ length: count }).map((_, index) => (
        <div
          key={index}
          className="h-36 animate-pulse rounded-2xl border border-[var(--line)] bg-white"
        />
      ))}
    </>
  );

  return (
    <div className="space-y-6">
      {error && (
        <div className="rounded-2xl border border-red-200 bg-red-50 p-4 text-sm text-red-600">
          {error}
        </div>
      )}

      <section className="grid gap-4 sm:grid-cols-2 xl:grid-cols-3">
        {loading && !overview && renderPlaceholderCards(3)}
        {!loading && overview &&
          headlineMetrics.map((metric) => (
            <div
              key={metric.title}
              className="rounded-2xl border border-[var(--line)] bg-white p-6 shadow-[0_20px_36px_rgba(0,0,0,0.08)] transition-transform hover:-translate-y-1"
            >
              <div className="flex items-center justify-between">
                <div>
                  <p className="text-xs uppercase tracking-[0.22em] text-[rgba(20,20,20,0.45)]">
                    {metric.title}
                  </p>
                  <p className="mt-3 text-3xl font-semibold text-[var(--text)]">{metric.value}</p>
                </div>
                <metric.icon className="h-10 w-10 text-[var(--accent)]" />
              </div>
              <p className="mt-6 text-sm text-[rgba(20,20,20,0.55)]">{metric.detail}</p>
            </div>
          ))}
      </section>

      <section className="rounded-2xl border border-[var(--line)] bg-white p-6 shadow-[0_22px_44px_rgba(0,0,0,0.08)]">
        <div className="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
          <h2 className="section-title">运营快照</h2>
          {generatedAtText && (
            <p className="text-xs uppercase tracking-[0.18em] text-[rgba(20,20,20,0.45)]">
              数据更新于 {generatedAtText}
            </p>
          )}
        </div>
        <div className="mt-4 grid gap-4 md:grid-cols-3">
          {loading && !overview && renderPlaceholderCards(3)}
          {!loading && overview &&
            secondaryMetrics.map((metric) => (
              <div
                key={metric.title}
                className="rounded-2xl border border-[var(--line)] bg-[var(--bg-subtle)] p-5 shadow-sm"
              >
                <div className="flex items-center gap-3">
                  <div className="rounded-xl border border-[rgba(20,20,20,0.12)] bg-white p-2">
                    <metric.icon className="h-6 w-6 text-[var(--accent)]" />
                  </div>
                  <div>
                    <p className="text-xs uppercase tracking-[0.18em] text-[rgba(20,20,20,0.45)]">
                      {metric.title}
                    </p>
                    <p className="mt-2 text-2xl font-semibold text-[var(--text)]">{metric.value}</p>
                  </div>
                </div>
                <p className="mt-4 text-sm text-[rgba(20,20,20,0.55)]">{metric.detail}</p>
              </div>
            ))}
        </div>
      </section>
    </div>
  );
};

export default Dashboard;
