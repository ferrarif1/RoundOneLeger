/* eslint-disable @typescript-eslint/no-unused-vars, no-unused-vars */
import { CommandLineIcon, MagnifyingGlassIcon, UserCircleIcon } from '@heroicons/react/24/outline';
import { clsx } from 'clsx';

type TopBarLiteProps = {
  title?: string;
  onSearch?: (value: string) => void;
  onCommand?: () => void;
  onProfile?: () => void;
};

export const TopBarLite = ({ title = '控制台', onSearch, onCommand, onProfile }: TopBarLiteProps) => {
  return (
    <header className="flex items-center justify-between border-b border-[var(--c-line)] bg-white px-6 py-3">
      <div className="min-w-0">
        <p className="text-xs uppercase tracking-[0.2em] text-[var(--c-muted)]">工作区</p>
        <h1 className="mt-1 truncate text-lg font-semibold text-[var(--c-text)]">{title}</h1>
      </div>
      <div className="flex items-center gap-3">
        <div className="relative w-64">
          <MagnifyingGlassIcon className="pointer-events-none absolute left-3 top-2.5 h-4 w-4 text-[var(--c-muted)]" />
          <input
            type="search"
            placeholder="搜索或跳转"
            className="w-full rounded-full border border-[var(--c-line)] bg-white pl-9 pr-3 py-2 text-sm text-[var(--c-text)] focus:border-[var(--c-line-strong)] focus:outline-none"
            onChange={(e) => onSearch?.(e.target.value)}
          />
        </div>
        <button
          type="button"
          onClick={onCommand}
          className={clsx(
            'flex items-center gap-2 rounded-full border border-[var(--c-line)] bg-white px-3 py-2 text-sm font-medium text-[var(--c-text)]'
          )}
        >
          <CommandLineIcon className="h-4 w-4 text-[var(--c-muted)]" />
          Cmd/Ctrl + K
        </button>
        <button
          type="button"
          onClick={onProfile}
          className="flex items-center gap-2 rounded-full border border-[var(--c-line)] bg-white px-3 py-2 text-sm text-[var(--c-text)]"
        >
          <UserCircleIcon className="h-5 w-5 text-[var(--c-muted)]" />
          账户
        </button>
      </div>
    </header>
  );
};
