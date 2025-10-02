import {
  ChangeEvent,
  Fragment,
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState
} from 'react';
import {
  DocumentTextIcon,
  FolderIcon,
  TableCellsIcon,
  XMarkIcon
} from '@heroicons/react/24/outline';
import type { AxiosError } from 'axios';

import '@/styles/eidos-ledger.css';
import api from '../api/client';
import { LedgerLayout } from '../components/ledger/LedgerLayout';
import { LedgerListCard } from '../components/ledger/LedgerListCard';
import { LedgerEditorCard } from '../components/ledger/LedgerEditorCard';
import { ToolbarActions } from '../components/ledger/ToolbarActions';
import { InlineTableEditor } from '../components/ledger/InlineTableEditor';
import { DocumentEditor } from '../components/ledger/DocumentEditor';
import { ToastStack } from '../components/ledger/ToastStack';
import type {
  Workspace,
  WorkspaceColumn,
  WorkspaceKind,
  WorkspaceNode,
  WorkspaceRow
} from '../components/ledger/types';

const DEFAULT_TITLE = '未命名台账';
const MIN_COLUMN_WIDTH = 140;
const DEFAULT_COLUMN_WIDTH = 220;

const normalizeColumnWidth = (width?: number) =>
  Math.max(MIN_COLUMN_WIDTH, width ?? DEFAULT_COLUMN_WIDTH);

const collectColumnIds = (a: WorkspaceColumn[], b: WorkspaceColumn[]) => {
  const ids = new Set<string>();
  a.forEach((column) => ids.add(column.id));
  b.forEach((column) => ids.add(column.id));
  return Array.from(ids);
};

const areColumnsEqual = (a: WorkspaceColumn[], b: WorkspaceColumn[]) => {
  if (a.length !== b.length) {
    return false;
  }
  for (let index = 0; index < a.length; index += 1) {
    const original = a[index];
    const current = b[index];
    if (!current) {
      return false;
    }
    if (original.id !== current.id) {
      return false;
    }
    if ((original.title || '') !== (current.title || '')) {
      return false;
    }
    if (normalizeColumnWidth(original.width) !== normalizeColumnWidth(current.width)) {
      return false;
    }
  }
  return true;
};

const areRowsEqual = (
  a: WorkspaceRow[],
  b: WorkspaceRow[],
  columnIds: string[]
) => {
  if (a.length !== b.length) {
    return false;
  }
  for (let index = 0; index < a.length; index += 1) {
    const original = a[index];
    const current = b[index];
    if (!current || original.id !== current.id) {
      return false;
    }
    for (const columnId of columnIds) {
      const originalValue = original.cells?.[columnId] ?? '';
      const currentValue = current.cells?.[columnId] ?? '';
      if (originalValue !== currentValue) {
        return false;
      }
    }
  }
  return true;
};

const CREATION_OPTIONS = [
  {
    kind: 'sheet' as WorkspaceKind,
    label: '新建表格台账',
    description: '用于结构化字段的协同维护',
    icon: TableCellsIcon
  },
  {
    kind: 'document' as WorkspaceKind,
    label: '新建在线文档',
    description: '记录会议纪要、方案与流程说明',
    icon: DocumentTextIcon
  },
  {
    kind: 'folder' as WorkspaceKind,
    label: '新建文件夹',
    description: '整理不同台账与文档',
    icon: FolderIcon
  }
];

const generateId = () => {
  if (typeof crypto !== 'undefined' && typeof crypto.randomUUID === 'function') {
    return crypto.randomUUID();
  }
  return `id-${Date.now().toString(16)}-${Math.random().toString(16).slice(2)}`;
};

const fillRowCells = (row: WorkspaceRow, columns: WorkspaceColumn[]): WorkspaceRow => {
  const cells: Record<string, string> = {};
  columns.forEach((column) => {
    cells[column.id] = row.cells?.[column.id] ?? '';
  });
  return { id: row.id, cells };
};

const createEmptyRow = (columns: WorkspaceColumn[]): WorkspaceRow => ({
  id: generateId(),
  cells: columns.reduce<Record<string, string>>((acc, column) => {
    acc[column.id] = '';
    return acc;
  }, {})
});

const flattenWorkspaces = (nodes: WorkspaceNode[]): WorkspaceNode[] => {
  const result: WorkspaceNode[] = [];
  const walk = (list: WorkspaceNode[]) => {
    list.forEach((node) => {
      result.push(node);
      if (node.children?.length) {
        walk(node.children);
      }
    });
  };
  walk(nodes);
  return result;
};

const normalizeKind = (value?: string): WorkspaceKind => {
  if (!value) {
    return 'sheet';
  }
  const lower = value.toLowerCase();
  if (lower === 'document') {
    return 'document';
  }
  if (lower === 'folder') {
    return 'folder';
  }
  return 'sheet';
};

const formatTimestamp = (value?: string) => {
  if (!value) {
    return '';
  }
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return '';
  }
  return new Intl.DateTimeFormat('zh-CN', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit'
  }).format(date);
};

const PasteModal = ({
  open,
  onClose,
  onSubmit
}: {
  open: boolean;
  onClose: () => void;
  onSubmit: (text: string, delimiter: string, hasHeader: boolean) => Promise<void>;
}) => {
  const [text, setText] = useState('');
  const [delimiter, setDelimiter] = useState<'tab' | 'comma' | 'semicolon' | 'space'>('tab');
  const [hasHeader, setHasHeader] = useState(true);
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (!open) {
      setText('');
      setDelimiter('tab');
      setHasHeader(true);
      setSubmitting(false);
      setError(null);
    }
  }, [open]);

  if (!open) {
    return null;
  }

  const handleSubmit = async () => {
    if (!text.trim()) {
      setError('请粘贴需要导入的数据。');
      return;
    }
    setSubmitting(true);
    setError(null);
    try {
      await onSubmit(text, delimiter, hasHeader);
      onClose();
    } catch (err) {
      const axiosError = err as AxiosError<{ error?: string }>;
      setError(axiosError.response?.data?.error || axiosError.message || '导入失败，请稍后再试。');
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <div className="fixed inset-0 z-40 flex items-center justify-center bg-black/40 px-4">
      <div className="w-full max-w-2xl rounded-[var(--radius-lg)] bg-white p-6 shadow-[var(--shadow-soft)]">
        <div className="mb-4 flex items-center justify-between">
          <h3 className="text-lg font-semibold text-[var(--text)]">批量粘贴导入</h3>
          <button
            type="button"
            onClick={onClose}
            className="rounded-full border border-black/5 p-2 text-[var(--muted)] hover:text-[var(--text)]"
            aria-label="关闭"
          >
            <XMarkIcon className="h-5 w-5" />
          </button>
        </div>
        <div className="space-y-3 text-sm text-[var(--text)]">
          <textarea
            value={text}
            onChange={(event) => setText(event.target.value)}
            rows={10}
            className="w-full rounded-[var(--radius-md)] border border-black/5 bg-white/90 p-3 text-sm text-[var(--text)] focus:border-[var(--accent)] focus:outline-none focus:ring-2 focus:ring-[var(--accent)]/20"
            placeholder="将 Excel 或其他表格数据复制后粘贴在此处，每列使用制表符分隔"
          />
          <div className="flex flex-wrap items-center gap-4 text-xs text-[var(--muted)]">
            <label className="flex items-center gap-2">
              <span>分隔符</span>
              <select
                value={delimiter}
                onChange={(event) => setDelimiter(event.target.value as typeof delimiter)}
                className="rounded-full border border-black/10 bg-white/90 px-2 py-1 text-[var(--text)] focus:border-[var(--accent)] focus:outline-none"
              >
                <option value="tab">制表符</option>
                <option value="comma">逗号</option>
                <option value="semicolon">分号</option>
                <option value="space">空格</option>
              </select>
            </label>
            <label className="flex items-center gap-2">
              <input
                type="checkbox"
                checked={hasHeader}
                onChange={(event) => setHasHeader(event.target.checked)}
                className="rounded border border-black/10 text-[var(--accent)] focus:ring-[var(--accent)]"
              />
              首行是表头
            </label>
          </div>
          {error && (
            <div className="rounded-full bg-red-50 px-3 py-2 text-sm text-red-600">{error}</div>
          )}
          <div className="flex justify-end gap-3 text-sm">
            <button
              type="button"
              onClick={onClose}
              className="rounded-full border border-black/10 px-4 py-2 text-[var(--muted)] hover:text-[var(--text)]"
            >
              取消
            </button>
            <button
              type="button"
              onClick={handleSubmit}
              disabled={submitting}
              className="rounded-full bg-[var(--accent)] px-4 py-2 font-medium text-white shadow-[var(--shadow-sm)] hover:bg-[var(--accent-2)] disabled:cursor-not-allowed"
            >
              {submitting ? '正在导入…' : '导入数据'}
            </button>
          </div>
        </div>
      </div>
    </div>
  );
};

const BatchEditModal = ({
  open,
  onClose,
  columns,
  onApply,
  selectedCount
}: {
  open: boolean;
  onClose: () => void;
  columns: WorkspaceColumn[];
  onApply: (columnId: string, value: string) => Promise<void> | void;
  selectedCount: number;
}) => {
  const [columnId, setColumnId] = useState('');
  const [value, setValue] = useState('');
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (!open) {
      return;
    }
    const firstColumn = columns[0]?.id ?? '';
    setColumnId(firstColumn);
    setValue('');
    setError(null);
    setSubmitting(false);
  }, [open, columns]);

  if (!open) {
    return null;
  }

  const handleSubmit = async () => {
    if (!columnId) {
      setError('请选择需要批量更新的字段。');
      return;
    }
    setSubmitting(true);
    setError(null);
    try {
      await onApply(columnId, value);
      onClose();
    } catch (err) {
      const axiosError = err as AxiosError<{ error?: string }>;
      setError(axiosError.response?.data?.error || axiosError.message || '批量更新失败，请稍后重试。');
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <div className="fixed inset-0 z-40 flex items-center justify-center bg-black/40 px-4">
      <div className="w-full max-w-lg rounded-[var(--radius-lg)] bg-white p-6 shadow-[var(--shadow-soft)]">
        <div className="mb-4 flex items-center justify-between">
          <div>
            <h3 className="text-lg font-semibold text-[var(--text)]">批量编辑</h3>
            <p className="mt-1 text-xs text-[var(--muted)]">将同步更新选中的 {selectedCount} 行内容。</p>
          </div>
          <button
            type="button"
            onClick={onClose}
            className="rounded-full border border-black/5 p-2 text-[var(--muted)] hover:text-[var(--text)]"
            aria-label="关闭"
          >
            <XMarkIcon className="h-5 w-5" />
          </button>
        </div>
        <div className="space-y-4 text-sm text-[var(--text)]">
          <label className="flex flex-col gap-2">
            <span className="text-xs text-[var(--muted)]">选择字段</span>
            <select
              value={columnId}
              onChange={(event) => setColumnId(event.target.value)}
              className="rounded-full border border-black/10 bg-white/90 px-3 py-2 text-sm text-[var(--text)] focus:border-[var(--accent)] focus:outline-none focus:ring-2 focus:ring-[var(--accent)]/20"
            >
              {columns.map((column) => (
                <option key={column.id} value={column.id}>
                  {column.title || '未命名字段'}
                </option>
              ))}
            </select>
          </label>
          <label className="flex flex-col gap-2">
            <span className="text-xs text-[var(--muted)]">新的内容</span>
            <textarea
              value={value}
              onChange={(event) => setValue(event.target.value)}
              rows={4}
              className="w-full resize-none rounded-[var(--radius-md)] border border-black/10 bg-white/90 px-3 py-2 text-sm text-[var(--text)] focus:border-[var(--accent)] focus:outline-none focus:ring-2 focus:ring-[var(--accent)]/20"
            />
          </label>
          {error && <div className="rounded-full bg-red-50 px-3 py-2 text-sm text-red-600">{error}</div>}
        </div>
        <div className="mt-6 flex justify-end gap-3 text-sm">
          <button
            type="button"
            onClick={onClose}
            className="rounded-full border border-black/10 px-4 py-2 text-[var(--muted)] hover:text-[var(--text)]"
          >
            取消
          </button>
          <button
            type="button"
            onClick={handleSubmit}
            disabled={submitting || !columnId}
            className="rounded-full bg-[var(--accent)] px-4 py-2 font-medium text-white shadow-[var(--shadow-sm)] hover:bg-[var(--accent-2)] disabled:cursor-not-allowed"
          >
            {submitting ? '正在应用…' : '应用更改'}
          </button>
        </div>
      </div>
    </div>
  );
};

const Assets = () => {
  const [workspaceTree, setWorkspaceTree] = useState<WorkspaceNode[]>([]);
  const [selectedId, setSelectedId] = useState<string | null>(null);
  const [currentWorkspace, setCurrentWorkspace] = useState<Workspace | null>(null);
  const [columns, setColumns] = useState<WorkspaceColumn[]>([]);
  const [rows, setRows] = useState<WorkspaceRow[]>([]);
  const [name, setName] = useState('');
  const [documentContent, setDocumentContent] = useState('');
  const [loading, setLoading] = useState(false);
  const [status, setStatus] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [showPasteModal, setShowPasteModal] = useState(false);
  const [showBatchEditModal, setShowBatchEditModal] = useState(false);
  const [tableSearch, setTableSearch] = useState('');
  const [listQuery, setListQuery] = useState('');
  const [selectedRowIds, setSelectedRowIds] = useState<string[]>([]);
  const [saving, setSaving] = useState(false);
  const excelInputRef = useRef<HTMLInputElement>(null);

  const allNodes = useMemo(() => flattenWorkspaces(workspaceTree), [workspaceTree]);
  const workspaceMap = useMemo(() => {
    const map = new Map<string, WorkspaceNode>();
    allNodes.forEach((node) => {
      map.set(node.id, node);
    });
    return map;
  }, [allNodes]);

  const selectedNode = selectedId ? workspaceMap.get(selectedId) : null;
  const selectedKind: WorkspaceKind = currentWorkspace?.kind ?? normalizeKind(selectedNode?.kind);
  const isSheet = selectedKind === 'sheet';
  const isDocument = selectedKind === 'document';
  const isFolder = selectedKind === 'folder';
  const showDocumentEditor = isDocument;
  const folderChildren = selectedNode?.children ?? [];

  const normalizedOriginalColumns = useMemo(() => {
    if (!currentWorkspace || currentWorkspace.kind !== 'sheet') {
      return [] as WorkspaceColumn[];
    }
    const source = currentWorkspace.columns ?? [];
    const base = source.length
      ? source
      : [
          { id: generateId(), title: '字段 1', width: DEFAULT_COLUMN_WIDTH }
        ];
    return base.map((column, index) => ({
      ...column,
      id: column.id || generateId(),
      title: column.title || `字段 ${index + 1}`,
      width: normalizeColumnWidth(column.width)
    }));
  }, [currentWorkspace]);

  const normalizedOriginalRows = useMemo(() => {
    if (!currentWorkspace || currentWorkspace.kind !== 'sheet') {
      return [] as WorkspaceRow[];
    }
    const source = currentWorkspace.rows ?? [];
    return source.length ? source.map((row) => fillRowCells(row, normalizedOriginalColumns)) : [];
  }, [currentWorkspace, normalizedOriginalColumns]);

  const comparisonColumnIds = useMemo(
    () => (selectedKind === 'sheet' ? collectColumnIds(normalizedOriginalColumns, columns) : []),
    [columns, normalizedOriginalColumns, selectedKind]
  );

  const hasUnsavedChanges = useMemo(() => {
    if (!currentWorkspace) {
      return false;
    }
    const trimmedName = name.trim() || DEFAULT_TITLE;
    if (trimmedName !== (currentWorkspace.name || DEFAULT_TITLE)) {
      return true;
    }
    if (currentWorkspace.kind === 'sheet') {
      if (!areColumnsEqual(normalizedOriginalColumns, columns)) {
        return true;
      }
      if (!areRowsEqual(normalizedOriginalRows, rows, comparisonColumnIds)) {
        return true;
      }
      if ((documentContent || '') !== (currentWorkspace.document || '')) {
        return true;
      }
    }
    if (currentWorkspace.kind === 'document') {
      if ((documentContent || '') !== (currentWorkspace.document || '')) {
        return true;
      }
    }
    return false;
  }, [
    columns,
    comparisonColumnIds,
    currentWorkspace,
    documentContent,
    name,
    normalizedOriginalColumns,
    normalizedOriginalRows,
    rows
  ]);

  const filteredRows = useMemo(() => {
    if (!isSheet) {
      return rows;
    }
    const keyword = tableSearch.trim().toLowerCase();
    if (!keyword) {
      return rows;
    }
    return rows.filter((row) =>
      columns.some((column) => (row.cells[column.id] ?? '').toLowerCase().includes(keyword))
    );
  }, [rows, columns, tableSearch, isSheet]);

  const filteredRowIds = useMemo(() => filteredRows.map((row) => row.id), [filteredRows]);
  const selectedFilteredCount = useMemo(
    () => filteredRows.filter((row) => selectedRowIds.includes(row.id)).length,
    [filteredRows, selectedRowIds]
  );
  const hasSelection = selectedRowIds.length > 0;
  const selectAllState = useMemo<'all' | 'some' | 'none'>(() => {
    if (!filteredRows.length || selectedFilteredCount === 0) {
      return 'none';
    }
    if (selectedFilteredCount === filteredRows.length) {
      return 'all';
    }
    return 'some';
  }, [filteredRows.length, selectedFilteredCount]);

  useEffect(() => {
    setSelectedRowIds((prev) => {
      if (!prev.length) {
        return prev;
      }
      const allowed = new Set(rows.map((row) => row.id));
      const next = prev.filter((id) => allowed.has(id));
      return next.length === prev.length ? prev : next;
    });
  }, [rows]);

  const refreshList = useCallback(
    async (focusId?: string | null) => {
      try {
        const { data } = await api.get<{ items: WorkspaceNode[] }>('/api/v1/workspaces');
        const nodes = Array.isArray(data?.items) ? data.items : [];
        setWorkspaceTree(nodes);
        const flattened = flattenWorkspaces(nodes);
        if (!flattened.length) {
          setSelectedId(null);
          setCurrentWorkspace(null);
          setColumns([]);
          setRows([]);
          setDocumentContent('');
          setName('');
          return;
        }
        const candidate = focusId ?? selectedId;
        const nextId = candidate && flattened.some((item) => item.id === candidate) ? candidate : flattened[0].id;
        setSelectedId(nextId);
      } catch (err) {
        console.error('加载台账列表失败', err);
        setError('无法加载台账列表，请稍后再试。');
      }
    },
    [selectedId]
  );

  const loadWorkspace = useCallback(
    async (id: string) => {
      try {
        setLoading(true);
        setError(null);
        const { data } = await api.get<{ workspace: Workspace }>(`/api/v1/workspaces/${id}`);
        if (!data?.workspace) {
          throw new Error('未获取到台账数据');
        }
        const workspace = data.workspace;
        const kind = normalizeKind(workspace.kind);
        const normalizedColumns =
          kind === 'sheet'
            ? (workspace.columns.length
                ? workspace.columns
                : [
                    {
                      id: generateId(),
                      title: '字段 1',
                      width: DEFAULT_COLUMN_WIDTH
                    }
                  ]
              ).map((column, index) => ({
                ...column,
                id: column.id || generateId(),
                title: column.title || `字段 ${index + 1}`,
                width: normalizeColumnWidth(column.width)
              }))
            : [];
        const normalizedRows =
          kind === 'sheet' && workspace.rows.length
            ? workspace.rows.map((row) => fillRowCells(row, normalizedColumns))
            : [];
        const normalizedDocument = kind === 'folder' ? '' : workspace.document || '';

        setCurrentWorkspace({ ...workspace, kind, parentId: workspace.parentId ?? '' });
        setColumns(normalizedColumns);
        setRows(normalizedRows);
        setName(workspace.name || DEFAULT_TITLE);
        setDocumentContent(normalizedDocument);
        setStatus(null);
        setError(null);
      } catch (err) {
        console.error('加载台账失败', err);
        const axiosError = err as AxiosError<{ error?: string }>;
        setError(axiosError.response?.data?.error || axiosError.message || '无法加载台账内容。');
      } finally {
        setLoading(false);
      }
    },
    []
  );

  useEffect(() => {
    refreshList().catch((err) => console.error(err));
  }, [refreshList]);

  useEffect(() => {
    if (selectedId) {
      loadWorkspace(selectedId).catch((err) => console.error(err));
    }
  }, [selectedId, loadWorkspace]);

  useEffect(() => {
    setTableSearch('');
    setSelectedRowIds([]);
  }, [currentWorkspace?.id]);

  const updateColumnTitle = useCallback(
    (id: string, title: string) => {
      if (!isSheet) return;
      setColumns((prev) => prev.map((column) => (column.id === id ? { ...column, title } : column)));
    },
    [isSheet]
  );

  const removeColumn = useCallback(
    (id: string) => {
      if (!isSheet) return;
      setColumns((prev) => prev.filter((column) => column.id !== id));
      setRows((prev) =>
        prev.map((row) => {
          const nextCells = { ...row.cells };
          delete nextCells[id];
          return { ...row, cells: nextCells };
        })
      );
    },
    [isSheet]
  );

  const addColumn = useCallback(() => {
    if (!isSheet) return;
    const title = window.prompt('请输入新列的标题', `列 ${columns.length + 1}`);
    if (!title) {
      return;
    }
    const id = generateId();
    const column: WorkspaceColumn = { id, title, width: DEFAULT_COLUMN_WIDTH };
    setColumns((prev) => [...prev, column]);
    setRows((prev) => prev.map((row) => ({ ...row, cells: { ...row.cells, [id]: '' } })));
  }, [columns.length, isSheet]);

  const addRow = useCallback(() => {
    if (!isSheet) return;
    setRows((prev) => [...prev, createEmptyRow(columns)]);
  }, [columns, isSheet]);

  const removeRow = useCallback(
    (id: string) => {
      if (!isSheet) return;
      setRows((prev) => prev.filter((row) => row.id !== id));
    },
    [isSheet]
  );

  const updateCell = useCallback(
    (rowId: string, columnId: string, value: string) => {
      if (!isSheet) return;
      setRows((prev) =>
        prev.map((row) => (row.id === rowId ? { ...row, cells: { ...row.cells, [columnId]: value } } : row))
      );
    },
    [isSheet]
  );

  const handleColumnResize = useCallback((columnId: string, width: number) => {
    if (!isSheet) return;
    setColumns((prev) => prev.map((column) => (column.id === columnId ? { ...column, width } : column)));
  }, [isSheet]);

  const toggleRowSelection = useCallback(
    (rowId: string) => {
      setSelectedRowIds((prev) => (prev.includes(rowId) ? prev.filter((id) => id !== rowId) : [...prev, rowId]));
    },
    []
  );

  const toggleSelectAll = useCallback(() => {
    setSelectedRowIds((prev) => {
      const current = new Set(prev);
      const allSelected = filteredRowIds.every((id) => current.has(id));
      if (allSelected) {
        filteredRowIds.forEach((id) => current.delete(id));
      } else {
        filteredRowIds.forEach((id) => current.add(id));
      }
      return Array.from(current);
    });
  }, [filteredRowIds]);

  const handleBatchEditApply = useCallback(
    async (columnId: string, value: string) => {
      setRows((prev) =>
        prev.map((row) =>
          selectedRowIds.includes(row.id) ? { ...row, cells: { ...row.cells, [columnId]: value } } : row
        )
      );
    },
    [selectedRowIds]
  );

  const handleRemoveSelectedRows = useCallback(() => {
    if (!selectedRowIds.length) {
      return;
    }
    if (!window.confirm(`确认删除选中的 ${selectedRowIds.length} 行记录？`)) {
      return;
    }
    setRows((prev) => prev.filter((row) => !selectedRowIds.includes(row.id)));
    setSelectedRowIds([]);
  }, [selectedRowIds]);

  const resolveParentForCreation = useCallback(() => {
    if (!selectedNode) {
      return '';
    }
    if (selectedNode.kind === 'folder') {
      return selectedNode.id;
    }
    return selectedNode.parentId ?? '';
  }, [selectedNode]);

  const handleCreateWorkspace = useCallback(
    async (kind: WorkspaceKind) => {
      const labels: Record<WorkspaceKind, { title: string; prompt: string }> = {
        sheet: { title: '新建表格台账', prompt: '请输入新台账名称' },
        document: { title: '新建在线文档', prompt: '请输入文档名称' },
        folder: { title: '新建文件夹', prompt: '请输入文件夹名称' }
      };
      const defaultName = kind === 'folder' ? '未命名文件夹' : kind === 'document' ? '未命名文档' : DEFAULT_TITLE;
      const userInput = window.prompt(labels[kind].prompt, defaultName);
      if (userInput === null) {
        return;
      }
      const title = userInput.trim() || defaultName;
      const parentId = resolveParentForCreation();
      const payload: Record<string, unknown> = {
        name: title,
        kind,
        parentId
      };
      if (kind === 'sheet') {
        payload.columns = [
          { id: generateId(), title: '字段 1', width: DEFAULT_COLUMN_WIDTH },
          { id: generateId(), title: '字段 2', width: DEFAULT_COLUMN_WIDTH }
        ];
        payload.rows = [];
        payload.document = '';
      }
      try {
        const { data } = await api.post<{ workspace: Workspace }>('/api/v1/workspaces', payload);
        setStatus(`${labels[kind].title}已创建。`);
        await refreshList(data.workspace?.id ?? null);
      } catch (err) {
        console.error('创建台账失败', err);
        const axiosError = err as AxiosError<{ error?: string }>;
        setError(axiosError.response?.data?.error || axiosError.message || '创建失败，请稍后再试。');
      }
    },
    [refreshList, resolveParentForCreation]
  );

  const handleSave = useCallback(async () => {
    if (!currentWorkspace) {
      return;
    }
    setStatus(null);
    setError(null);
    const trimmedName = name.trim() || DEFAULT_TITLE;
    const payload: Record<string, unknown> = { name: trimmedName };
    if (isSheet) {
      payload.document = documentContent;
      payload.columns = columns.map((column, index) => ({
        id: column.id || generateId(),
        title: column.title || `字段 ${index + 1}`,
        width: column.width ?? DEFAULT_COLUMN_WIDTH
      }));
      payload.rows = rows.map((row) => ({ id: row.id, cells: row.cells }));
    } else if (isDocument) {
      payload.document = documentContent;
    }
    setSaving(true);
    try {
      await api.put(`/api/v1/workspaces/${currentWorkspace.id}`, payload);
      setStatus('已保存所有更改。');
      await refreshList(currentWorkspace.id);
    } catch (err) {
      console.error('保存失败', err);
      const axiosError = err as AxiosError<{ error?: string }>;
      setError(axiosError.response?.data?.error || axiosError.message || '保存失败，请稍后再试。');
    } finally {
      setSaving(false);
    }
  }, [columns, currentWorkspace, documentContent, isDocument, isSheet, name, refreshList, rows]);

  const handleDeleteWorkspace = useCallback(async () => {
    if (!currentWorkspace) {
      return;
    }
    if (!window.confirm('确认删除当前台账？该操作无法撤销。')) {
      return;
    }
    try {
      await api.delete(`/api/v1/workspaces/${currentWorkspace.id}`);
      setStatus('已删除当前台账。');
      await refreshList();
    } catch (err) {
      console.error('删除台账失败', err);
      const axiosError = err as AxiosError<{ error?: string }>;
      setError(axiosError.response?.data?.error || axiosError.message || '删除失败，请稍后再试。');
    }
  }, [currentWorkspace, refreshList]);

  const handleImportExcel = useCallback(
    async (event: ChangeEvent<HTMLInputElement>) => {
      const file = event.target.files?.[0];
      event.target.value = '';
      if (!file || !currentWorkspace || !isSheet) {
        return;
      }
      try {
        const formData = new FormData();
        formData.append('file', file);
        const { data } = await api.post<{ workspace: Workspace }>(
          `/api/v1/workspaces/${currentWorkspace.id}/import/excel`,
          formData,
          {
            headers: { 'Content-Type': 'multipart/form-data' }
          }
        );
        if (data?.workspace) {
          await loadWorkspace(data.workspace.id);
          setStatus('Excel 数据已导入。');
        }
      } catch (err) {
        console.error('导入 Excel 失败', err);
        const axiosError = err as AxiosError<{ error?: string }>;
        setError(axiosError.response?.data?.error || axiosError.message || '导入失败，请确认文件格式。');
      }
    },
    [currentWorkspace, isSheet, loadWorkspace]
  );

  const handleImportText = useCallback(
    async (text: string, delimiter: string, hasHeader: boolean) => {
      if (!currentWorkspace || !isSheet) {
        setStatus('仅表格台账支持粘贴导入。');
        return;
      }
      const { data } = await api.post<{ workspace: Workspace }>(
        `/api/v1/workspaces/${currentWorkspace.id}/import/text`,
        {
          text,
          delimiter,
          hasHeader
        }
      );
      if (data?.workspace) {
        await loadWorkspace(data.workspace.id);
        setStatus('粘贴内容已导入。');
      }
    },
    [currentWorkspace, isSheet, loadWorkspace]
  );

  const handleExport = useCallback(async () => {
    if (!currentWorkspace || !isSheet) {
      setStatus('仅表格台账支持导出 Excel。');
      return;
    }
    try {
      const { data } = await api.get<ArrayBuffer>(
        `/api/v1/workspaces/${currentWorkspace.id}/export`,
        { responseType: 'arraybuffer' }
      );
      const blob = new Blob([data], {
        type: 'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet'
      });
      const url = URL.createObjectURL(blob);
      const anchor = document.createElement('a');
      anchor.href = url;
      anchor.download = `${(currentWorkspace.name || DEFAULT_TITLE).replace(/\s+/g, '_')}.xlsx`;
      document.body.appendChild(anchor);
      anchor.click();
      anchor.remove();
      URL.revokeObjectURL(url);
      setStatus('已导出 Excel 文件。');
    } catch (err) {
      console.error('导出失败', err);
      const axiosError = err as AxiosError<{ error?: string }>;
      setError(axiosError.response?.data?.error || axiosError.message || '导出失败，请稍后再试。');
    }
  }, [currentWorkspace, isSheet]);

  const sidebar = (
    <LedgerListCard
      items={workspaceTree}
      selectedId={selectedId}
      onSelect={(node) => setSelectedId(node.id)}
      onCreate={handleCreateWorkspace}
      search={listQuery}
      onSearchChange={setListQuery}
      creationOptions={CREATION_OPTIONS}
      formatTimestamp={formatTimestamp}
    />
  );

  const editorHeader = (
    <Fragment>
      <input
        value={name}
        onChange={(event) => setName(event.target.value)}
        className="w-full rounded-full border border-black/5 bg-white/90 px-4 py-2 text-lg font-semibold text-[var(--text)] focus:border-[var(--accent)] focus:outline-none focus:ring-2 focus:ring-[var(--accent)]/20"
      />
    </Fragment>
  );

  const editorToolbar = (
    <ToolbarActions
      onSave={handleSave}
      onDelete={handleDeleteWorkspace}
      onExcelImport={() => excelInputRef.current?.click()}
      onPasteImport={() => setShowPasteModal(true)}
      onExport={handleExport}
      disabledSheetActions={!isSheet}
      busy={saving}
      dirty={hasUnsavedChanges}
    />
  );

  const editorStatus = <ToastStack status={status} error={error} />;

  const editorBody = (
    <Fragment>
      {loading && (
        <div className="rounded-[var(--radius-lg)] border border-black/5 bg-white/80 p-6 text-center text-sm text-[var(--muted)]">
          正在加载台账内容…
        </div>
      )}
      {!loading && currentWorkspace && (
        <Fragment>
          {isSheet && (
            <InlineTableEditor
              columns={columns}
              rows={rows}
              filteredRows={filteredRows}
              selectedRowIds={selectedRowIds}
              onToggleRowSelection={toggleRowSelection}
              onToggleSelectAll={toggleSelectAll}
              onUpdateColumnTitle={updateColumnTitle}
              onRemoveColumn={removeColumn}
              onResizeColumn={handleColumnResize}
              onUpdateCell={updateCell}
              onRemoveRow={removeRow}
              onAddRow={addRow}
              onAddColumn={addColumn}
              searchTerm={tableSearch}
              onSearchTermChange={setTableSearch}
              onOpenBatchEdit={() => setShowBatchEditModal(true)}
              onRemoveSelected={handleRemoveSelectedRows}
              hasSelection={hasSelection}
              selectAllState={selectAllState}
              isSheet={isSheet}
              minColumnWidth={MIN_COLUMN_WIDTH}
              defaultColumnWidth={DEFAULT_COLUMN_WIDTH}
            />
          )}
          {showDocumentEditor && (
            <DocumentEditor
              value={documentContent}
              editable={isDocument}
              onChange={setDocumentContent}
              onStatus={setStatus}
            />
          )}
          {isFolder && (
            <section className="mt-10 space-y-4">
              <div className="flex items-center justify-between text-sm">
                <h3 className="text-base font-semibold text-[var(--text)]">文件夹内容</h3>
                <span className="text-xs text-[var(--muted)]">共 {folderChildren.length} 项</span>
              </div>
              {folderChildren.length ? (
                <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-3">
                  {folderChildren.map((child) => {
                    const Icon = child.kind === 'folder' ? FolderIcon : child.kind === 'document' ? DocumentTextIcon : TableCellsIcon;
                    return (
                      <button
                        key={child.id}
                        type="button"
                        onClick={() => setSelectedId(child.id)}
                        className="flex w-full flex-col gap-1 rounded-[var(--radius-md)] border border-black/5 bg-white/90 p-4 text-left text-sm text-[var(--muted)] transition hover:border-[var(--accent)]/60 hover:text-[var(--text)]"
                      >
                        <Icon className="h-5 w-5 text-[var(--accent)]" />
                        <span className="truncate font-medium text-[var(--text)]">{child.name || DEFAULT_TITLE}</span>
                        {formatTimestamp(child.updatedAt) && (
                          <span className="text-[10px] text-[var(--muted)]">更新于 {formatTimestamp(child.updatedAt)}</span>
                        )}
                      </button>
                    );
                  })}
                </div>
              ) : (
                <div className="rounded-[var(--radius-lg)] border border-dashed border-black/10 bg-white/80 p-6 text-sm text-[var(--muted)]">
                  当前文件夹暂无内容，可使用左上角“新建”按钮创建台账或文档。
                </div>
              )}
            </section>
          )}
        </Fragment>
      )}
      {!loading && !currentWorkspace && (
        <div className="rounded-[var(--radius-lg)] border border-black/5 bg-white/80 p-6 text-center text-sm text-[var(--muted)]">
          请选择左侧的台账或新建一个台账以开始编辑。
        </div>
      )}
    </Fragment>
  );

  return (
    <div className="eidos-ledger-root">
      <div className="eidos-ledger-wrapper">
        <LedgerLayout
          sidebar={sidebar}
          editor={
            <LedgerEditorCard title={editorHeader} toolbar={editorToolbar} status={editorStatus}>
              {editorBody}
            </LedgerEditorCard>
          }
        />
      </div>

      <input
        ref={excelInputRef}
        type="file"
        accept=".xlsx,application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
        className="hidden"
        onChange={handleImportExcel}
      />

      <BatchEditModal
        open={showBatchEditModal}
        onClose={() => setShowBatchEditModal(false)}
        columns={isSheet ? columns : []}
        onApply={handleBatchEditApply}
        selectedCount={selectedRowIds.length}
      />

      <PasteModal open={showPasteModal} onClose={() => setShowPasteModal(false)} onSubmit={handleImportText} />
    </div>
  );
};

export default Assets;
