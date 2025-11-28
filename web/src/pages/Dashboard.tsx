import { useEffect, useMemo, useState } from 'react';
import api from '../api/client';
import {
  ChartPieIcon,
  ClockIcon,
  LinkIcon,
  TagIcon,
  ArrowPathIcon,
  SignalIcon
} from '@heroicons/react/24/outline';

type LedgerType = 'ips' | 'personnel' | 'systems';

interface LedgerOverview {
  type: LedgerType;
  count: number;
  last_updated?: string;
}

interface TagCount {
  tag: string;
  count: number;
}

interface RelationshipStats {
  total: number;
  by_ledger: Record<string, number>;
}

interface RecentEntry {
  id: string;
  type: LedgerType;
  name: string;
  updated_at: string;
  tags?: string[];
}

interface OverviewStats {
  ledgers: LedgerOverview[];
  tag_top: TagCount[];
  relationships: RelationshipStats;
  recent: RecentEntry[];
}

const ledgerLabels: Record<LedgerType, string> = {
  ips: 'IP 资产',
  personnel: '人员',
  systems: '系统'
};

const ledgerColors: Record<LedgerType, string> = {
  ips: 'bg-[rgba(99,102,241,0.12)] text-[rgba(79,70,229,0.9)]',
  personnel: 'bg-[rgba(16,185,129,0.12)] text-[rgba(5,150,105,0.9)]',
  systems: 'bg-[rgba(248,180,0,0.12)] text-[rgba(217,119,6,0.95)]'
};

const formatDate = (value?: string) => {
  if (!value) return '—';
  const d = new Date(value);
  if (Number.isNaN(d.getTime())) return '—';
  return d.toLocaleString();
};

const Dashboard = () => {
  const [stats, setStats] = useState<OverviewStats | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    (async () => {
      try {
        const { data } = await api.get<{ stats: OverviewStats }>('/api/v1/overview');
        setStats(data.stats);
      } catch (error) {
        console.error('overview fetch failed', error);
        setStats(null);
      } finally {
        setLoading(false);
      }
    })();
  }, []);

  const ledgerData = useMemo(() => {
    const defaults: LedgerOverview[] = [
      { type: 'ips', count: 0 },
      { type: 'personnel', count: 0 },
      { type: 'systems', count: 0 }
    ];
    if (!stats) return defaults;
    return defaults.map((item) => stats.ledgers.find((l) => l.type === item.type) || item);
  }, [stats]);

  const hasContent = stats && (stats.tag_top?.length || stats.recent?.length || ledgerData.some((l) => l.count > 0));
  const latestUpdate = stats?.recent?.[0]?.updated_at || stats?.ledgers?.[0]?.last_updated;

  return (
    <div className="space-y-6">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div>
          <p className="text-xs font-semibold uppercase tracking-[0.2em] text-[rgba(20,20,20,0.55)]">概览</p>
          <h2 className="section-title">台账统计面板</h2>
          <p className="text-sm text-[rgba(20,20,20,0.6)]">
            按照当前台账数据自动汇总核心指标，帮助你快速评估资源规模与联动关系。
          </p>
        </div>
        <div className="flex items-center gap-2 rounded-full border border-[var(--line)] bg-white px-3 py-1.5 text-xs text-[rgba(20,20,20,0.65)] shadow-sm">
          <ClockIcon className="h-4 w-4 text-[rgba(20,20,20,0.55)]" />
          最近更新：{formatDate(latestUpdate)}
        </div>
      </div>

      <section className="rounded-lg border border-[var(--line)] bg-white p-6 shadow-none">
        <div className="flex items-center gap-2 text-sm text-[rgba(20,20,20,0.55)]">
          <ChartPieIcon className="h-5 w-5 text-[var(--accent)]" />
          数据总览
        </div>
        <div className="mt-4 grid gap-4 md:grid-cols-3">
          {ledgerData.map((ledger) => (
            <div
              key={ledger.type}
              className="rounded-md border border-[var(--line)] bg-[var(--bg-subtle)] p-4 shadow-none"
            >
              <div className="flex items-center justify-between">
                <span
                  className={`rounded-full px-3 py-1 text-xs font-semibold tracking-wide ${ledgerColors[ledger.type]}`}
                >
                  {ledgerLabels[ledger.type]}
                </span>
                <span className="text-[11px] uppercase tracking-[0.18em] text-[rgba(20,20,20,0.55)]">总量</span>
              </div>
              <p className="mt-4 text-3xl font-semibold text-[var(--text)]">{ledger.count}</p>
              <p className="mt-1 text-xs text-[rgba(20,20,20,0.55)]">
                最近更新时间 {formatDate(ledger.last_updated)}
              </p>
            </div>
          ))}
        </div>
      </section>

      <div className="grid gap-4 lg:grid-cols-3">
        <section className="lg:col-span-2 rounded-lg border border-[var(--line)] bg-white p-6 shadow-none">
          <div className="flex items-center gap-2 text-sm text-[rgba(20,20,20,0.55)]">
            <LinkIcon className="h-5 w-5 text-[var(--accent)]" />
            关联关系
          </div>
          <div className="mt-4 grid gap-3 sm:grid-cols-3">
            <div className="rounded-md border border-[var(--line)] bg-[var(--bg-subtle)] p-4">
              <p className="text-xs uppercase tracking-[0.2em] text-[rgba(20,20,20,0.55)]">总引用数</p>
              <p className="mt-2 text-2xl font-semibold text-[var(--text)]">
                {stats?.relationships?.total ?? 0}
              </p>
              <p className="mt-1 text-xs text-[rgba(20,20,20,0.55)]">跨台账的链接关系数量</p>
            </div>
            {(['ips', 'personnel', 'systems'] as LedgerType[]).map((type) => (
              <div key={type} className="rounded-md border border-[var(--line)] bg-white p-4">
                <p className="text-xs uppercase tracking-[0.18em] text-[rgba(20,20,20,0.55)]">
                  指向 {ledgerLabels[type]}
                </p>
                <p className="mt-2 text-xl font-semibold text-[var(--text)]">
                  {stats?.relationships?.by_ledger?.[type] ?? 0}
                </p>
                <p className="mt-1 text-xs text-[rgba(20,20,20,0.55)]">链接出现次数</p>
              </div>
            ))}
          </div>
        </section>

        <section className="rounded-lg border border-[var(--line)] bg-white p-6 shadow-none">
          <div className="flex items-center gap-2 text-sm text-[rgba(20,20,20,0.55)]">
            <TagIcon className="h-5 w-5 text-[var(--accent)]" />
            高频标签
          </div>
          <div className="mt-4 space-y-3">
            {(stats?.tag_top ?? []).length === 0 && (
              <p className="text-sm text-[rgba(20,20,20,0.55)]">暂无标签分布，请先创建台账条目。</p>
            )}
            {(stats?.tag_top ?? []).map((tag) => (
              <div key={tag.tag} className="rounded-md border border-[var(--line)] bg-[var(--bg-subtle)] p-3">
                <div className="flex items-center justify-between text-sm font-semibold text-[var(--text)]">
                  <span className="truncate">{tag.tag}</span>
                  <span className="text-[rgba(20,20,20,0.65)]">{tag.count}</span>
                </div>
                <div className="mt-2 h-2 rounded-full bg-white">
                  <div
                    className="h-2 rounded-full bg-[var(--accent)] transition-all"
                    style={{ width: `${Math.min(100, 8 * tag.count)}%` }}
                  />
                </div>
              </div>
            ))}
          </div>
        </section>
      </div>

      <section className="rounded-lg border border-[var(--line)] bg-white p-6 shadow-none">
        <div className="flex items-center gap-2 text-sm text-[rgba(20,20,20,0.55)]">
          <SignalIcon className="h-5 w-5 text-[var(--accent)]" />
          最近变更
        </div>
        {(!hasContent && !loading) && (
          <p className="mt-4 text-sm text-[rgba(20,20,20,0.55)]">暂无可展示的数据，先从侧边栏新增台账吧。</p>
        )}
        {loading && (
          <div className="mt-4 flex items-center gap-2 text-sm text-[rgba(20,20,20,0.65)]">
            <ArrowPathIcon className="h-4 w-4 animate-spin" />
            正在获取数据...
          </div>
        )}
        <div className="mt-4 divide-y divide-[rgba(20,20,20,0.08)]">
          {(stats?.recent ?? []).map((item) => (
            <div key={item.id} className="flex flex-col gap-2 py-3 sm:flex-row sm:items-center sm:justify-between">
              <div className="flex flex-col gap-1">
                <div className="flex items-center gap-2">
                  <span
                    className={`rounded-full px-3 py-1 text-xs font-semibold tracking-wide ${ledgerColors[item.type]}`}
                  >
                    {ledgerLabels[item.type]}
                  </span>
                  <p className="text-sm font-semibold text-[var(--text)]">{item.name}</p>
                </div>
                <div className="flex flex-wrap items-center gap-2 text-xs text-[rgba(20,20,20,0.6)]">
                  <ClockIcon className="h-4 w-4 text-[rgba(20,20,20,0.45)]" />
                  {formatDate(item.updated_at)}
                  {(item.tags || []).map((tag) => (
                    <span
                      key={tag}
                      className="rounded-full bg-[var(--bg-subtle)] px-2 py-1 text-[11px] font-medium text-[rgba(20,20,20,0.7)]"
                    >
                      {tag}
                    </span>
                  ))}
                </div>
              </div>
            </div>
          ))}
        </div>
      </section>
    </div>
  );
};

export default Dashboard;
