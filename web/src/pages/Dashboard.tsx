import { SparklesIcon, ShieldCheckIcon, CpuChipIcon } from '@heroicons/react/24/outline';

const metrics = [
  {
    title: '认证成功率',
    value: '99.7%',
    change: '+1.2% 本周',
    icon: ShieldCheckIcon
  },
  {
    title: '活跃设备',
    value: '42',
    change: '6 台待审批',
    icon: CpuChipIcon
  },
  {
    title: '资产变更',
    value: '18',
    change: '今日新增 3 条',
    icon: SparklesIcon
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
          <p className="text-xs uppercase tracking-[0.18em] text-night-300">指纹匹配</p>
          <p className="mt-2 text-2xl font-semibold text-night-50">38/38</p>
          <p className="mt-4 text-sm text-night-200">所有今日登录设备均通过硬件指纹验证。</p>
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
