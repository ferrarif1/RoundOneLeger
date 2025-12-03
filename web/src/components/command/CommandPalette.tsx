/* eslint-disable @typescript-eslint/no-unused-vars, no-unused-vars */
import { useEffect } from 'react';
import { CommandLineIcon, MagnifyingGlassIcon } from '@heroicons/react/24/outline';
import { clsx } from 'clsx';

type CommandItem = {
  id: string;
  label: string;
  shortcut?: string;
  onSelect: () => void;
};

type CommandPaletteProps = {
  open: boolean;
  query: string;
  items: CommandItem[];
  activeIndex: number;
  onQueryChange: (value: string) => void;
  onClose: () => void;
  onMove: (direction: 1 | -1) => void;
  onSelect: (index: number) => void;
};

export const CommandPalette = ({
  open,
  query,
  items,
  activeIndex,
  onQueryChange,
  onClose,
  onMove,
  onSelect
}: CommandPaletteProps) => {
  useEffect(() => {
    const handler = (event: KeyboardEvent) => {
      if (!open) return;
      if (event.key === 'Escape') {
        onClose();
      } else if (event.key === 'ArrowDown') {
        event.preventDefault();
        onMove(1);
      } else if (event.key === 'ArrowUp') {
        event.preventDefault();
        onMove(-1);
      } else if (event.key === 'Enter') {
        event.preventDefault();
        onSelect(activeIndex);
      }
    };
    window.addEventListener('keydown', handler);
    return () => window.removeEventListener('keydown', handler);
  }, [open, onClose, onMove, onSelect, activeIndex]);

  if (!open) return null;

  return (
    <div className="fixed inset-0 z-[var(--z-command,100)] flex items-start justify-center bg-[var(--c-overlay)] px-4 py-12">
      <div className="w-full max-w-2xl overflow-hidden rounded-lg border border-[var(--c-line)] bg-white shadow-[var(--shadow-popover)]">
        <div className="flex items-center gap-2 border-b border-[var(--c-line)] px-3 py-2.5">
          <MagnifyingGlassIcon className="h-4 w-4 text-[var(--c-muted)]" />
          <input
            autoFocus
            value={query}
            onChange={(e) => onQueryChange(e.target.value)}
            placeholder="搜索页面、台账或命令"
            className="w-full border-none bg-transparent text-sm text-[var(--c-text)] outline-none"
          />
          <div className="flex items-center gap-1 rounded-full border border-[var(--c-line)] bg-[var(--c-surface-subtle)] px-2 py-1 text-[11px] text-[var(--c-muted)]">
            <CommandLineIcon className="h-3 w-3" />
            Esc
          </div>
        </div>
        <div className="max-h-96 overflow-y-auto">
          {items.length === 0 ? (
            <p className="px-4 py-6 text-sm text-[var(--c-muted)]">无匹配结果</p>
          ) : (
            items.map((item, index) => (
              <button
                key={item.id}
                type="button"
                onClick={() => onSelect(index)}
                className={clsx(
                  'flex w-full items-center justify-between px-4 py-3 text-sm',
                  index === activeIndex ? 'bg-[var(--c-hover)] text-[var(--c-text)]' : 'text-[var(--c-text)]'
                )}
              >
                <span className="truncate text-left">{item.label}</span>
                {item.shortcut && (
                  <span className="rounded border border-[var(--c-line)] px-2 py-1 text-[11px] text-[var(--c-muted)]">
                    {item.shortcut}
                  </span>
                )}
              </button>
            ))
          )}
        </div>
      </div>
    </div>
  );
};
