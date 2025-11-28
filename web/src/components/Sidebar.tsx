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

const expandedWidth = 320;
const collapsedWidth = 80;

const Sidebar = ({ collapsed, onToggle }: SidebarProps) => (
  <div
    className="hidden lg:flex flex-col border-r border-[var(--line)] bg-[#f7f7f7]"
    style={{ width: collapsed ? collapsedWidth : expandedWidth }}
  >
    <div className="px-6 py-6 border-b border-[var(--line)]">
      <div className={clsx('flex items-center text-[var(--text)]', collapsed ? 'justify-center' : 'gap-3')}>
        <div className="rounded-lg border border-[var(--line)] bg-white p-2.5">
          <SparklesIcon className="h-6 w-6 text-[var(--accent)]" />
        </div>
        {!collapsed && (
          <div className="min-w-0">
            <p className="text-xs uppercase tracking-[0.24em] text-[var(--muted)]">RoundOneLeger</p>
            <p className="mt-1 text-lg font-semibold tracking-tight text-[var(--text)]">RoundOneLeger</p>
          </div>
        )}
      </div>
      <button
        type="button"
        onClick={onToggle}
        aria-pressed={collapsed}
        aria-label={collapsed ? '展开菜单栏' : '折叠菜单栏'}
        className={clsx(
          'mt-4 inline-flex w-full items-center justify-center gap-2 rounded-full border border-[var(--line)] bg-white px-4 py-2 text-sm font-medium text-[var(--text)]',
          collapsed && 'px-3'
        )}
      >
        <ChevronDoubleLeftIcon className={clsx('h-4 w-4 text-[var(--muted)]', collapsed && 'rotate-180')} />
        {!collapsed && <span className="whitespace-nowrap">折叠菜单</span>}
      </button>
    </div>
    <nav className={clsx('flex-1 space-y-2 px-4 py-5', collapsed && 'px-2')}>
      {links.map(({ to, label, icon: Icon }) => (
        <NavLink
          key={to}
          to={to}
          className={({ isActive }) =>
            clsx(
              'group flex items-center gap-3 rounded-lg border text-sm font-medium min-w-0',
              collapsed ? 'justify-center px-2 py-2.5' : 'px-4 py-2.5',
              isActive
                ? 'border-[var(--line-strong)] bg-[#eef1f5] text-[var(--text)]'
                : 'border-[var(--line)] bg-white hover:border-[var(--line-strong)] hover:bg-[#f4f6f8]'
            )
          }
        >
          {({ isActive }) => (
            <>
              <Icon
                className={clsx(
                  'h-5 w-5',
                  isActive ? 'text-[var(--accent)]' : 'text-[var(--muted)] group-hover:text-[var(--accent)]'
                )}
              />
              {!collapsed && <span className="truncate">{label}</span>}
            </>
          )}
        </NavLink>
      ))}
    </nav>
    {!collapsed && (
      <div className="px-6 pb-6 text-xs leading-relaxed text-[var(--muted)]">
        浅色侧栏，简洁线框，呼应 Eidos 的极简风格。
      </div>
    )}
  </div>
);

export default Sidebar;
