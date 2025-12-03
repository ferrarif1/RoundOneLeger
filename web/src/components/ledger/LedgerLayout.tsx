import { ReactNode, useEffect, useState } from 'react';
import { Bars3Icon, ChevronDoubleLeftIcon, XMarkIcon } from '@heroicons/react/24/outline';
import { clsx } from 'clsx';

interface LedgerLayoutProps {
  sidebar: ReactNode;
  editor: ReactNode;
}

export const LedgerLayout = ({ sidebar, editor }: LedgerLayoutProps) => {
  const [sidebarOpen, setSidebarOpen] = useState(false);
  const [sidebarCollapsed, setSidebarCollapsed] = useState(false);

  useEffect(() => {
    if (typeof window === 'undefined') {
      return;
    }
    const handleResize = () => {
      if (window.innerWidth >= 1024) {
        setSidebarOpen(false);
      } else {
        setSidebarCollapsed(false);
      }
    };
    handleResize();
    window.addEventListener('resize', handleResize);
    return () => window.removeEventListener('resize', handleResize);
  }, []);

  return (
    <>
      <div className="roleger-ledger-mobile-toggle">
        <button
          type="button"
          onClick={() => setSidebarOpen((prev) => !prev)}
          aria-expanded={sidebarOpen}
          className={clsx('roleger-btn roleger-btn--ghost roleger-ledger-toggle-button')}
        >
          {sidebarOpen ? <XMarkIcon className="h-4 w-4" /> : <Bars3Icon className="h-4 w-4" />}
          {sidebarOpen ? '收起台账列表' : '显示台账列表'}
        </button>
      </div>
      <div
        className={clsx(
          'roleger-ledger-container',
          sidebarOpen && 'sidebar-open',
          sidebarCollapsed && 'sidebar-collapsed'
        )}
      >
        <aside className={clsx('roleger-ledger-list', sidebarOpen && 'is-open')}>
          <button
            type="button"
            className={clsx('roleger-ledger-collapse-handle', sidebarCollapsed && 'is-collapsed')}
            onClick={() => setSidebarCollapsed((prev) => !prev)}
            aria-pressed={sidebarCollapsed}
            aria-expanded={!sidebarCollapsed}
            aria-label={sidebarCollapsed ? '展开台账列表' : '折叠台账列表'}
          >
            <ChevronDoubleLeftIcon className={clsx('h-4 w-4', sidebarCollapsed && 'rotate-180')} />
          </button>
          <div className="roleger-ledger-list-scroll">{sidebar}</div>
        </aside>
        <section className="roleger-ledger-editor">{editor}</section>
      </div>
    </>
  );
};
