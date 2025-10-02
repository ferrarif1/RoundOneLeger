import { ReactNode, useEffect, useState } from 'react';
import { Bars3Icon, XMarkIcon } from '@heroicons/react/24/outline';
import { clsx } from 'clsx';

interface LedgerLayoutProps {
  sidebar: ReactNode;
  editor: ReactNode;
}

export const LedgerLayout = ({ sidebar, editor }: LedgerLayoutProps) => {
  const [sidebarOpen, setSidebarOpen] = useState(false);

  useEffect(() => {
    if (typeof window === 'undefined') {
      return;
    }
    const handleResize = () => {
      if (window.innerWidth >= 1024) {
        setSidebarOpen(false);
      }
    };
    window.addEventListener('resize', handleResize);
    return () => window.removeEventListener('resize', handleResize);
  }, []);

  return (
    <>
      <div className="eidos-ledger-mobile-toggle">
        <button
          type="button"
          onClick={() => setSidebarOpen((prev) => !prev)}
          className={clsx('eidos-btn eidos-btn--ghost eidos-ledger-toggle-button')}
        >
          {sidebarOpen ? <XMarkIcon className="h-4 w-4" /> : <Bars3Icon className="h-4 w-4" />}
          {sidebarOpen ? '收起台账列表' : '显示台账列表'}
        </button>
      </div>
      <div className={clsx('eidos-ledger-container', sidebarOpen && 'sidebar-open')}>
        <aside className={clsx('eidos-ledger-list', sidebarOpen && 'is-open')}>
          <div className="eidos-ledger-list-scroll">{sidebar}</div>
        </aside>
        <section className="eidos-ledger-editor">{editor}</section>
      </div>
    </>
  );
};
