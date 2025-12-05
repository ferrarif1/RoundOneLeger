import { useEffect, useMemo, useRef, useState } from 'react';
import { createPortal } from 'react-dom';
import {
  ArrowsPointingInIcon,
  ArrowsPointingOutIcon,
  PencilSquareIcon,
  PlusIcon,
  TrashIcon,
  XMarkIcon
} from '@heroicons/react/24/outline';

import type { WorkspaceColumn, WorkspaceRow } from './types';

const SheetCell = ({
  value,
  onChange,
  placeholder
}: {
  value: string;
  onChange: (next: string) => void;
  placeholder?: string;
}) => {
  const textareaRef = useRef<HTMLTextAreaElement>(null);

  useEffect(() => {
    const element = textareaRef.current;
    if (!element) {
      return;
    }
    element.style.height = 'auto';
    element.style.height = `${element.scrollHeight}px`;
  }, [value]);

  return (
    <textarea
      ref={textareaRef}
      value={value}
      onChange={(event) => onChange(event.target.value)}
      rows={1}
      placeholder={placeholder}
      spellCheck={false}
      className="w-full resize-none overflow-hidden rounded-[6px] border border-[var(--line)] bg-white px-3 py-2 text-sm text-[var(--text)] outline-none focus:border-[var(--line-strong)] whitespace-pre-wrap break-words"
    />
  );
};

interface InlineTableEditorProps {
  columns: WorkspaceColumn[];
  rows: WorkspaceRow[];
  filteredRows: WorkspaceRow[];
  selectedRowIds: string[];
  highlightedRowIds?: string[];
  onToggleRowSelection: (id: string) => void;
  onToggleSelectAll: () => void;
  onUpdateColumnTitle: (columnId: string, title: string) => void;
  onRemoveColumn: (columnId: string) => void;
  onResizeColumn: (columnId: string, width: number) => void;
  onReorderColumns?: (sourceId: string, targetId: string) => void;
  onUpdateCell: (rowId: string, columnId: string, value: string) => void;
  onRemoveRow: (rowId: string) => void;
  onAddRow: () => void;
  onAddColumn: () => void;
  searchTerm: string;
  onSearchTermChange: (value: string) => void;
  onOpenBatchEdit: () => void;
  onRemoveSelected: () => void;
  onToggleHighlight?: () => void;
  onExportSelected?: () => void;
  onSaveFullscreen?: () => void;
  dirty?: boolean;
  hasSelection: boolean;
  selectAllState: 'all' | 'some' | 'none';
  isSheet: boolean;
  minColumnWidth: number;
  defaultColumnWidth: number;
}

export const InlineTableEditor = ({
  columns,
  rows,
  filteredRows,
  selectedRowIds,
  highlightedRowIds = [],
  onToggleRowSelection,
  onToggleSelectAll,
  onUpdateColumnTitle,
  onRemoveColumn,
  onResizeColumn,
  onReorderColumns,
  onUpdateCell,
  onRemoveRow,
  onAddRow,
  onAddColumn,
  searchTerm,
  onSearchTermChange,
  onOpenBatchEdit,
  onRemoveSelected,
  onToggleHighlight,
  onExportSelected,
  onSaveFullscreen,
  dirty,
  hasSelection,
  selectAllState,
  isSheet,
  minColumnWidth,
  defaultColumnWidth
}: InlineTableEditorProps) => {
  const selectAllRef = useRef<HTMLInputElement>(null);
  const [displayMode, setDisplayMode] = useState<'default' | 'wide' | 'fullscreen'>('default');
  const autosaveRef = useRef<number | null>(null);
  const [page, setPage] = useState(1);
  const pageSize = 100;

  const isWide = displayMode === 'wide';
  const isFullscreen = displayMode === 'fullscreen';

  useEffect(() => {
    if (!selectAllRef.current) {
      return;
    }
    if (selectAllState === 'some') {
      selectAllRef.current.indeterminate = true;
      selectAllRef.current.checked = false;
    } else {
      selectAllRef.current.indeterminate = false;
      selectAllRef.current.checked = selectAllState === 'all';
    }
  }, [selectAllState]);

  useEffect(() => {
    if (!isSheet) {
      setDisplayMode('default');
    }
  }, [isSheet]);

  useEffect(() => {
    if (!isFullscreen) {
      return;
    }
    const previousOverflow = document.body.style.overflow;
    document.body.style.overflow = 'hidden';
    return () => {
      document.body.style.overflow = previousOverflow;
    };
  }, [isFullscreen]);

  useEffect(() => {
    if (!isFullscreen || !onSaveFullscreen) return;
    if (autosaveRef.current) window.clearInterval(autosaveRef.current);
    autosaveRef.current = window.setInterval(() => {
      if (dirty) onSaveFullscreen();
    }, 8000);
    return () => {
      if (autosaveRef.current) window.clearInterval(autosaveRef.current);
      autosaveRef.current = null;
    };
  }, [isFullscreen, onSaveFullscreen, dirty]);

  const handleResizeStart = (event: React.MouseEvent<HTMLSpanElement>, columnId: string) => {
    event.preventDefault();
    const startX = event.clientX;
    const column = columns.find((item) => item.id === columnId);
    const startWidth = column?.width ?? defaultColumnWidth;

    const handleMove = (moveEvent: MouseEvent) => {
      const delta = moveEvent.clientX - startX;
      const width = Math.max(minColumnWidth, startWidth + delta);
      onResizeColumn(columnId, width);
    };

    const handleUp = () => {
      document.removeEventListener('mousemove', handleMove);
      document.removeEventListener('mouseup', handleUp);
    };

    document.addEventListener('mousemove', handleMove);
    document.addEventListener('mouseup', handleUp);
  };

  const renderColumnSizing = () => (
    <colgroup>
      <col style={{ width: '48px' }} />
      {columns.map((column) => (
        <col key={column.id} style={{ width: column.width ? `${column.width}px` : undefined }} />
      ))}
      <col style={{ width: '80px' }} />
    </colgroup>
  );

  const renderTableHeader = () => (
    <thead className="bg-transparent">
      <tr className="text-left text-xs font-medium text-[var(--muted)]">
        <th className="w-12 px-2 py-2">
          <input
            ref={selectAllRef}
            type="checkbox"
            onChange={onToggleSelectAll}
            className="h-4 w-4 rounded border-[var(--muted)]/40 text-[var(--accent)] focus:ring-[var(--accent)]"
          />
        </th>
        {columns.map((column) => (
          <th
            key={column.id}
            className="relative px-3 py-2"
            draggable
            onDragStart={(e) => {
              e.dataTransfer.setData('text/column-id', column.id);
              e.dataTransfer.effectAllowed = 'move';
            }}
            onDragOver={(e) => {
              if (onReorderColumns) {
                e.preventDefault();
                e.dataTransfer.dropEffect = 'move';
              }
            }}
            onDrop={(e) => {
              if (!onReorderColumns) return;
              e.preventDefault();
              const sourceId = e.dataTransfer.getData('text/column-id');
              const targetId = column.id;
              if (sourceId && targetId && sourceId !== targetId) {
                onReorderColumns(sourceId, targetId);
              }
            }}
            style={{ minWidth: `${Math.max(minColumnWidth, (column.width ?? defaultColumnWidth) - 32)}px` }}
          >
            <div className="flex items-center gap-2">
              <input
                value={column.title}
                onChange={(event) => onUpdateColumnTitle(column.id, event.target.value)}
                className="w-full rounded-[6px] border border-transparent bg-transparent px-2 py-1 text-sm font-semibold text-[var(--text)] focus:border-[var(--line-strong)] focus:outline-none"
              />
              <button
                type="button"
                onClick={() => onRemoveColumn(column.id)}
                className="rounded-full border border-[var(--line)] bg-white p-1 text-[var(--muted)]"
                aria-label="删除列"
              >
                <TrashIcon className="h-4 w-4" />
              </button>
            </div>
            <span
              role="separator"
              tabIndex={-1}
              onMouseDown={(event) => handleResizeStart(event, column.id)}
              className="absolute right-0 top-0 h-full w-2 cursor-col-resize select-none rounded-full bg-transparent"
            />
          </th>
        ))}
        <th className="px-3 py-2 text-right text-xs text-[var(--muted)]">操作</th>
      </tr>
    </thead>
  );

  const totalPages = Math.max(1, Math.ceil(filteredRows.length / pageSize));
  useEffect(() => {
    if (page > totalPages) {
      setPage(totalPages);
    }
  }, [page, totalPages]);

  const visibleRows = useMemo(() => {
    const start = (page - 1) * pageSize;
    return filteredRows.slice(start, start + pageSize);
  }, [filteredRows, page]);

  const renderTableBody = () => {
    if (!visibleRows.length) {
      return (
        <tbody>
          <tr>
            <td colSpan={columns.length + 2} className="px-6 py-10 text-center text-sm text-[var(--muted)]">
              {rows.length
                ? '未找到匹配的记录，尝试调整搜索条件或清空筛选。'
                : '暂无记录，使用“新增行”或导入功能开始填写内容。'}
            </td>
          </tr>
          <tr className="border-t border-[var(--line)]">
            <td className="px-2 py-3" />
            <td colSpan={columns.length} className="px-3 py-3 text-sm text-[var(--accent)]">
              <button type="button" onClick={onAddRow} className="flex items-center gap-2">
                <PlusIcon className="h-4 w-4" />
                新增行
              </button>
            </td>
            <td className="px-3 py-3 text-right">
              <button type="button" onClick={onAddColumn} className="text-[var(--accent)]">
                + 新增列
              </button>
            </td>
          </tr>
        </tbody>
      );
    }

    return (
      <tbody>
        {visibleRows.map((row) => {
          const rowSelected = selectedRowIds.includes(row.id);
          return (
            <tr
              key={row.id}
              className={`align-top border-t border-[var(--line)] ${rowSelected ? 'bg-[#f4f6f8]' : 'bg-white'} ${
                highlightedRowIds.includes(row.id) ? 'font-semibold' : ''
              }`}
            >
              <td className="px-2 py-3 align-top">
                <input
                  type="checkbox"
                  checked={rowSelected}
                  onChange={() => onToggleRowSelection(row.id)}
                  className="h-4 w-4 rounded border-[var(--muted)]/40 text-[var(--accent)] focus:ring-[var(--accent)]"
                />
              </td>
              {columns.map((column) => (
                <td key={column.id} className="px-3 py-2 align-top">
                  <SheetCell
                    value={row.cells[column.id] ?? ''}
                    onChange={(next) => onUpdateCell(row.id, column.id, next)}
                    placeholder={column.title}
                  />
                </td>
              ))}
              <td className="px-3 py-2 text-right align-top">
                <button
                  type="button"
                  onClick={() => onRemoveRow(row.id)}
                  className="rounded-full border border-[var(--line)] bg-white p-2 text-[var(--muted)]"
                  aria-label="删除行"
                >
                  <TrashIcon className="h-4 w-4" />
                </button>
              </td>
            </tr>
          );
        })}
        <tr className="border-t border-[var(--line)] bg-white">
          <td className="px-2 py-3" />
          <td colSpan={columns.length} className="px-3 py-3 text-sm text-[var(--muted)]">
            <button type="button" onClick={onAddRow} className="flex items-center gap-2 text-[var(--accent)]">
              <PlusIcon className="h-4 w-4" />
              新增行
            </button>
          </td>
          <td className="px-3 py-3 text-right">
            <button type="button" onClick={onAddColumn} className="text-[var(--accent)]">
              + 新增列
            </button>
          </td>
        </tr>
      </tbody>
    );
  };

  const handleToggleWide = () => {
    setDisplayMode((current) => (current === 'wide' ? 'default' : 'wide'));
  };

  const handleToggleFullscreen = () => {
    setDisplayMode((current) => (current === 'fullscreen' ? 'default' : 'fullscreen'));
  };

  const baseButtonClass =
    'flex items-center gap-1 rounded-full border border-[var(--line)] bg-white px-3 py-2 text-xs text-[var(--muted)] disabled:cursor-not-allowed disabled:opacity-60';
  const dangerButtonClass =
    'flex items-center gap-1 rounded-full border border-red-200 bg-red-50 px-3 py-2 text-xs text-red-500 disabled:cursor-not-allowed disabled:opacity-60';

  const renderToolbar = (fullscreen: boolean) => {
    const containerClass = fullscreen
      ? 'sticky top-0 z-30 flex flex-wrap items-center justify-between gap-3 border-b border-[var(--line)] bg-white px-2 py-4'
      : 'flex flex-wrap items-center justify-between gap-3';
    const fullscreenLabel = isFullscreen ? '退出全屏' : '全屏编辑';
    const fullscreenIcon = isFullscreen ? (
      <ArrowsPointingInIcon className="h-4 w-4" />
    ) : (
      <ArrowsPointingOutIcon className="h-4 w-4" />
    );
    const wideLabel = isWide ? '退出宽屏' : '宽屏视图';
    const wideIcon = isWide ? (
      <ArrowsPointingInIcon className="h-4 w-4" />
    ) : (
      <ArrowsPointingOutIcon className="h-4 w-4" />
    );
    const pager = (
      <div className="flex items-center gap-2">
        <span className="text-[var(--muted)]">第 {page}/{totalPages} 页</span>
        <button
          type="button"
          className={baseButtonClass}
          onClick={() => setPage((p) => Math.max(1, p - 1))}
          disabled={page <= 1}
        >
          上一页
        </button>
        <button
          type="button"
          className={baseButtonClass}
          onClick={() => setPage((p) => Math.min(totalPages, p + 1))}
          disabled={page >= totalPages}
        >
          下一页
        </button>
      </div>
    );

    return (
      <div className={containerClass}>
        <div className="flex flex-wrap items-center gap-3 text-[var(--text)]">
          <h3 className="text-base font-semibold">表格数据</h3>
          <button type="button" className={baseButtonClass} onClick={onAddColumn}>
            <PlusIcon className="h-4 w-4" />
            新增列
          </button>
          <button type="button" className={baseButtonClass} onClick={onAddRow}>
            <PlusIcon className="h-4 w-4" />
            新增行
          </button>
        </div>
        <div className="flex flex-wrap items-center gap-2 text-xs text-[var(--muted)]">
          {pager}
          <div className="relative">
            <input
              value={searchTerm}
              onChange={(event) => onSearchTermChange(event.target.value)}
              placeholder="搜索关键字"
              className="w-44 rounded-full border border-[var(--line)] bg-white px-3 py-2 text-sm text-[var(--text)] focus:border-[var(--line-strong)] focus:outline-none"
            />
            {searchTerm && (
              <button
                type="button"
                onClick={() => onSearchTermChange('')}
                className="absolute inset-y-0 right-2 flex items-center text-[var(--muted)]"
                aria-label="清空搜索"
              >
                <XMarkIcon className="h-4 w-4" />
              </button>
            )}
          </div>
          <button
            type="button"
            onClick={onOpenBatchEdit}
            className={baseButtonClass}
            disabled={!hasSelection || !columns.length}
          >
            <PencilSquareIcon className="h-4 w-4" />
            批量编辑
          </button>
          <button
            type="button"
            onClick={onToggleHighlight}
            className={baseButtonClass}
            disabled={!hasSelection}
          >
            高亮/加粗
          </button>
          <button
            type="button"
            onClick={onExportSelected}
            className={baseButtonClass}
            disabled={!hasSelection}
          >
            导出选中 Excel
          </button>
          <button
            type="button"
            onClick={onRemoveSelected}
            className={dangerButtonClass}
            disabled={!hasSelection}
          >
            <TrashIcon className="h-4 w-4" />
            删除选中
          </button>
          <button
            type="button"
            onClick={handleToggleWide}
            className={baseButtonClass}
            disabled={isFullscreen}
          >
            {wideIcon}
            {wideLabel}
          </button>
          <button
            type="button"
            onClick={handleToggleFullscreen}
            className={`${baseButtonClass} ${
              isFullscreen ? 'border-[var(--accent)] bg-[var(--accent)]/10 text-[var(--accent)] hover:bg-[var(--accent)]/20' : ''
            }`}
          >
            {fullscreenIcon}
            {fullscreenLabel}
          </button>
          {isFullscreen && onSaveFullscreen && (
            <button type="button" onClick={onSaveFullscreen} className={baseButtonClass}>
              保存
            </button>
          )}
        </div>
      </div>
    );
  };

  const renderSelectionBanner = (fullscreen: boolean) => {
    if (!hasSelection) {
      return null;
    }
    return (
      <div
        className={`rounded-lg bg-[#f4f6f8] px-3 py-2 text-xs text-[var(--accent)] ${fullscreen ? 'mx-1' : ''}`}
      >
        已选中 {selectedRowIds.length} 行，支持批量编辑或删除。
      </div>
    );
  };

  const renderTableShell = (fullscreen: boolean) => {
    const wrapperClass = [
      'relative rounded-lg',
      fullscreen ? 'flex-1 overflow-auto bg-white' : 'overflow-auto bg-white',
      !fullscreen && isWide ? 'max-w-none' : ''
    ]
      .filter(Boolean)
      .join(' ');
    const minWidthClass = fullscreen || isFullscreen || isWide ? 'min-w-[1200px]' : 'min-w-[800px]';

    return (
      <div className={wrapperClass}>
        <table
          className={`${minWidthClass} w-full table-fixed text-sm leading-snug text-[var(--text)]`}
          style={{ tableLayout: 'fixed' }}
        >
          {renderColumnSizing()}
          {renderTableHeader()}
          {renderTableBody()}
        </table>
      </div>
    );
  };

  if (!isSheet) {
    return null;
  }

  return (
    <>
      <section className="space-y-4">
        {renderToolbar(false)}
        {renderSelectionBanner(false)}
        {isFullscreen ? (
          <div className="rounded-lg border border-dashed border-[var(--line)] bg-white p-6 text-center text-sm text-[var(--muted)]">
            正在全屏编辑表格，点击“退出全屏”返回当前视图。
          </div>
        ) : (
          renderTableShell(false)
        )}
      </section>
      {isFullscreen &&
        createPortal(
          <div className="fixed inset-0 z-[120] flex flex-col bg-[rgba(17,24,39,0.12)]">
            <div className="flex h-full flex-col bg-white">
              {renderToolbar(true)}
              <div className="flex-1 overflow-hidden px-6 pb-6">
                <div className="flex h-full flex-col gap-3">
                  {renderSelectionBanner(true)}
                  {renderTableShell(true)}
                </div>
              </div>
            </div>
          </div>,
          document.body
        )}
    </>
  );
};
