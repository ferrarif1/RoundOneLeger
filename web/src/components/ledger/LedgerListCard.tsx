import { useEffect, useMemo, useRef, useState } from 'react';
import type React from 'react';
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
  onMove?: (intent: { sourceId: string; targetId: string | null; position: 'before' | 'after' | 'into' }) => void;
  search: string;
  onSearchChange: (value: string) => void;
  creationOptions: CreationOption[];
  formatTimestamp?: (value?: string) => string;
};

const filterTree = (nodes: WorkspaceNode[], keyword: string): WorkspaceNode[] => {
  const trimmed = keyword.trim();
  if (!trimmed) {
    return nodes;
  }
  const lower = trimmed.toLowerCase();
  const walk = (list: WorkspaceNode[]): WorkspaceNode[] => {
    const results: WorkspaceNode[] = [];
    for (const node of list) {
      const matches = (node.name || '').toLowerCase().includes(lower);
      const children = node.children ? walk(node.children) : [];
      const normalizedChildren = children.length ? children : undefined;
      if (matches || normalizedChildren) {
        results.push({ ...node, children: normalizedChildren });
      }
    }
    return results;
  };
  return walk(nodes);
};

export const LedgerListCard = ({
  items,
  selectedId,
  onSelect,
  onCreate,
  onMove,
  search,
  onSearchChange,
  creationOptions,
  formatTimestamp
}: LedgerListCardProps) => {
  const [menuOpen, setMenuOpen] = useState(false);
  const menuRef = useRef<HTMLDivElement | null>(null);
  const menuButtonRef = useRef<HTMLButtonElement | null>(null);
  const [expanded, setExpanded] = useState<Set<string>>(new Set());
  const [draggingId, setDraggingId] = useState<string | null>(null);
  const [hoverTarget, setHoverTarget] = useState<{ id: string; position: 'before' | 'after' } | null>(null);

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

  useEffect(() => {
    if (!expanded.size && filtered.length) {
      const initial = new Set<string>();
      filtered.forEach((node) => {
        if (node.kind === 'folder') {
          initial.add(node.id);
        }
      });
      setExpanded(initial);
    }
  }, [expanded.size, filtered]);

  const toggleFolder = (id: string) => {
    setExpanded((prev) => {
      const next = new Set(prev);
      if (next.has(id)) {
        next.delete(id);
      } else {
        next.add(id);
      }
      return next;
    });
  };

  const renderNodes = (nodes: WorkspaceNode[], depth = 0) =>
    nodes.map((node) => {
      const isFolder = node.kind === 'folder';
      const isExpanded = expanded.has(node.id);
      const hasChildren = Boolean(node.children?.length);
      const showChildren = isFolder && isExpanded && hasChildren;

      return (
        <div key={node.id} className="rounded-xl">
          <div
            draggable
            onDragStart={() => setDraggingId(node.id)}
            onDragEnd={() => {
              setDraggingId(null);
              setHoverTarget(null);
            }}
            onDragOver={(event) => {
              if (!draggingId || draggingId === node.id) {
                return;
              }
              const rect = (event.currentTarget as HTMLDivElement).getBoundingClientRect();
              const offsetY = event.clientY - rect.top;
              const position = offsetY < rect.height / 2 ? 'before' : 'after';
              setHoverTarget({ id: node.id, position });
              event.preventDefault();
            }}
            onDrop={(event: React.DragEvent<HTMLDivElement>) => {
              if (!onMove || !draggingId || draggingId === node.id) return;
              event.preventDefault();
              const hover = hoverTarget?.position ?? 'after';
              // 默认落在文件夹时直接放入；按住 Alt 时维持同级前/后插入
              const intentPosition = isFolder && !event.altKey ? 'into' : hover;
              onMove({ sourceId: draggingId, targetId: node.id, position: intentPosition });
              setDraggingId(null);
              setHoverTarget(null);
            }}
          >
            <LedgerItem
              node={node}
              depth={depth}
              active={node.id === selectedId}
              onSelect={onSelect}
              formatTimestamp={formatTimestamp}
              onToggleFolder={isFolder ? () => toggleFolder(node.id) : undefined}
              expanded={isExpanded}
            />
          </div>
          {hoverTarget?.id === node.id && hoverTarget.position === 'before' && (
            <div className="h-1 rounded-full bg-[var(--accent)]/60" />
          )}
          {showChildren ? <div className="pl-4">{renderNodes(node.children!, depth + 1)}</div> : null}
          {hoverTarget?.id === node.id && hoverTarget.position === 'after' && (
            <div className="h-1 rounded-full bg-[var(--accent)]/60" />
          )}
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
          className="roledger-btn"
          onClick={() => setMenuOpen((prev) => !prev)}
        >
          <PlusIcon className="h-4 w-4" />
          新建
        </button>
      </div>
      <div className="mt-4">
        <div className="roledger-search">
          <MagnifyingGlassIcon className="h-4 w-4 text-[var(--muted)]" />
          <input
            value={search}
            onChange={(event) => onSearchChange(event.target.value)}
            placeholder="搜索台账或文件夹"
          />
        </div>
      </div>
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
                  className="flex w-full items-start gap-3 px-4 py-3 text-left"
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
          <div
            className="space-y-1"
            onDragOver={(event) => {
              if (draggingId) {
                event.preventDefault();
              }
            }}
            onDrop={(event) => {
              if (!onMove || !draggingId) return;
              event.preventDefault();
              onMove({ sourceId: draggingId, targetId: null, position: 'after' });
              setDraggingId(null);
            }}
          >
            {renderNodes(filtered)}
          </div>
        ) : (
          <div className="rounded-2xl border border-dashed border-[var(--muted)]/30 bg-white/70 p-6 text-center text-sm text-[var(--muted)]">
            暂无匹配台账，尝试调整搜索或新建一个。
          </div>
        )}
      </div>
    </div>
  );
};
