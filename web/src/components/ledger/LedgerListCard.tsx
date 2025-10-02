import { useEffect, useMemo, useRef, useState } from 'react';
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
};

const filterTree = (nodes: WorkspaceNode[], keyword: string): WorkspaceNode[] => {
  if (!keyword.trim()) {
    return nodes;
  }
  const lower = keyword.trim().toLowerCase();
  const walk = (list: WorkspaceNode[]): WorkspaceNode[] => {
    return list
      .map((node) => {
        const matches = (node.name || '').toLowerCase().includes(lower);
        const children = node.children ? walk(node.children) : [];
        if (matches || children.length) {
          return { ...node, children };
        }
        return null;
      })
      .filter((node): node is WorkspaceNode => node !== null);
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
  formatTimestamp
}: LedgerListCardProps) => {
  const [menuOpen, setMenuOpen] = useState(false);
  const menuRef = useRef<HTMLDivElement | null>(null);
  const menuButtonRef = useRef<HTMLButtonElement | null>(null);

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

  const renderNodes = (nodes: WorkspaceNode[], depth = 0) =>
    nodes.map((node) => (
      <div key={node.id}>
        <LedgerItem
          node={node}
          depth={depth}
          active={node.id === selectedId}
          onSelect={onSelect}
          formatTimestamp={formatTimestamp}
        />
        {node.children?.length ? <div>{renderNodes(node.children, depth + 1)}</div> : null}
      </div>
    ));

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
