import { NavLink } from 'react-router-dom';
import { clsx } from 'clsx';
import {
  ChartPieIcon,
  ChevronDoubleLeftIcon,
  DocumentTextIcon,
  ShieldCheckIcon,
  SparklesIcon,
  Squares2X2Icon,
  UserGroupIcon
} from '@heroicons/react/24/outline';

const links = [
  { to: '/dashboard', label: '概览', icon: ChartPieIcon },
  { to: '/assets', label: '台账', icon: Squares2X2Icon },
  { to: '/ip-allowlist', label: 'IP 白名单', icon: ShieldCheckIcon },
  { to: '/audit-logs', label: '审计日志', icon: DocumentTextIcon },
  { to: '/users', label: '用户中心', icon: UserGroupIcon }
];

type SidebarProps = {
  collapsed: boolean;
  onToggle: () => void;
};

const Sidebar = ({ collapsed, onToggle }: SidebarProps) => (
  <aside
    className={clsx(
      'hidden lg:flex flex-col border-r border-[rgba(20,20,20,0.08)] bg-[var(--bg-subtle)]/80 backdrop-blur-md transition-[width] duration-300 ease-out',
      collapsed ? 'w-20' : 'w-72 xl:w-80'
    )}
  >
    <div className="px-6 py-8 border-b border-[rgba(20,20,20,0.08)]">
      <div className={clsx('flex items-center text-[var(--text)]', collapsed ? 'justify-center' : 'gap-3')}>
        <div className="rounded-2xl border border-[rgba(20,20,20,0.12)] bg-white p-2.5 shadow-sm">
          <SparklesIcon className="h-6 w-6 text-[var(--accent)]" />
        </div>
        <div className={clsx('min-w-0 transition-opacity duration-200', collapsed && 'opacity-0')}>
          <p className="text-xs uppercase tracking-[0.32em] text-[rgba(20,20,20,0.45)]">RoundOneLeger</p>
          <p className="mt-1 text-xl font-semibold tracking-tight text-[var(--text)]">RoundOneLeger</p>
        </div>
      </div>
      <button
        type="button"
        onClick={onToggle}
        aria-pressed={collapsed}
        aria-label={collapsed ? '展开菜单栏' : '折叠菜单栏'}
        className={clsx(
          'mt-5 inline-flex w-full items-center justify-center gap-2 rounded-full border-2 border-[#0f62fe] bg-white/95 px-5 py-2.5 text-sm font-semibold tracking-[0.32em] text-[#0f62fe] shadow-[0_12px_30px_rgba(15,98,254,0.18)] transition-all duration-200 hover:-translate-y-0.5 hover:bg-white focus:outline-none focus-visible:ring-4 focus-visible:ring-[#0f62fe]/20',
          collapsed && 'px-3 tracking-[0.2em]'
        )}
      >
        <ChevronDoubleLeftIcon
          className={clsx(
            'h-4 w-4 text-[#0f62fe] transition-transform',
            collapsed && 'rotate-180'
          )}
        />
        <span className={clsx('transition-opacity duration-200', collapsed && 'sr-only')}>折叠菜单</span>
      </button>
    </div>
    <nav className={clsx('flex-1 space-y-2 px-4 py-6', collapsed && 'px-2')}>
      {links.map(({ to, label, icon: Icon }) => (
        <NavLink
          key={to}
          to={to}
          className={({ isActive }) =>
            clsx(
              'group flex items-center gap-3 rounded-2xl border text-sm font-medium tracking-wide transition-all duration-200 min-w-0',
              collapsed ? 'justify-center px-2 py-3' : 'px-4 py-3',
              isActive
                ? 'border-black bg-black text-white shadow-[0_16px_32px_rgba(0,0,0,0.18)]'
                : 'border-[rgba(20,20,20,0.12)] bg-white hover:-translate-y-0.5 hover:border-black/80 hover:shadow-[0_18px_34px_rgba(0,0,0,0.12)]'
            )
          }
        >
          {({ isActive }) => (
            <>
              <Icon
                className={[
                  'h-5 w-5 transition-colors',
                  isActive
                    ? 'text-white'
                    : 'text-[rgba(20,20,20,0.55)] group-hover:text-black'
                ].join(' ')}
              />
              <span className={clsx('truncate transition-opacity duration-150', collapsed && 'sr-only')}>{label}</span>
            </>
          )}
        </NavLink>
      ))}
    </nav>
    <div className={clsx('px-6 pb-8 text-xs leading-relaxed text-[rgba(20,20,20,0.55)]', collapsed && 'sr-only')}>
      采用黑白线框与圆角面板，构建类似 ChatGPT 控制台的沉浸式空间感。
    </div>
  </aside>
);

export default Sidebar;
