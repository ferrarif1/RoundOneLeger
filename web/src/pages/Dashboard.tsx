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
        <div key={metric.title} className="rounded-3xl border border-white bg-white/90 p-6 shadow-glow">
          <div className="flex items-center justify-between">
            <div>
              <p className="text-xs uppercase tracking-[0.22em] text-night-300">{metric.title}</p>
              <p className="mt-3 text-3xl font-semibold text-night-50">{metric.value}</p>
            </div>
            <metric.icon className="h-10 w-10 text-neon-500" />
          </div>
          <p className="mt-6 text-sm text-night-200">{metric.change}</p>
        </div>
      ))}
    </section>

    <section className="rounded-3xl border border-white bg-white/80 p-6 shadow-glow">
      <h2 className="section-title">安全脉冲</h2>
      <div className="mt-4 grid gap-4 md:grid-cols-3">
        <div className="rounded-2xl border border-ink-200 bg-white p-5 shadow-sm">
          <p className="text-xs uppercase tracking-[0.18em] text-night-300">链完整性</p>
          <p className="mt-2 text-2xl font-semibold text-night-50">100%</p>
          <p className="mt-4 text-sm text-night-200">最新审计链校验通过，无篡改风险。</p>
        </div>
        <div className="rounded-2xl border border-ink-200 bg-white p-5 shadow-sm">
          <p className="text-xs uppercase tracking-[0.18em] text-night-300">协同编辑</p>
          <p className="mt-2 text-2xl font-semibold text-night-50">48 次</p>
          <p className="mt-4 text-sm text-night-200">多用户实时维护共享台账与智能文档。</p>
        </div>
        <div className="rounded-2xl border border-ink-200 bg-white p-5 shadow-sm">
          <p className="text-xs uppercase tracking-[0.18em] text-night-300">IP 合规</p>
          <p className="mt-2 text-2xl font-semibold text-night-50">12 条</p>
          <p className="mt-4 text-sm text-night-200">卡片化展示白名单健康状态。</p>
        </div>
      </div>
    </section>
  </div>
);

export default Dashboard;
