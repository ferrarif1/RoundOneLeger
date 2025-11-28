import { useLocation, useNavigate } from 'react-router-dom';
import { useSession } from '../hooks/useSession';
import { BellIcon, PowerIcon, UserCircleIcon } from '@heroicons/react/24/outline';

const titles: Record<string, string> = {
  '/dashboard': '实时概览',
  '/assets': '台账编排',
  '/ip-allowlist': 'IP 白名单',
  '/audit-logs': '审计追踪',
  '/users': '用户中心'
};

const TopBar = () => {
  const { pathname } = useLocation();
  const navigate = useNavigate();
  const { setToken, username, admin } = useSession();

  const handleLogout = () => {
    setToken(null);
    navigate('/login', { replace: true });
  };

  return (
    <header className="sticky top-0 z-30 border-b border-[var(--line)] bg-white">
      <div className="flex items-center justify-between px-6 py-4">
        <div className="min-w-0">
          <p className="text-xs uppercase tracking-[0.24em] text-[var(--muted)]">控制中心</p>
          <h1 className="mt-1 text-xl font-semibold text-[var(--text)] truncate">{titles[pathname] ?? '控制台'}</h1>
        </div>
        <div className="flex items-center gap-3 text-[var(--muted)]">
          <button
            className="rounded-full border border-[var(--line)] bg-white p-2.5"
            aria-label="通知"
          >
            <BellIcon className="h-5 w-5" />
          </button>
          <div className="flex min-w-0 items-center gap-2 rounded-full border border-[var(--line)] bg-white px-3 py-2">
            <UserCircleIcon className="h-6 w-6 text-[var(--accent)]" />
            <div className="min-w-0 text-left">
              <p className="text-xs uppercase tracking-wider text-[var(--muted)] truncate">
                {admin ? '管理员' : '用户'}
              </p>
              <p className="text-sm font-semibold text-[var(--text)] truncate">{username || '未命名账户'}</p>
            </div>
          </div>
          <button
            onClick={handleLogout}
            className="button-primary !px-4 !text-xs !font-semibold"
            aria-label="退出登录"
          >
            <PowerIcon className="h-4 w-4" />
            退出
          </button>
        </div>
      </div>
    </header>
  );
};

export default TopBar;
