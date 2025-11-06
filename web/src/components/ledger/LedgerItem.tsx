import { DocumentTextIcon, FolderIcon, TableCellsIcon } from '@heroicons/react/24/outline';
import { clsx } from 'clsx';
import { useMemo } from 'react';
import type { DragEvent } from 'react';

import type { WorkspaceNode } from './types';

type LedgerItemProps = {
  node: WorkspaceNode;
  depth?: number;
  active?: boolean;
  onSelect: (node: WorkspaceNode) => void;
  formatTimestamp?: (value?: string) => string;
  draggable?: boolean;
  isDragging?: boolean;
  isDropTarget?: boolean;
  onDragStart?: (node: WorkspaceNode) => void;
  onDragEnd?: () => void;
  onDragOver?: (event: DragEvent<HTMLButtonElement>, node: WorkspaceNode) => void;
  onDragLeave?: (event: DragEvent<HTMLButtonElement>, node: WorkspaceNode) => void;
  onDrop?: (event: DragEvent<HTMLButtonElement>, node: WorkspaceNode) => void;
};

const iconForKind = (kind: WorkspaceNode['kind']) => {
  if (kind === 'folder') {
    return FolderIcon;
  }
  if (kind === 'document') {
    return DocumentTextIcon;
  }
  return TableCellsIcon;
};

const labelForKind = (kind: WorkspaceNode['kind']) => {
  if (kind === 'folder') {
    return '文件夹';
  }
  if (kind === 'document') {
    return '文档';
  }
  return '表格';
};

export const LedgerItem = ({
  node,
  depth = 0,
  active = false,
  onSelect,
  formatTimestamp,
  draggable = false,
  isDragging = false,
  isDropTarget = false,
  onDragStart,
  onDragEnd,
  onDragOver,
  onDragLeave,
  onDrop
}: LedgerItemProps) => {
  const Icon = iconForKind(node.kind);
  const timeText = useMemo(() => (formatTimestamp ? formatTimestamp(node.updatedAt) : ''), [
    formatTimestamp,
    node.updatedAt
  ]);
  const paddingLeft = 12 + depth * 18;
  const label = labelForKind(node.kind);
  const displayName = node.name || '未命名台账';

  const handleDragStart = (event: DragEvent<HTMLButtonElement>) => {
    if (!draggable) {
      return;
    }
    event.dataTransfer.effectAllowed = 'move';
    event.dataTransfer.setData('text/plain', node.id);
    onDragStart?.(node);
  };

  const handleDragOver = (event: DragEvent<HTMLButtonElement>) => {
    if (!draggable) {
      return;
    }
    onDragOver?.(event, node);
  };

  const handleDragLeave = (event: DragEvent<HTMLButtonElement>) => {
    if (!draggable) {
      return;
    }
    onDragLeave?.(event, node);
  };

  const handleDrop = (event: DragEvent<HTMLButtonElement>) => {
    if (!draggable) {
      return;
    }
    onDrop?.(event, node);
  };

  const handleDragEnd = () => {
    if (!draggable) {
      return;
    }
    onDragEnd?.();
  };

  return (
    <button
      type="button"
      onClick={() => onSelect(node)}
      draggable={draggable}
      onDragStart={handleDragStart}
      onDragEnd={handleDragEnd}
      onDragOver={handleDragOver}
      onDragLeave={handleDragLeave}
      onDrop={handleDrop}
      style={{ paddingLeft }}
      className={clsx(
        'ledger-item group mt-1 flex w-full items-center gap-3 rounded-2xl border px-4 py-2 text-left text-sm transition',
        draggable && 'cursor-grab active:cursor-grabbing',
        isDragging && 'opacity-70',
        active
          ? 'border-[var(--accent)] bg-[var(--card-bg)] text-[var(--accent)] shadow-[var(--shadow-sm)]'
          : 'border-white/70 bg-white/50 text-[var(--muted)] hover:border-[var(--accent)]/40 hover:text-[var(--text)]',
        !active && isDropTarget && 'border-[var(--accent)]/60 bg-[var(--accent)]/10 text-[var(--accent)] shadow-[var(--shadow-sm)]'
      )}
    >
      <Icon className="h-4 w-4 shrink-0 text-[var(--accent)] group-hover:scale-105 transition-transform" />
      <div className="flex min-w-0 flex-1 flex-col">
        <span className="truncate font-medium text-[var(--text)]">{displayName}</span>
        <div className="mt-1 flex flex-wrap items-center gap-2 text-[10px] text-[var(--muted)]">
          <span className="rounded-full border border-[var(--muted)]/30 bg-white/70 px-2 py-0.5">{label}</span>
          {node.rowCount !== undefined && (
            <span className="rounded-full border border-[var(--muted)]/30 bg-white/70 px-2 py-0.5">{node.rowCount} 行</span>
          )}
          {timeText && <span>{timeText}</span>}
        </div>
      </div>
    </button>
  );
};
