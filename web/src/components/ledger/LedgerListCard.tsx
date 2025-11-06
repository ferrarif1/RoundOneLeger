import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import type { DragEvent } from 'react';
import { MagnifyingGlassIcon, PlusIcon } from '@heroicons/react/24/outline';
import type { ComponentType, SVGProps } from 'react';

import type { WorkspaceKind, WorkspaceNode } from './types';
import { LedgerItem } from './LedgerItem';

type CreationOption = {
  kind: WorkspaceKind;
  label: string;
  description: string;
  icon: ComponentType<SVGProps<SVGSVGElement>>;
};

type LedgerListCardProps = {
  items: WorkspaceNode[];
  selectedId?: string | null;
  onSelect: (node: WorkspaceNode) => void;
  onCreate: (kind: WorkspaceKind) => void;
  search: string;
  onSearchChange: (value: string) => void;
  creationOptions: CreationOption[];
  formatTimestamp?: (value?: string) => string;
  onMove?: (sourceId: string, targetParentId: string | null) => Promise<void> | void;
  canMove?: (sourceId: string, targetParentId: string | null) => boolean;
};

const filterTree = (nodes: WorkspaceNode[], keyword: string): WorkspaceNode[] => {
  if (!keyword.trim()) {
    return nodes;
  }
  const lower = keyword.trim().toLowerCase();
  const walk = (list: WorkspaceNode[]): WorkspaceNode[] => {
    const result: WorkspaceNode[] = [];
    list.forEach((node) => {
      const matches = (node.name || '').toLowerCase().includes(lower);
      const children = node.children ? walk(node.children) : [];
      if (matches || children.length) {
        result.push({
          ...node,
          children: children.length ? children : undefined
        });
      }
    });
    return result;
  };
  return walk(nodes);
};

export const LedgerListCard = ({
  items,
  selectedId,
  onSelect,
  onCreate,
  search,
  onSearchChange,
  creationOptions,
  formatTimestamp,
  onMove,
  canMove
}: LedgerListCardProps) => {
  const [menuOpen, setMenuOpen] = useState(false);
  const menuRef = useRef<HTMLDivElement | null>(null);
  const menuButtonRef = useRef<HTMLButtonElement | null>(null);
  const [draggingId, setDraggingId] = useState<string | null>(null);
  const [dropTargetId, setDropTargetId] = useState<string | null>(null);
  const [rootActive, setRootActive] = useState(false);

  useEffect(() => {
    const handleClick = (event: MouseEvent) => {
      if (!menuOpen) {
        return;
      }
      const target = event.target as Node;
      if (menuRef.current?.contains(target) || menuButtonRef.current?.contains(target)) {
        return;
      }
      setMenuOpen(false);
    };
    document.addEventListener('mousedown', handleClick);
    return () => document.removeEventListener('mousedown', handleClick);
  }, [menuOpen]);

  const filtered = useMemo(() => filterTree(items, search), [items, search]);

  const resetDragState = useCallback(() => {
    setDraggingId(null);
    setDropTargetId(null);
    setRootActive(false);
  }, []);

  const handleDragStart = useCallback(
    (node: WorkspaceNode) => {
      if (!onMove) {
        return;
      }
      setDraggingId(node.id);
      setDropTargetId(null);
      setRootActive(false);
    },
    [onMove]
  );

  const handleDragEnd = useCallback(() => {
    resetDragState();
  }, [resetDragState]);

  const handleDragOverNode = useCallback(
    (event: DragEvent<HTMLButtonElement>, node: WorkspaceNode) => {
      if (!draggingId || !onMove) {
        return;
      }
      if (draggingId === node.id) {
        return;
      }
      if (node.kind !== 'folder') {
        return;
      }
      const allowed = canMove ? canMove(draggingId, node.id) : true;
      if (!allowed) {
        return;
      }
      event.preventDefault();
      event.dataTransfer.dropEffect = 'move';
      if (dropTargetId !== node.id) {
        setDropTargetId(node.id);
      }
      if (rootActive) {
        setRootActive(false);
      }
    },
    [canMove, draggingId, dropTargetId, onMove, rootActive]
  );

  const handleDragLeaveNode = useCallback(
    (event: DragEvent<HTMLButtonElement>, node: WorkspaceNode) => {
      if (!draggingId) {
        return;
      }
      const nextTarget = event.relatedTarget as Node | null;
      if (!nextTarget || !event.currentTarget.contains(nextTarget)) {
        setDropTargetId((prev) => (prev === node.id ? null : prev));
      }
    },
    [draggingId]
  );

  const handleDropOnNode = useCallback(
    async (event: DragEvent<HTMLButtonElement>, node: WorkspaceNode) => {
      if (!draggingId || !onMove) {
        return;
      }
      if (draggingId === node.id) {
        return;
      }
      if (node.kind !== 'folder') {
        return;
      }
      const allowed = canMove ? canMove(draggingId, node.id) : true;
      if (!allowed) {
        return;
      }
      event.preventDefault();
      resetDragState();
      await onMove(draggingId, node.id);
    },
    [canMove, draggingId, onMove, resetDragState]
  );

  const handleDragOverRoot = useCallback(
    (event: DragEvent<HTMLDivElement>) => {
      if (!draggingId || !onMove) {
        return;
      }
      const allowed = canMove ? canMove(draggingId, null) : true;
      if (!allowed) {
        if (rootActive) {
          setRootActive(false);
        }
        return;
      }
      event.preventDefault();
      event.dataTransfer.dropEffect = 'move';
      if (!rootActive) {
        setRootActive(true);
      }
      if (dropTargetId !== null) {
        setDropTargetId(null);
      }
    },
    [canMove, draggingId, dropTargetId, onMove, rootActive]
  );

  const handleDropOnRoot = useCallback(
    async (event: DragEvent<HTMLDivElement>) => {
      if (!draggingId || !onMove) {
        return;
      }
      const allowed = canMove ? canMove(draggingId, null) : true;
      if (!allowed) {
        return;
      }
      event.preventDefault();
      resetDragState();
      await onMove(draggingId, null);
    },
    [canMove, draggingId, onMove, resetDragState]
  );

  const handleRootDragLeave = useCallback(() => {
    if (rootActive) {
      setRootActive(false);
    }
  }, [rootActive]);

  const showRootDrop = useMemo(() => {
    if (!draggingId || !onMove) {
      return false;
    }
    return canMove ? canMove(draggingId, null) : true;
  }, [canMove, draggingId, onMove]);

  const renderNodes = (nodes: WorkspaceNode[], depth = 0) =>
    nodes.map((node) => {
      const droppable = Boolean(
        onMove &&
          draggingId &&
          node.kind === 'folder' &&
          draggingId !== node.id &&
          (canMove ? canMove(draggingId, node.id) : true)
      );
      return (
        <div key={node.id}>
          <LedgerItem
            node={node}
            depth={depth}
            active={node.id === selectedId}
            onSelect={onSelect}
            formatTimestamp={formatTimestamp}
            draggable={Boolean(onMove)}
            isDragging={draggingId === node.id}
            isDropTarget={dropTargetId === node.id}
            onDragStart={handleDragStart}
            onDragEnd={handleDragEnd}
            onDragOver={droppable ? handleDragOverNode : undefined}
            onDragLeave={droppable ? handleDragLeaveNode : undefined}
            onDrop={droppable ? handleDropOnNode : undefined}
          />
          {node.children?.length ? <div>{renderNodes(node.children, depth + 1)}</div> : null}
        </div>
      );
    });

  return (
    <div className="ledger-list-card">
      <div className="flex items-center justify-between">
        <h2 className="text-base font-semibold text-[var(--text)]">台账列表</h2>
        <button
          ref={menuButtonRef}
          type="button"
          className="eidos-btn"
          onClick={() => setMenuOpen((prev) => !prev)}
        >
          <PlusIcon className="h-4 w-4" />
          新建
        </button>
      </div>
      <div className="mt-4">
        <div className="eidos-search">
          <MagnifyingGlassIcon className="h-4 w-4 text-[var(--muted)]" />
          <input
            value={search}
            onChange={(event) => onSearchChange(event.target.value)}
            placeholder="搜索台账或文件夹"
          />
        </div>
      </div>
      {showRootDrop ? (
        <div
          className={`mt-4 rounded-2xl border border-dashed px-4 py-3 text-sm transition ${
            rootActive
              ? 'border-[var(--accent)] bg-[var(--accent)]/10 text-[var(--accent)]'
              : 'border-black/10 text-[var(--muted)]'
          }`}
          onDragOver={handleDragOverRoot}
          onDragLeave={handleRootDragLeave}
          onDrop={handleDropOnRoot}
        >
          拖放到此处移出文件夹
        </div>
      ) : null}
      {menuOpen && (
        <div
          ref={menuRef}
          className="z-10 mt-3 rounded-2xl bg-white/95 shadow-[var(--shadow-soft)] ring-1 ring-black/5"
        >
          <ul className="divide-y divide-black/5 text-sm text-[var(--text)]">
            {creationOptions.map((option) => (
              <li key={option.kind}>
                <button
                  type="button"
                  onClick={() => {
                    onCreate(option.kind);
                    setMenuOpen(false);
                  }}
                  className="flex w-full items-start gap-3 px-4 py-3 text-left transition hover:bg-[var(--accent)]/10"
                >
                  <option.icon className="mt-0.5 h-4 w-4 text-[var(--accent)]" />
                  <span>
                    <span className="block font-medium">{option.label}</span>
                    <span className="mt-1 block text-xs text-[var(--muted)]">{option.description}</span>
                  </span>
                </button>
              </li>
            ))}
          </ul>
        </div>
      )}
      <div className="mt-4 flex-1 overflow-y-auto pr-1">
        {filtered.length ? (
          <div className="space-y-1">{renderNodes(filtered)}</div>
        ) : (
          <div className="rounded-2xl border border-dashed border-[var(--muted)]/30 bg-white/70 p-6 text-center text-sm text-[var(--muted)]">
            暂无匹配台账，尝试调整搜索或新建一个。
          </div>
        )}
      </div>
    </div>
  );
};
