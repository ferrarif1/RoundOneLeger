/* eslint-disable @typescript-eslint/no-unused-vars, no-unused-vars */
import React, { useMemo, useState } from 'react';
import { ArrowsUpDownIcon, PlusIcon } from '@heroicons/react/24/outline';
import { FixedSizeList as List, type ListChildComponentProps } from 'react-window';
import type { Property, RecordItem, ViewColumn } from '../../types/roledger';

export type DataTableProps = {
  properties: Property[];
  columns: ViewColumn[];
  records: RecordItem[];
  activeIds?: string[];
  loading?: boolean;
  onSelect?: (recordId: string) => void;
  onDelete?: (recordId: string) => void | Promise<void>;
  onCellEdit: (recordId: string, propertyId: string, value: unknown) => void;
  onAddRecord: () => void | Promise<void>;
  onReorderColumn?: (sourceId: string, targetId: string) => void;
  onResizeColumn?: (propertyId: string, width: number) => void;
};

const MemoCellInput = React.memo(function MemoCellInput({
  recordId,
  propertyId,
  value,
  onEdit
}: {
  recordId: string;
  propertyId: string;
  value: unknown;
  onEdit: (recordId: string, propertyId: string, value: unknown) => void;
}) {
  const display = value != null ? String(value) : '';
  return (
    <input
      className="w-full truncate rounded-[6px] border border-[var(--c-line)] bg-white px-2 py-1.5 text-sm text-[var(--c-text)] focus:border-[var(--c-line-strong)] focus:outline-none"
      style={{ whiteSpace: 'nowrap', textOverflow: 'ellipsis', overflow: 'hidden' }}
      value={display}
      title={display}
      onChange={(e) => onEdit(recordId, propertyId, e.target.value)}
    />
  );
});

export const DataTable = ({
  properties,
  columns,
  records,
  activeIds = [],
  loading = false,
  onSelect,
  onDelete,
  onCellEdit,
  onAddRecord,
  onReorderColumn,
  onResizeColumn
}: DataTableProps) => {
  const visibleColumns = useMemo(() => columns.filter((col) => col.visible), [columns]);
  const [resizing, setResizing] = useState<{ id: string; startX: number; startWidth: number } | null>(null);

  const handleResizeStart = (e: React.MouseEvent<HTMLSpanElement>, columnId: string) => {
    const col = visibleColumns.find((c) => c.propertyId === columnId);
    if (!col || !onResizeColumn) return;
    setResizing({ id: columnId, startX: e.clientX, startWidth: col.width || 160 });
  };

  const handleMouseMove = (e: React.MouseEvent) => {
    if (!resizing || !onResizeColumn) return;
    const delta = e.clientX - resizing.startX;
    onResizeColumn(resizing.id, Math.max(120, resizing.startWidth + delta));
  };

  const handleMouseUp = () => {
    if (resizing) setResizing(null);
  };

  const columnWidths = visibleColumns.map((c) => c.width || 180);
  const gridTemplate = `80px ${columnWidths.map((w) => `${w}px`).join(' ')} 100px`;

  const itemData = useMemo(
    () => ({
      records,
      visibleColumns,
      properties,
      activeIds,
      onSelect,
      onDelete,
      onCellEdit
    }),
    [records, visibleColumns, properties, activeIds, onSelect, onDelete, onCellEdit]
  );

  const Row = ({ index, style, data }: ListChildComponentProps<typeof itemData>) => {
    const isNewRow = index >= data.records.length;
    if (isNewRow) {
      return (
        <div className="contents" style={style}>
          <div className="border-t border-[var(--c-line)] px-2 py-2" />
          <div
            className="border-t border-[var(--c-line)] px-3 py-2 text-sm text-[var(--c-text)]"
            style={{ gridColumn: `span ${data.visibleColumns.length + 1}` }}
          >
            <button type="button" onClick={onAddRecord} className="flex items-center gap-2">
              <PlusIcon className="h-4 w-4" />
              新建记录
            </button>
          </div>
          <div className="border-t border-[var(--c-line)] px-2 py-2" />
        </div>
      );
    }
    const record = data.records[index];
    const active = data.activeIds.includes(record.id);
    return (
      <div
        className={`contents ${active ? 'bg-[var(--c-hover)]' : ''}`}
        style={style}
        onClick={() => data.onSelect && data.onSelect(record.id)}
      >
        <div className="border-t border-[var(--c-line)] px-2 py-3 text-[11px] text-[var(--c-muted)]">
          {record.id.slice(-6)}
        </div>
        {data.visibleColumns.map((col) => {
          const prop = data.properties.find((p) => p.id === col.propertyId);
          const value = record.properties?.[col.propertyId];
          return (
            <div key={`${record.id}-${col.propertyId}`} className="border-t border-[var(--c-line)] px-3 py-2">
              <MemoCellInput
                recordId={record.id}
                propertyId={prop?.id ?? col.propertyId}
                value={value}
                onEdit={data.onCellEdit}
              />
            </div>
          );
        })}
        <div className="border-t border-[var(--c-line)] px-3 py-2 text-right">
          {data.onDelete && (
            <button
              type="button"
              onClick={(e) => {
                e.stopPropagation();
                data.onDelete?.(record.id);
              }}
              className="rounded-full border border-[var(--c-line)] bg-white px-3 py-1 text-xs text-[var(--c-muted)]"
            >
              删除
            </button>
          )}
        </div>
      </div>
    );
  };

  return (
    <div
      className="relative rounded-lg border border-[var(--c-line)] bg-white"
      onMouseMove={handleMouseMove}
      onMouseUp={handleMouseUp}
    >
      <div
        className="grid items-center border-b border-[var(--c-line)] bg-[var(--c-surface-subtle)] px-2 py-2 text-sm font-semibold text-[var(--c-text)]"
        style={{ display: 'grid', gridTemplateColumns: gridTemplate }}
      >
        <div />
        {visibleColumns.map((col) => {
          const prop = properties.find((p) => p.id === col.propertyId);
          return (
            <div key={col.propertyId} className="relative flex items-center gap-2">
              {prop?.name || col.propertyId}
              {onReorderColumn && <ArrowsUpDownIcon className="h-4 w-4 text-[var(--c-muted)]" />}
              {onResizeColumn && (
                <span
                  role="separator"
                  tabIndex={-1}
                  onMouseDown={(e) => handleResizeStart(e, col.propertyId)}
                  className="absolute right-0 top-0 h-full w-1 cursor-col-resize select-none"
                />
              )}
            </div>
          );
        })}
        <div className="text-right text-xs text-[var(--c-muted)]">操作</div>
      </div>
      <div style={{ display: 'grid', gridTemplateColumns: gridTemplate }}>
        <List
          height={420}
          itemCount={records.length + 1}
          itemSize={44}
          width="100%"
          itemData={itemData}
          itemKey={(index) => (index >= records.length ? 'new-row' : records[index].id)}
        >
          {Row}
        </List>
      </div>
      {loading && (
        <div className="absolute inset-0 rounded-lg bg-white/70 backdrop-blur-[1px]">
          <div style={{ display: 'grid', gridTemplateColumns: gridTemplate }}>
            {Array.from({ length: 8 }).map((_, idx) => (
              <div
                key={`skeleton-${idx}`}
                className="contents text-[var(--c-muted)]"
                style={{ height: 44, borderTop: '1px solid var(--c-line)' }}
              >
                <div className="px-2 py-3">
                  <div className="h-3 w-12 animate-pulse rounded bg-[var(--c-line)]" />
                </div>
                {visibleColumns.map((col) => (
                  <div key={`${col.propertyId}-sk-${idx}`} className="px-3 py-2">
                    <div className="h-3 w-24 animate-pulse rounded bg-[var(--c-line)]" />
                  </div>
                ))}
                <div className="px-3 py-2 text-right">
                  <div className="ml-auto h-3 w-10 animate-pulse rounded bg-[var(--c-line)]" />
                </div>
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  );
};
