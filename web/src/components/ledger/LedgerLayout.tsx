import { ReactNode, useState } from 'react';
import { Bars3Icon, XMarkIcon } from '@heroicons/react/24/outline';

interface LedgerLayoutProps {
  sidebar: ReactNode;
  editor: ReactNode;
}

export const LedgerLayout = ({ sidebar, editor }: LedgerLayoutProps) => {
  const [sidebarOpen, setSidebarOpen] = useState(false);

  return (
    <div className="flex min-h-[calc(100vh-64px)] w-full flex-col gap-6 bg-[var(--bg)] px-4 py-6 lg:flex-row lg:px-8">
      <div className="lg:hidden">
        <button
          type="button"
          onClick={() => setSidebarOpen((prev) => !prev)}
          className="flex items-center gap-2 rounded-full bg-white px-4 py-2 text-sm text-[var(--text)] shadow-[var(--shadow-sm)]"
        >
          {sidebarOpen ? <XMarkIcon className="h-4 w-4" /> : <Bars3Icon className="h-4 w-4" />}
          {sidebarOpen ? '收起台账列表' : '显示台账列表'}
        </button>
        {sidebarOpen && <div className="mt-4">{sidebar}</div>}
      </div>
      <div className="hidden lg:block lg:w-[320px] xl:w-[360px]">{sidebar}</div>
      <div className="flex-1 overflow-hidden">{editor}</div>
    </div>
  );
};
