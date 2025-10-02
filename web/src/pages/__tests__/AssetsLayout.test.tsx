import { render, screen, fireEvent } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';

import { LedgerLayout } from '../../components/ledger/LedgerLayout';
import { LedgerListCard } from '../../components/ledger/LedgerListCard';
import { LedgerEditorCard } from '../../components/ledger/LedgerEditorCard';
import type { WorkspaceNode } from '../../components/ledger/types';

describe('ledger layout', () => {
  const nodes: WorkspaceNode[] = [
    { id: '1', name: '表格台账', kind: 'sheet', children: [], updatedAt: '2024-05-01T00:00:00Z' },
    { id: '2', name: '审批流程', kind: 'document', children: [] }
  ];

  it('renders list and editor regions', () => {
    render(
      <LedgerLayout
        sidebar={
          <LedgerListCard
            items={nodes}
            selectedId="1"
            onSelect={vi.fn()}
            onCreate={vi.fn()}
            search=""
            onSearchChange={vi.fn()}
            creationOptions={[]}
          />
        }
        editor={<LedgerEditorCard title={<div>编辑区</div>}>正文</LedgerEditorCard>}
      />
    );

    expect(screen.getByText('表格台账')).toBeInTheDocument();
    expect(screen.getByText('编辑区')).toBeInTheDocument();
    expect(screen.getByText('正文')).toBeInTheDocument();
  });

  it('filters ledger list by search input', () => {
    const handleSelect = vi.fn();
    const { rerender } = render(
      <LedgerListCard
        items={nodes}
        selectedId="1"
        onSelect={handleSelect}
        onCreate={vi.fn()}
        search=""
        onSearchChange={vi.fn()}
        creationOptions={[]}
      />
    );

    expect(screen.getByText('审批流程')).toBeInTheDocument();

    rerender(
      <LedgerListCard
        items={nodes}
        selectedId="1"
        onSelect={handleSelect}
        onCreate={vi.fn()}
        search="审批"
        onSearchChange={vi.fn()}
        creationOptions={[]}
      />
    );

    expect(screen.queryByText('表格台账')).not.toBeInTheDocument();
    expect(screen.getByText('审批流程')).toBeInTheDocument();
  });

  it('calls select when clicking ledger card', () => {
    const handleSelect = vi.fn();
    render(
      <LedgerListCard
        items={nodes}
        selectedId={null}
        onSelect={handleSelect}
        onCreate={vi.fn()}
        search=""
        onSearchChange={vi.fn()}
        creationOptions={[]}
      />
    );

    fireEvent.click(screen.getByText('表格台账'));
    expect(handleSelect).toHaveBeenCalled();
  });
});
