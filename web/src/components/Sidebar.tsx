import { NavLink } from 'react-router-dom';
import {
  ChartPieIcon,
  CpuChipIcon,
  DocumentTextIcon,
  ShieldCheckIcon,
  SparklesIcon,
  Squares2X2Icon
} from '@heroicons/react/24/outline';

const links = [
  { to: '/dashboard', label: '概览', icon: ChartPieIcon },
  { to: '/assets', label: '台账', icon: Squares2X2Icon },
  { to: '/devices', label: '设备', icon: CpuChipIcon },
  { to: '/ip-allowlist', label: 'IP 白名单', icon: ShieldCheckIcon },
  { to: '/audit-logs', label: '审计日志', icon: DocumentTextIcon }
];

const Sidebar = () => (
  <aside className="hidden lg:flex w-72 xl:w-80 flex-col border-r border-night-700/60 bg-white/70 backdrop-blur-2xl">
    <div className="px-6 py-8">
      <div className="flex items-center gap-3 text-night-50">
        <SparklesIcon className="h-7 w-7 text-neon-500" />
        <div className="min-w-0">
          <p className="text-xs uppercase tracking-[0.32em] text-night-300">Ledger</p>
          <p className="mt-1 text-xl font-semibold tracking-tight">Eidos Control</p>
        </div>
      </div>
    </div>
    <nav className="flex-1 space-y-2 px-4">
      {links.map(({ to, label, icon: Icon }) => (
        <NavLink
          key={to}
          to={to}
          className={({ isActive }) =>
            [
              'flex items-center gap-3 px-4 py-3 rounded-2xl transition-all duration-200 bg-white/80 border border-white shadow-sm min-w-0',
              isActive
                ? 'border-neon-500/60 text-night-50 shadow-glow'
                : 'hover:border-night-600 hover:shadow-glow hover:text-night-50'
            ].join(' ')
          }
        >
          <Icon className="h-5 w-5 text-night-200" />
          <span className="text-sm font-medium tracking-wide truncate">{label}</span>
        </NavLink>
      ))}
    </nav>
    <div className="px-6 py-6 text-xs text-night-300">
      柔和的卡片式分层结构，营造类似样例界面的轻盈触感。
    </div>
  </aside>
);

export default Sidebar;
