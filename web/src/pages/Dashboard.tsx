import { SparklesIcon, ShieldCheckIcon, PencilSquareIcon } from '@heroicons/react/24/outline';

const metrics = [
  {
    title: '认证成功率',
    value: '99.7%',
    change: '+1.2% 本周',
    icon: ShieldCheckIcon
  },
  {
    title: '活跃台账',
    value: '12',
    change: '3 个正在协作编辑',
    icon: SparklesIcon
  },
  {
    title: '文档更新',
    value: '27',
    change: '今日新增 5 条批注',
    icon: PencilSquareIcon
  }
];

const Dashboard = () => (
  <div className="space-y-6">
    <section className="grid gap-4 sm:grid-cols-2 xl:grid-cols-3">
      {metrics.map((metric) => (
        <div
          key={metric.title}
          className="rounded-2xl border border-[var(--line)] bg-white p-6 shadow-[0_20px_36px_rgba(0,0,0,0.08)] transition-transform hover:-translate-y-1"
        >
          <div className="flex items-center justify-between">
            <div>
              <p className="text-xs uppercase tracking-[0.22em] text-[rgba(20,20,20,0.45)]">{metric.title}</p>
              <p className="mt-3 text-3xl font-semibold text-[var(--text)]">{metric.value}</p>
            </div>
            <metric.icon className="h-10 w-10 text-[var(--accent)]" />
          </div>
          <p className="mt-6 text-sm text-[rgba(20,20,20,0.55)]">{metric.change}</p>
        </div>
      ))}
    </section>

    <section className="rounded-2xl border border-[var(--line)] bg-white p-6 shadow-[0_22px_44px_rgba(0,0,0,0.08)]">
      <h2 className="section-title">安全脉冲</h2>
      <div className="mt-4 grid gap-4 md:grid-cols-3">
        <div className="rounded-2xl border border-[var(--line)] bg-[var(--bg-subtle)] p-5 shadow-sm">
          <p className="text-xs uppercase tracking-[0.18em] text-[rgba(20,20,20,0.45)]">链完整性</p>
          <p className="mt-2 text-2xl font-semibold text-[var(--text)]">100%</p>
          <p className="mt-4 text-sm text-[rgba(20,20,20,0.55)]">最新审计链校验通过，无篡改风险。</p>
        </div>
        <div className="rounded-2xl border border-[var(--line)] bg-[var(--bg-subtle)] p-5 shadow-sm">
          <p className="text-xs uppercase tracking-[0.18em] text-[rgba(20,20,20,0.45)]">协同编辑</p>
          <p className="mt-2 text-2xl font-semibold text-[var(--text)]">48 次</p>
          <p className="mt-4 text-sm text-[rgba(20,20,20,0.55)]">多用户实时维护共享台账与智能文档。</p>
        </div>
        <div className="rounded-2xl border border-[var(--line)] bg-[var(--bg-subtle)] p-5 shadow-sm">
          <p className="text-xs uppercase tracking-[0.18em] text-[rgba(20,20,20,0.45)]">IP 合规</p>
          <p className="mt-2 text-2xl font-semibold text-[var(--text)]">12 条</p>
          <p className="mt-4 text-sm text-[rgba(20,20,20,0.55)]">卡片化展示白名单健康状态。</p>
        </div>
      </div>
    </section>
  </div>
);

export default Dashboard;
