import {
  ArrowRightIcon,
  BookOpenIcon,
  ChatBubbleLeftEllipsisIcon,
  CursorArrowRaysIcon,
  InformationCircleIcon,
  MagnifyingGlassIcon,
  ShieldCheckIcon
} from '@heroicons/react/24/outline';

const queryTags = ['MATCH', 'RETURN 1', 'LIMIT 25'];

const featureCards = [
  {
    label: 'GUIDE',
    title: '快速了解 RoundOne Ledger 导航体验',
    description: '通过快捷操作与指令，让每日的台账查询和编辑更高效。',
    action: '开始导览',
    icon: CursorArrowRaysIcon,
    accent: 'bg-[var(--bg-subtle)]'
  },
  {
    label: 'DATASET',
    title: '试用示例数据集，练习查询与记录',
    description: '使用预置样例进行检索与结构化录入，熟悉交互细节。',
    action: '探索样例',
    icon: BookOpenIcon,
    accent: 'bg-[var(--bg-subtle)]'
  },
  {
    label: 'HELP US IMPROVE',
    title: '分享你的反馈',
    description: '提交你的体验或建议，帮助我们持续完善交互与性能。',
    action: '发送反馈',
    icon: ChatBubbleLeftEllipsisIcon,
    accent: 'bg-[var(--bg-subtle)]'
  }
];

const dbPolicies = [
  {
    title: '强制使用内置数据库',
    description: '所有台账操作默认走本地数据库实例，无需手动切换或外部依赖。'
  },
  {
    title: '自动完成本地配置',
    description: '启动时自动准备数据库连接与基础索引，省去手工配置步骤。'
  },
  {
    title: '导入/导出 ZIP 打包',
    description: '迁移与共享数据时，仅接受包含数据库文件与媒体资源的 ZIP 包，确保信息完整。'
  }
];

const Dashboard = () => (
  <div className="space-y-6">
    <section className="rounded-3xl border border-[var(--line)] bg-white/95 p-6 shadow-[0_24px_50px_rgba(0,0,0,0.08)] backdrop-blur-md">
      <div className="flex flex-col gap-4">
        <div className="flex flex-wrap items-center justify-between gap-4">
          <div className="flex flex-wrap items-center gap-3">
            <div className="flex items-center gap-2 rounded-full border border-[var(--line)] bg-[var(--bg-subtle)] px-3 py-1.5">
              <span className="h-2.5 w-2.5 rounded-full bg-[#3ac58a]" aria-hidden />
              <span className="text-sm font-semibold tracking-tight text-[var(--text)]">neo4j/4</span>
            </div>
            <span className="rounded-full border border-[var(--line)] bg-[var(--bg-subtle)] px-3 py-1 text-xs font-semibold uppercase tracking-[0.22em] text-[rgba(20,20,20,0.55)]">
              数据库
            </span>
            <div className="inline-flex items-center gap-2 rounded-full border border-[rgba(20,20,20,0.1)] bg-white px-3 py-1 text-xs text-[rgba(20,20,20,0.55)] shadow-sm">
              <InformationCircleIcon className="h-4 w-4 text-[rgba(20,20,20,0.55)]" />
              <span>默认活跃连接</span>
            </div>
          </div>
          <div className="flex flex-wrap items-center gap-2 text-sm text-[rgba(20,20,20,0.75)]">
            {queryTags.map((tag) => (
              <span
                key={tag}
                className="rounded-full border border-[rgba(20,20,20,0.12)] bg-white px-3 py-1.5 text-xs font-semibold tracking-wide text-[rgba(20,20,20,0.7)] shadow-sm"
              >
                {tag}
              </span>
            ))}
          </div>
        </div>

        <div className="grid gap-4 md:grid-cols-12 md:items-center">
          <div className="md:col-span-8">
            <div className="relative flex items-center rounded-2xl border border-[rgba(20,20,20,0.1)] bg-white px-4 py-3 shadow-[0_12px_30px_rgba(0,0,0,0.06)]">
              <MagnifyingGlassIcon className="h-5 w-5 text-[rgba(20,20,20,0.45)]" />
              <input
                type="text"
                placeholder="match (n) return 1 limit 25"
                className="ml-3 w-full border-0 bg-transparent text-[var(--text)] placeholder:text-[rgba(20,20,20,0.45)] focus:outline-none"
              />
              <button
                type="button"
                className="ml-3 inline-flex items-center gap-1 rounded-full bg-[var(--text)] px-4 py-2 text-sm font-semibold text-white shadow-[0_14px_30px_rgba(0,0,0,0.18)] transition-transform hover:-translate-y-0.5"
              >
                运行
                <ArrowRightIcon className="h-4 w-4" />
              </button>
            </div>
            <p className="mt-2 text-xs text-[rgba(20,20,20,0.55)]">在此输入语句以快速检索或写入台账数据。</p>
          </div>

          <div className="md:col-span-4 grid gap-3 sm:grid-cols-3">
            {[
              { label: 'Nodes', value: '0' },
              { label: 'Relationships', value: '0' },
              { label: 'Property keys', value: '0' }
            ].map((item) => (
              <div
                key={item.label}
                className="rounded-2xl border border-[rgba(20,20,20,0.08)] bg-[var(--bg-subtle)] px-3 py-3 text-center shadow-sm"
              >
                <p className="text-[11px] uppercase tracking-[0.16em] text-[rgba(20,20,20,0.5)]">{item.label}</p>
                <p className="mt-2 text-lg font-semibold text-[var(--text)]">{item.value}</p>
              </div>
            ))}
          </div>
        </div>
      </div>
    </section>

    <section className="rounded-3xl border border-[var(--line)] bg-white/95 p-6 shadow-[0_22px_44px_rgba(0,0,0,0.08)]">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div>
          <p className="text-sm font-semibold uppercase tracking-[0.18em] text-[rgba(20,20,20,0.55)]">welcome</p>
          <h2 className="mt-2 text-2xl font-semibold text-[var(--text)]">快速开启台账之旅</h2>
          <p className="mt-1 text-sm text-[rgba(20,20,20,0.6)]">通过指引、示例数据和反馈入口，加速熟悉 RoundOne Ledger。</p>
        </div>
        <button
          type="button"
          className="inline-flex items-center gap-2 rounded-full border border-[var(--line)] bg-white px-4 py-2 text-sm font-semibold text-[var(--text)] shadow-[0_12px_30px_rgba(0,0,0,0.08)] transition-transform hover:-translate-y-0.5"
        >
          <ShieldCheckIcon className="h-5 w-5" />
          已连接到 RoundOne Ledger
        </button>
      </div>

      <div className="mt-6 grid gap-4 xl:grid-cols-3 md:grid-cols-2">
        {featureCards.map((card) => (
          <article
            key={card.title}
            className="group flex flex-col justify-between rounded-3xl border border-[var(--line)] bg-white p-5 shadow-[0_18px_32px_rgba(0,0,0,0.08)] transition-transform hover:-translate-y-1"
          >
            <div className={`flex h-12 w-12 items-center justify-center rounded-2xl ${card.accent}`}>
              <card.icon className="h-6 w-6 text-[rgba(20,20,20,0.7)]" />
            </div>
            <div className="mt-4 space-y-2">
              <p className="text-xs font-semibold uppercase tracking-[0.2em] text-[rgba(20,20,20,0.55)]">{card.label}</p>
              <h3 className="text-lg font-semibold text-[var(--text)]">{card.title}</h3>
              <p className="text-sm text-[rgba(20,20,20,0.6)]">{card.description}</p>
            </div>
            <button
              type="button"
              className="mt-6 inline-flex items-center gap-1.5 text-sm font-semibold text-[var(--text)] transition-transform group-hover:translate-x-1"
            >
              {card.action}
              <ArrowRightIcon className="h-4 w-4" />
            </button>
          </article>
        ))}
      </div>

      <div className="mt-6 rounded-3xl border border-[var(--line)] bg-[var(--bg-subtle)] p-5 shadow-[0_18px_32px_rgba(0,0,0,0.06)]">
        <p className="text-xs font-semibold uppercase tracking-[0.18em] text-[rgba(20,20,20,0.55)]">数据库策略</p>
        <h3 className="mt-2 text-lg font-semibold text-[var(--text)]">自动使用本地数据库，数据迁移统一打包</h3>
        <div className="mt-4 grid gap-3 md:grid-cols-3 sm:grid-cols-2">
          {dbPolicies.map((policy) => (
            <div
              key={policy.title}
              className="flex flex-col gap-1 rounded-2xl border border-[rgba(20,20,20,0.08)] bg-white/80 p-4 shadow-sm"
            >
              <p className="text-sm font-semibold text-[var(--text)]">{policy.title}</p>
              <p className="text-sm text-[rgba(20,20,20,0.6)]">{policy.description}</p>
            </div>
          ))}
        </div>
      </div>
    </section>
  </div>
);

export default Dashboard;
