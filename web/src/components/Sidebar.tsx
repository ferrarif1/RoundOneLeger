import { NavLink } from 'react-router-dom';
import {
  ChartPieIcon,
  DocumentTextIcon,
  ShieldCheckIcon,
  SparklesIcon,
  Squares2X2Icon,
  ClipboardDocumentCheckIcon,
  UserGroupIcon
} from '@heroicons/react/24/outline';

const links = [
  { to: '/dashboard', label: '概览', icon: ChartPieIcon },
  { to: '/assets', label: '台账', icon: Squares2X2Icon },
  { to: '/ip-allowlist', label: 'IP 白名单', icon: ShieldCheckIcon },
  { to: '/audit-logs', label: '审计日志', icon: DocumentTextIcon },
  { to: '/approvals', label: '身份审批', icon: ClipboardDocumentCheckIcon },
  { to: '/users', label: '用户中心', icon: UserGroupIcon }
];

const Sidebar = () => (
  <aside className="hidden lg:flex w-72 xl:w-80 flex-col border-r border-[rgba(20,20,20,0.08)] bg-[var(--bg-subtle)]/80 backdrop-blur-md">
    <div className="px-6 py-8 border-b border-[rgba(20,20,20,0.08)]">
      <div className="flex items-center gap-3 text-[var(--text)]">
        <div className="rounded-2xl border border-[rgba(20,20,20,0.12)] bg-white p-2.5 shadow-sm">
          <SparklesIcon className="h-6 w-6 text-[var(--accent)]" />
        </div>
        <div className="min-w-0">
          <p className="text-xs uppercase tracking-[0.32em] text-[rgba(20,20,20,0.45)]">RoundOneLeger</p>
          <p className="mt-1 text-xl font-semibold tracking-tight text-[var(--text)]">RoundOneLeger</p>
        </div>
      </div>
    </div>
    <nav className="flex-1 space-y-2 px-4 py-6">
      {links.map(({ to, label, icon: Icon }) => (
        <NavLink
          key={to}
          to={to}
          className={({ isActive }) =>
            [
              'group flex items-center gap-3 rounded-2xl border px-4 py-3 text-sm font-medium tracking-wide transition-all duration-200 min-w-0',
              isActive
                ? 'border-black bg-black text-white shadow-[0_16px_32px_rgba(0,0,0,0.18)]'
                : 'border-[rgba(20,20,20,0.12)] bg-white hover:-translate-y-0.5 hover:border-black/80 hover:shadow-[0_18px_34px_rgba(0,0,0,0.12)]'
            ].join(' ')
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
              <span className="truncate">{label}</span>
            </>
          )}
        </NavLink>
      ))}
    </nav>
    <div className="px-6 pb-8 text-xs leading-relaxed text-[rgba(20,20,20,0.55)]">
      采用黑白线框与圆角面板，构建类似 ChatGPT 控制台的沉浸式空间感。
    </div>
  </aside>
);

export default Sidebar;
