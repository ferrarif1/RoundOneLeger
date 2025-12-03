/* eslint-disable @typescript-eslint/no-unused-vars, no-unused-vars */
import type React from 'react';
import { Fragment, type JSX } from 'react';
import { ChevronRightIcon, FolderIcon, PlusIcon } from '@heroicons/react/24/outline';
import { clsx } from 'clsx';

export type NavNode = {
  id: string;
  title: string;
  icon?: React.ComponentType<React.SVGProps<SVGSVGElement>>;
  children?: NavNode[];
  isExpanded?: boolean;
  isActive?: boolean;
};

type SidebarNavProps = {
  tree: NavNode[];
  onToggle: (id: string) => void;
  onSelect: (id: string) => void;
  onCreate?: (parentId?: string) => void;
  className?: string;
};

const NodeRow = ({ node, depth, onToggle, onSelect, onCreate }: SidebarNavProps & { node: NavNode; depth: number }) => {
  const Icon = node.icon ?? FolderIcon;
  const hasChildren = Boolean(node.children?.length);
  return (
    <div
      className={clsx(
        'group flex items-center gap-2 rounded-md px-2 py-1.5 text-sm text-[var(--c-text)]',
        node.isActive ? 'bg-[var(--c-hover)] font-semibold' : 'hover:bg-[var(--c-hover)]'
      )}
      style={{ paddingLeft: 8 + depth * 12 }}
    >
      <button
        type="button"
        aria-label="展开/折叠"
        className="flex h-6 w-6 items-center justify-center text-[var(--c-muted)]"
        onClick={() => onToggle(node.id)}
      >
        {hasChildren && <ChevronRightIcon className={clsx('h-4 w-4 transition-transform', node.isExpanded && 'rotate-90')} />}
      </button>
      <button
        type="button"
        className="flex flex-1 items-center gap-2 text-left"
        onClick={() => onSelect(node.id)}
      >
        <Icon className="h-4 w-4 text-[var(--c-muted)]" />
        <span className="truncate">{node.title}</span>
      </button>
      {onCreate && (
        <button
          type="button"
          aria-label="新建子项"
          className="invisible group-hover:visible text-[var(--c-muted)] hover:text-[var(--c-text)]"
          onClick={() => onCreate(node.id)}
        >
          <PlusIcon className="h-4 w-4" />
        </button>
      )}
    </div>
  );
};

export const SidebarNav = ({ tree, onToggle, onSelect, onCreate, className }: SidebarNavProps) => {
  const renderTree = (nodes: NavNode[], depth = 0): JSX.Element => (
    <Fragment>
      {nodes.map((node) => (
        <Fragment key={node.id}>
          <NodeRow node={node} depth={depth} tree={tree} onToggle={onToggle} onSelect={onSelect} onCreate={onCreate} />
          {node.isExpanded && node.children?.length ? renderTree(node.children, depth + 1) : null}
        </Fragment>
      ))}
    </Fragment>
  );

  return (
    <aside className={clsx('flex h-full w-72 flex-col border-r border-[var(--c-line)] bg-[var(--c-surface-subtle)] px-2 py-3', className)}>
      <div className="mb-3 flex items-center justify-between px-2 text-xs font-semibold uppercase tracking-[0.2em] text-[var(--c-muted)]">
        <span>导航</span>
        {onCreate && (
          <button
            type="button"
            className="flex items-center gap-1 rounded-full border border-[var(--c-line)] bg-white px-2 py-1 text-[11px] font-medium text-[var(--c-text)]"
            onClick={() => onCreate()}
          >
            <PlusIcon className="h-3 w-3" />
            新建
          </button>
        )}
      </div>
      <div className="space-y-1 overflow-y-auto">{renderTree(tree)}</div>
    </aside>
  );
};
