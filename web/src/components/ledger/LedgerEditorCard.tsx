import { ReactNode } from 'react';

interface LedgerEditorCardProps {
  title: ReactNode;
  toolbar?: ReactNode;
  status?: ReactNode;
  children: ReactNode;
}

export const LedgerEditorCard = ({ title, toolbar, status, children }: LedgerEditorCardProps) => {
  return (
    <section className="flex h-full flex-col rounded-[var(--radius-lg)] bg-white/95 p-6 shadow-[var(--shadow-soft)]">
      <header className="flex flex-col gap-4 border-b border-black/5 pb-6">
        <div className="flex flex-wrap items-center justify-between gap-4">
          <div className="min-w-0 flex-1 text-[var(--text)]">{title}</div>
          {toolbar}
        </div>
        {status}
      </header>
      <div className="mt-6 flex-1 overflow-y-auto">{children}</div>
    </section>
  );
};
