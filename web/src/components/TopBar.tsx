import { useLocation, useNavigate } from 'react-router-dom';
import { useSession } from '../hooks/useSession';
import { BellIcon, PowerIcon, UserCircleIcon } from '@heroicons/react/24/outline';

const titles: Record<string, string> = {
  '/dashboard': '实时概览',
  '/assets': '台账编排',
  '/devices': '终端设备',
  '/ip-allowlist': 'IP 白名单',
  '/audit-logs': '审计追踪'
};

const TopBar = () => {
  const { pathname } = useLocation();
  const navigate = useNavigate();
  const { setToken } = useSession();

  const handleLogout = () => {
    setToken(null);
    navigate('/login', { replace: true });
  };

  return (
    <header className="sticky top-0 z-30 border-b border-night-700/60 bg-white/80 backdrop-blur-xl">
      <div className="flex items-center justify-between px-6 py-4">
        <div className="min-w-0">
          <p className="text-xs uppercase tracking-[0.28em] text-night-300">控制中心</p>
          <h1 className="text-2xl font-semibold text-night-50 truncate">{titles[pathname] ?? '控制台'}</h1>
        </div>
        <div className="flex items-center gap-3 text-night-200">
          <button className="rounded-full border border-white bg-white/80 p-2 shadow-sm transition-colors hover:text-night-50" aria-label="通知">
            <BellIcon className="h-5 w-5" />
          </button>
          <div className="flex min-w-0 items-center gap-2 rounded-full border border-white bg-white/90 px-3 py-2 shadow-sm">
            <UserCircleIcon className="h-6 w-6 text-neon-500" />
            <div className="min-w-0 text-left">
              <p className="text-xs uppercase tracking-wider text-night-300 truncate">Admin</p>
              <p className="text-sm font-medium text-night-50 truncate">主控台</p>
            </div>
          </div>
          <button
            onClick={handleLogout}
            className="button-primary !px-4 !text-xs !font-semibold"
            aria-label="退出登录"
          >
            <PowerIcon className="h-4 w-4" />
          </button>
        </div>
      </div>
    </header>
  );
};

export default TopBar;
