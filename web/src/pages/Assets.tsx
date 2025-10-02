import {
  ChangeEvent,
  Fragment,
  MouseEvent as ReactMouseEvent,
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState
} from 'react';
import {
  ArrowDownTrayIcon,
  ArrowUpTrayIcon,
  PlusIcon,
  TrashIcon,
  XMarkIcon,
  ClipboardDocumentListIcon,
  PencilSquareIcon,
  ChevronDownIcon,
  DocumentTextIcon,
  FolderIcon,
  TableCellsIcon
} from '@heroicons/react/24/outline';
import type { AxiosError } from 'axios';

import api from '../api/client';

type WorkspaceKind = 'sheet' | 'document' | 'folder';

type WorkspaceColumn = {
  id: string;
  title: string;
  width?: number;
};

type WorkspaceRow = {
  id: string;
  cells: Record<string, string>;
};

type WorkspaceNode = {
  id: string;
  name: string;
  kind: WorkspaceKind;
  parentId?: string | null;
  createdAt?: string;
  updatedAt?: string;
  children?: WorkspaceNode[];
};

type Workspace = {
  id: string;
  name: string;
  kind: WorkspaceKind;
  parentId?: string | null;
  columns: WorkspaceColumn[];
  rows: WorkspaceRow[];
  document?: string;
  createdAt?: string;
  updatedAt?: string;
};

const DEFAULT_TITLE = '未命名台账';
const MIN_COLUMN_WIDTH = 140;
const DEFAULT_COLUMN_WIDTH = 220;

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
      className="w-full resize-none overflow-hidden rounded-xl border border-night-100/40 bg-white/80 px-3 py-2 text-sm text-night-500 outline-none transition focus:border-neon-400 focus:ring-2 focus:ring-neon-400/20 whitespace-pre-wrap break-words"
    />
  );
};

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

const CREATION_OPTIONS: Array<{
  kind: WorkspaceKind;
  label: string;
  description: string;
  icon: typeof TableCellsIcon;
}> = [
  {
    kind: 'sheet',
    label: '新建表格台账',
    description: '用于结构化字段的协同维护',
    icon: TableCellsIcon
  },
  {
    kind: 'document',
    label: '新建在线文档',
    description: '记录会议纪要、方案与流程说明',
    icon: DocumentTextIcon
  },
  {
    kind: 'folder',
    label: '新建文件夹',
    description: '整理不同台账与文档',
    icon: FolderIcon
  }
];

const ToolbarButton = ({
  icon: Icon,
  label,
  onClick,
  disabled = false
}: {
  icon: typeof ArrowDownTrayIcon;
  label: string;
  onClick: () => void;
  disabled?: boolean;
}) => (
  <button
    type="button"
    onClick={onClick}
    disabled={disabled}
    className="flex items-center gap-2 rounded-2xl border border-white/60 bg-white/90 px-3 py-2 text-xs font-medium text-night-400 shadow-sm transition hover:border-neon-400/70 hover:text-neon-500 disabled:cursor-not-allowed disabled:border-night-100/40 disabled:text-night-200"
  >
    <Icon className="h-4 w-4" />
    <span>{label}</span>
  </button>
);

const DocumentToolbarButton = ({
  label,
  tooltip,
  onClick,
  disabled = false
}: {
  label: string;
  tooltip?: string;
  onClick: () => void;
  disabled?: boolean;
}) => (
  <button
    type="button"
    onClick={onClick}
    disabled={disabled}
    title={tooltip ?? label}
    aria-label={tooltip ?? label}
    className="rounded-2xl border border-white/60 bg-white/90 px-2 py-1 text-[11px] font-semibold uppercase tracking-wide text-night-400 shadow-sm transition hover:border-neon-400/60 hover:text-neon-500 disabled:cursor-not-allowed disabled:border-night-100/40 disabled:text-night-200"
  >
    {label}
  </button>
);

type DocumentBlockValue = 'paragraph' | 'h1' | 'h2' | 'h3' | 'quote' | 'code';

const DOCUMENT_BLOCK_OPTIONS: Array<{ value: DocumentBlockValue; label: string; command: string }> = [
  { value: 'paragraph', label: '正文', command: 'P' },
  { value: 'h1', label: '标题 1', command: 'H1' },
  { value: 'h2', label: '标题 2', command: 'H2' },
  { value: 'h3', label: '标题 3', command: 'H3' },
  { value: 'quote', label: '引用', command: 'BLOCKQUOTE' },
  { value: 'code', label: '代码块', command: 'PRE' }
];

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
    <div className="fixed inset-0 z-40 flex items-center justify-center bg-night-900/60 px-4">
      <div className="w-full max-w-2xl rounded-3xl border border-white bg-white/95 p-6 shadow-2xl">
        <div className="mb-4 flex items-center justify-between">
          <h3 className="text-lg font-semibold text-night-100">批量粘贴导入</h3>
          <button
            type="button"
            onClick={onClose}
            className="rounded-full border border-night-700/40 p-2 text-night-400 hover:text-night-50"
            aria-label="关闭"
          >
            <XMarkIcon className="h-5 w-5" />
          </button>
        </div>
        <div className="space-y-3">
          <textarea
            value={text}
            onChange={(event) => setText(event.target.value)}
            placeholder="将 Excel 或其他表格数据复制后粘贴在此处，每列使用制表符分隔"
            rows={10}
            className="w-full rounded-2xl border border-night-200/50 bg-white/70 p-3 text-sm text-night-500 focus:border-neon-500 focus:outline-none focus:ring-2 focus:ring-neon-400/30"
          />
          <div className="flex flex-wrap items-center gap-4 text-sm text-night-300">
            <label className="flex items-center gap-2">
              <span>分隔符</span>
              <select
                value={delimiter}
                onChange={(event) => setDelimiter(event.target.value as typeof delimiter)}
                className="rounded-xl border border-night-200/60 bg-white/80 px-2 py-1 text-night-400 focus:border-neon-500 focus:outline-none"
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
                className="rounded border-night-200/60 text-neon-500 focus:ring-neon-500"
              />
              首行是表头
            </label>
          </div>
          {error && (
            <div className="rounded-2xl border border-red-400/60 bg-red-100/60 px-3 py-2 text-sm text-red-600">{error}</div>
          )}
          <div className="flex justify-end gap-3">
            <button
              type="button"
              className="rounded-xl border border-night-200/60 px-4 py-2 text-sm text-night-300 hover:text-night-50"
              onClick={onClose}
            >
              取消
            </button>
            <button
              type="button"
              className="button-primary"
              onClick={handleSubmit}
              disabled={submitting}
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
    <div className="fixed inset-0 z-40 flex items-center justify-center bg-night-900/60 px-4">
      <div className="w-full max-w-lg rounded-3xl border border-white bg-white/95 p-6 shadow-2xl">
        <div className="mb-4 flex items-center justify-between">
          <div>
            <h3 className="text-lg font-semibold text-night-100">批量编辑</h3>
            <p className="mt-1 text-xs text-night-400">将同步更新选中的 {selectedCount} 行内容。</p>
          </div>
          <button
            type="button"
            onClick={onClose}
            className="rounded-full border border-night-700/40 p-2 text-night-400 hover:text-night-50"
            aria-label="关闭"
          >
            <XMarkIcon className="h-5 w-5" />
          </button>
        </div>
        <div className="space-y-4 text-sm text-night-400">
          <label className="flex flex-col gap-2">
            <span className="text-xs text-night-300">选择字段</span>
            <select
              value={columnId}
              onChange={(event) => setColumnId(event.target.value)}
              className="rounded-2xl border border-night-100/60 bg-white/80 px-3 py-2 text-sm text-night-500 focus:border-neon-400 focus:outline-none focus:ring-2 focus:ring-neon-400/20"
            >
              {columns.map((column) => (
                <option key={column.id} value={column.id}>
                  {column.title || '未命名字段'}
                </option>
              ))}
            </select>
          </label>
          <label className="flex flex-col gap-2">
            <span className="text-xs text-night-300">新的内容</span>
            <textarea
              value={value}
              onChange={(event) => setValue(event.target.value)}
              rows={4}
              className="w-full resize-none rounded-2xl border border-night-100/60 bg-white/80 px-3 py-2 text-sm text-night-500 focus:border-neon-400 focus:outline-none focus:ring-2 focus:ring-neon-400/20"
            />
          </label>
          {error && (
            <div className="rounded-2xl border border-red-400/60 bg-red-100/60 px-3 py-2 text-sm text-red-600">{error}</div>
          )}
        </div>
        <div className="mt-6 flex justify-end gap-3 text-sm">
          <button
            type="button"
            className="rounded-xl border border-night-200/60 px-4 py-2 text-night-300 hover:text-night-50"
            onClick={onClose}
          >
            取消
          </button>
          <button
            type="button"
            className="button-primary"
            onClick={handleSubmit}
            disabled={submitting || !columnId}
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
  const [selectedBlock, setSelectedBlock] = useState<DocumentBlockValue>('paragraph');
  const [loading, setLoading] = useState(false);
  const [status, setStatus] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [showPasteModal, setShowPasteModal] = useState(false);
  const [showCreateMenu, setShowCreateMenu] = useState(false);
  const [showBatchEditModal, setShowBatchEditModal] = useState(false);
  const [searchTerm, setSearchTerm] = useState('');
  const [selectedRowIds, setSelectedRowIds] = useState<string[]>([]);
  const documentRef = useRef<HTMLDivElement>(null);
  const excelInputRef = useRef<HTMLInputElement>(null);
  const imageInputRef = useRef<HTMLInputElement>(null);
  const audioInputRef = useRef<HTMLInputElement>(null);
  const videoInputRef = useRef<HTMLInputElement>(null);
  const createMenuRef = useRef<HTMLDivElement>(null);
  const createButtonRef = useRef<HTMLButtonElement>(null);
  const selectAllRef = useRef<HTMLInputElement>(null);
  const resizeStateRef = useRef<{ columnId: string; startX: number; startWidth: number } | null>(null);

  const selectedWorkspace = useMemo(() => {
    if (!currentWorkspace) return null;
    return {
      ...currentWorkspace,
      columns,
      rows,
      document: documentContent
    };
  }, [currentWorkspace, columns, rows, documentContent]);

  const allNodes = useMemo(() => flattenWorkspaces(workspaceTree), [workspaceTree]);
  const workspaceMap = useMemo(() => {
    const map = new Map<string, WorkspaceNode>();
    allNodes.forEach((node) => {
      map.set(node.id, node);
    });
    return map;
  }, [allNodes]);

  const selectedNode = selectedId ? workspaceMap.get(selectedId) : null;
  const selectedKind: WorkspaceKind = selectedWorkspace?.kind ?? normalizeKind(selectedNode?.kind);
  const isSheet = selectedKind === 'sheet';
  const isDocument = selectedKind === 'document';
  const isFolder = selectedKind === 'folder';
  const canEditDocument = isSheet || isDocument;
  const folderChildren = selectedNode?.children ?? [];
  const filteredRows = useMemo(() => {
    if (!isSheet) {
      return rows;
    }
    const keyword = searchTerm.trim().toLowerCase();
    if (!keyword) {
      return rows;
    }
    return rows.filter((row) =>
      columns.some((column) =>
        (row.cells[column.id] ?? '')
          .toString()
          .toLowerCase()
          .includes(keyword)
      )
    );
  }, [rows, columns, searchTerm, isSheet]);

  const filteredRowIds = useMemo(() => filteredRows.map((row) => row.id), [filteredRows]);

  const selectedFilteredCount = useMemo(
    () => filteredRows.filter((row) => selectedRowIds.includes(row.id)).length,
    [filteredRows, selectedRowIds]
  );

  const hasSelection = selectedRowIds.length > 0;

  useEffect(() => {
    const checkbox = selectAllRef.current;
    if (!checkbox) {
      return;
    }
    const total = filteredRows.length;
    const selectedCount = selectedFilteredCount;
    checkbox.indeterminate = selectedCount > 0 && selectedCount < total;
    checkbox.checked = total > 0 && selectedCount === total;
  }, [filteredRows, selectedFilteredCount]);

  const toggleRowSelection = useCallback(
    (rowId: string) => {
      setSelectedRowIds((prev) =>
        prev.includes(rowId) ? prev.filter((id) => id !== rowId) : [...prev, rowId]
      );
    },
    [setSelectedRowIds]
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
          selectedRowIds.includes(row.id)
            ? { ...row, cells: { ...row.cells, [columnId]: value } }
            : row
        )
      );
    },
    [selectedRowIds, setRows]
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
  }, [selectedRowIds, setRows, setSelectedRowIds]);

  const handleResizeStart = useCallback(
    (event: ReactMouseEvent<HTMLSpanElement>, columnId: string) => {
      event.preventDefault();
      event.stopPropagation();
      const column = columns.find((item) => item.id === columnId);
      if (!column) {
        return;
      }
      resizeStateRef.current = {
        columnId,
        startX: event.clientX,
        startWidth: column.width ?? DEFAULT_COLUMN_WIDTH
      };
      document.body.style.cursor = 'col-resize';
    },
    [columns]
  );

  const refreshList = async (focusId?: string | null) => {
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
  };

  const loadWorkspace = async (id: string) => {
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
              width: Math.max(MIN_COLUMN_WIDTH, column.width ?? DEFAULT_COLUMN_WIDTH)
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
      setSelectedBlock('paragraph');

      if (documentRef.current) {
        documentRef.current.innerHTML = normalizedDocument;
      }
    } catch (err) {
      console.error('加载台账失败', err);
      const axiosError = err as AxiosError<{ error?: string }>;
      setError(axiosError.response?.data?.error || axiosError.message || '无法加载台账内容。');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    refreshList().catch((err) => console.error(err));
  }, []);

  useEffect(() => {
    if (selectedId) {
      loadWorkspace(selectedId).catch((err) => console.error(err));
    }
  }, [selectedId]);

  useEffect(() => {
    setSearchTerm('');
    setSelectedRowIds([]);
  }, [currentWorkspace?.id]);

  useEffect(() => {
    if (!showCreateMenu) {
      return;
    }
    const handleClick = (event: MouseEvent) => {
      const target = event.target as Node;
      if (
        createMenuRef.current &&
        !createMenuRef.current.contains(target) &&
        createButtonRef.current &&
        !createButtonRef.current.contains(target)
      ) {
        setShowCreateMenu(false);
      }
    };
    document.addEventListener('mousedown', handleClick);
    return () => document.removeEventListener('mousedown', handleClick);
  }, [showCreateMenu]);

  useEffect(() => {
    const handleMouseMove = (event: MouseEvent) => {
      const state = resizeStateRef.current;
      if (!state) {
        return;
      }
      const delta = event.clientX - state.startX;
      setColumns((prev) =>
        prev.map((column) =>
          column.id === state.columnId
            ? {
                ...column,
                width: Math.max(MIN_COLUMN_WIDTH, Math.round((state.startWidth ?? DEFAULT_COLUMN_WIDTH) + delta))
              }
            : column
        )
      );
    };
    const handleMouseUp = () => {
      if (resizeStateRef.current) {
        resizeStateRef.current = null;
        document.body.style.cursor = '';
      }
    };
    window.addEventListener('mousemove', handleMouseMove);
    window.addEventListener('mouseup', handleMouseUp);
    return () => {
      window.removeEventListener('mousemove', handleMouseMove);
      window.removeEventListener('mouseup', handleMouseUp);
    };
  }, []);

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

  const updateColumnTitle = (id: string, title: string) => {
    if (!isSheet) return;
    setColumns((prev) =>
      prev.map((column) => (column.id === id ? { ...column, title } : column))
    );
  };

  const removeColumn = (id: string) => {
    if (!isSheet) return;
    setColumns((prev) => prev.filter((column) => column.id !== id));
    setRows((prev) =>
      prev.map((row) => {
        const nextCells = { ...row.cells };
        delete nextCells[id];
        return { ...row, cells: nextCells };
      })
    );
  };

  const addColumn = () => {
    if (!isSheet) return;
    const title = window.prompt('请输入新列的标题', `列 ${columns.length + 1}`);
    if (!title) {
      return;
    }
    const id = generateId();
    const column: WorkspaceColumn = { id, title, width: DEFAULT_COLUMN_WIDTH };
    setColumns((prev) => [...prev, column]);
    setRows((prev) => prev.map((row) => ({ ...row, cells: { ...row.cells, [id]: '' } })));
  };

  const addRow = () => {
    if (!isSheet) return;
    setRows((prev) => [...prev, createEmptyRow(columns)]);
  };

  const removeRow = (id: string) => {
    if (!isSheet) return;
    setRows((prev) => prev.filter((row) => row.id !== id));
  };

  const updateCell = (rowId: string, columnId: string, value: string) => {
    if (!isSheet) return;
    setRows((prev) =>
      prev.map((row) =>
        row.id === rowId ? { ...row, cells: { ...row.cells, [columnId]: value } } : row
      )
    );
  };

  const handleSave = async () => {
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
    try {
      await api.put(`/api/v1/workspaces/${currentWorkspace.id}`, payload);
      setStatus('已保存所有更改。');
      await refreshList(currentWorkspace.id);
    } catch (err) {
      console.error('保存失败', err);
      const axiosError = err as AxiosError<{ error?: string }>;
      setError(axiosError.response?.data?.error || axiosError.message || '保存失败，请稍后再试。');
    }
  };

  const resolveParentForCreation = () => {
    if (!selectedNode) {
      return '';
    }
    if (selectedNode.kind === 'folder') {
      return selectedNode.id;
    }
    return selectedNode.parentId ?? '';
  };

  const handleCreateWorkspace = async (kind: WorkspaceKind) => {
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
      parentId,
      document: '',
      columns: [] as WorkspaceColumn[],
      rows: [] as WorkspaceRow[]
    };
    if (kind === 'sheet') {
      payload.columns = [{ id: generateId(), title: '字段 1', width: DEFAULT_COLUMN_WIDTH }];
      payload.rows = [];
    }
    if (kind === 'document') {
      payload.document = '';
    }
    setShowCreateMenu(false);
    try {
      const { data } = await api.post<{ workspace: Workspace }>('/api/v1/workspaces', payload);
      const createdId = data?.workspace?.id;
      await refreshList(createdId ?? undefined);
      if (createdId) {
        setSelectedId(createdId);
      }
      setStatus(`${labels[kind].title}已创建。`);
    } catch (err) {
      console.error('创建台账失败', err);
      const axiosError = err as AxiosError<{ error?: string }>;
      setError(axiosError.response?.data?.error || axiosError.message || '创建台账失败。');
    }
  };

  const handleDeleteWorkspace = async () => {
    if (!currentWorkspace) {
      return;
    }
    if (!window.confirm(`确认删除「${currentWorkspace.name || DEFAULT_TITLE}」？该操作无法撤销。`)) {
      return;
    }
    try {
      await api.delete(`/api/v1/workspaces/${currentWorkspace.id}`);
      setStatus('内容已删除。');
      const fallback = selectedNode?.parentId ?? currentWorkspace.parentId ?? null;
      await refreshList(fallback);
    } catch (err) {
      console.error('删除台账失败', err);
      const axiosError = err as AxiosError<{ error?: string }>;
      setError(axiosError.response?.data?.error || axiosError.message || '删除失败，请稍后再试。');
    }
  };

  const handleImportExcel = async (event: ChangeEvent<HTMLInputElement>) => {
    if (!currentWorkspace || !isSheet) {
      setStatus('仅表格台账支持导入功能。');
      event.target.value = '';
      return;
    }
    const file = event.target.files?.[0];
    event.target.value = '';
    if (!file) return;
    const formData = new FormData();
    formData.append('file', file);
    try {
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
  };

  const handleImportText = async (text: string, delimiter: string, hasHeader: boolean) => {
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
  };

  const handleExport = async () => {
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
    } catch (err) {
      console.error('导出失败', err);
      const axiosError = err as AxiosError<{ error?: string }>;
      setError(axiosError.response?.data?.error || axiosError.message || '导出失败，请稍后再试。');
    }
  };

  const execCommand = useCallback(
    (command: string, value?: string) => {
      if (!canEditDocument || !documentRef.current) return;
      documentRef.current.focus();
      try {
        document.execCommand(command, false, value);
        setDocumentContent(documentRef.current.innerHTML);
      } catch (err) {
        console.warn('执行编辑命令失败', command, err);
      }
    },
    [canEditDocument]
  );

  const insertHtml = useCallback(
    (html: string) => {
      execCommand('insertHTML', html);
    },
    [execCommand]
  );

  const applyBlock = useCallback(
    (value: DocumentBlockValue) => {
      const option = DOCUMENT_BLOCK_OPTIONS.find((item) => item.value === value);
      execCommand('formatBlock', option?.command ?? 'P');
    },
    [execCommand]
  );

  const handleBlockSelect = useCallback(
    (event: ChangeEvent<HTMLSelectElement>) => {
      const nextValue = (event.target.value as DocumentBlockValue) ?? 'paragraph';
      setSelectedBlock(nextValue);
      applyBlock(nextValue);
    },
    [applyBlock]
  );

  const insertChecklist = useCallback(() => {
    insertHtml(
      '<ul class="doc-checklist"><li><label><input type="checkbox" /> 待办事项</label></li></ul>'
    );
  }, [insertHtml]);

  const insertCallout = useCallback(() => {
    insertHtml(
      '<div class="doc-callout"><strong>提示：</strong>请在此补充说明。</div>'
    );
  }, [insertHtml]);

  const insertCodeBlock = useCallback(() => {
    insertHtml('<pre class="doc-code"><code>// 在此编写示例代码</code></pre>');
  }, [insertHtml]);

  const insertTable = useCallback(() => {
    const rowsInput = window.prompt('请输入表格行数 (1-20)', '3');
    const colsInput = window.prompt('请输入表格列数 (1-10)', '3');
    let rowsCount = Number.parseInt(rowsInput ?? '0', 10);
    let colsCount = Number.parseInt(colsInput ?? '0', 10);
    if (!Number.isFinite(rowsCount) || rowsCount <= 0) rowsCount = 3;
    if (!Number.isFinite(colsCount) || colsCount <= 0) colsCount = 3;
    rowsCount = Math.min(Math.max(rowsCount, 1), 20);
    colsCount = Math.min(Math.max(colsCount, 1), 10);
    const headerCells = Array.from({ length: colsCount })
      .map((_, index) => `<th>列 ${index + 1}</th>`)
      .join('');
    const bodyRows = Array.from({ length: rowsCount })
      .map(() => {
        const cells = Array.from({ length: colsCount })
          .map(() => '<td>填写内容</td>')
          .join('');
        return `<tr>${cells}</tr>`;
      })
      .join('');
    insertHtml(
      `<table class="doc-table"><thead><tr>${headerCells}</tr></thead><tbody>${bodyRows}</tbody></table>`
    );
  }, [insertHtml]);

  const insertDivider = useCallback(() => {
    execCommand('insertHorizontalRule');
  }, [execCommand]);

  const handleInsertLink = useCallback(() => {
    const url = window.prompt('请输入链接地址', 'https://');
    if (url) {
      execCommand('createLink', url);
    }
  }, [execCommand]);

  const applyHighlight = useCallback(() => {
    execCommand('hiliteColor', '#FFF9C4');
  }, [execCommand]);

  const clearFormatting = useCallback(() => {
    if (documentRef.current) {
      const normalized = documentRef.current.innerHTML.replace(
        /<mark class="doc-search-highlight">(.*?)<\/mark>/g,
        '$1'
      );
      documentRef.current.innerHTML = normalized;
      setDocumentContent(normalized);
    }
    execCommand('removeFormat');
    execCommand('unlink');
  }, [documentRef, execCommand, setDocumentContent]);

  const handleImageInsert = useCallback(
    (event: ChangeEvent<HTMLInputElement>) => {
      const file = event.target.files?.[0];
      event.target.value = '';
      if (!file || !canEditDocument) return;
      const reader = new FileReader();
      reader.onload = () => {
        const result = reader.result;
        if (typeof result === 'string') {
          insertHtml(
            `<figure class="doc-figure"><img src="${result}" alt="插入图片" /><figcaption>图片说明</figcaption></figure>`
          );
        }
      };
      reader.readAsDataURL(file);
    },
    [canEditDocument, insertHtml]
  );

  const handleMediaInsert = useCallback(
    (event: ChangeEvent<HTMLInputElement>, type: 'audio' | 'video') => {
      const file = event.target.files?.[0];
      event.target.value = '';
      if (!file || !canEditDocument) return;
      const reader = new FileReader();
      reader.onload = () => {
        const result = reader.result;
        if (typeof result === 'string') {
          const tag = type === 'audio' ? 'audio' : 'video';
          const extra = type === 'video' ? ' class="doc-video"' : '';
          insertHtml(`<${tag}${extra} controls src="${result}"></${tag}>`);
        }
      };
      reader.readAsDataURL(file);
    },
    [canEditDocument, insertHtml]
  );

  const handleDocumentSearch = useCallback(() => {
    if (!documentRef.current) {
      return;
    }
    const keyword = window.prompt('请输入要查找的内容');
    if (!keyword) {
      return;
    }
    const term = keyword.trim();
    if (!term) {
      return;
    }
    const container = documentRef.current;
    const resetHtml = container.innerHTML.replace(
      /<mark class="doc-search-highlight">(.*?)<\/mark>/g,
      '$1'
    );
    const escaped = term.replace(/[-/\\^$*+?.()|[\]{}]/g, '\\$&');
    const regex = new RegExp(escaped, 'gi');
    const hasMatch = regex.test(resetHtml);
    regex.lastIndex = 0;
    if (!hasMatch) {
      container.innerHTML = resetHtml;
      setDocumentContent(resetHtml);
      setStatus(`未在文档中找到 “${term}” 。`);
      return;
    }
    const highlighted = resetHtml.replace(
      regex,
      (match) => `<mark class="doc-search-highlight">${match}</mark>`
    );
    container.innerHTML = highlighted;
    setDocumentContent(highlighted);
    setStatus(`已高亮显示 “${term}” 的匹配内容。`);
  }, [setStatus]);

  const renderTree = (nodes: WorkspaceNode[], depth = 0): JSX.Element[] => {
    const items: JSX.Element[] = [];
    nodes.forEach((node) => {
      const Icon =
        node.kind === 'folder' ? FolderIcon : node.kind === 'document' ? DocumentTextIcon : TableCellsIcon;
      const isActive = node.id === selectedId;
      const paddingLeft = 12 + depth * 18;
      const typeLabel = node.kind === 'folder' ? '文件夹' : node.kind === 'document' ? '文档' : '表格';
      const displayName = node.name || DEFAULT_TITLE;
      items.push(
        <button
          key={node.id}
          type="button"
          onClick={() => setSelectedId(node.id)}
          style={{ paddingLeft: `${paddingLeft}px` }}
          className={`mt-1 flex w-full items-center gap-3 rounded-2xl border px-4 py-2 text-left text-sm transition ${
            isActive
              ? 'border-neon-500/60 bg-neon-500/10 text-neon-500 shadow-glow'
              : 'border-white bg-white/70 text-night-300 hover:text-night-50'
          }`}
        >
          <Icon className="h-4 w-4 shrink-0" />
          <div className="flex min-w-0 flex-1 flex-col">
            <span className="truncate font-medium">{displayName}</span>
            <div className="mt-1 flex flex-wrap items-center gap-2 text-[10px] text-night-400">
              <span className="rounded-full border border-night-200/60 bg-white/70 px-2 py-0.5">{typeLabel}</span>
              {formatTimestamp(node.updatedAt) && <span>{formatTimestamp(node.updatedAt)}</span>}
            </div>
          </div>
        </button>
      );
      if (node.children?.length) {
        items.push(...renderTree(node.children, depth + 1));
      }
    });
    return items;
  };

  const workspaceList = (
    <div className="space-y-2">
      <div className="relative">
        <button
          ref={createButtonRef}
          type="button"
          onClick={() => setShowCreateMenu((prev) => !prev)}
          className="flex w-full items-center justify-center gap-2 rounded-2xl border border-neon-500/40 bg-white/70 px-3 py-3 text-sm font-medium text-neon-500 transition hover:bg-neon-500/10"
        >
          <PlusIcon className="h-4 w-4" />
          新建
          <ChevronDownIcon className="h-4 w-4" />
        </button>
        {showCreateMenu && (
          <div
            ref={createMenuRef}
            className="absolute inset-x-0 top-full z-20 mt-2 rounded-2xl border border-night-100/40 bg-white/95 shadow-xl"
          >
            <ul className="divide-y divide-night-100/40 text-sm text-night-400">
              {CREATION_OPTIONS.map(({ kind, label, description, icon: OptionIcon }) => (
                <li key={kind}>
                  <button
                    type="button"
                    onClick={() => handleCreateWorkspace(kind)}
                    className="flex w-full items-start gap-3 px-4 py-3 text-left hover:bg-neon-500/5"
                  >
                    <OptionIcon className="mt-0.5 h-4 w-4 text-neon-500" />
                    <span>
                      <span className="block font-medium text-night-300">{label}</span>
                      <span className="mt-1 block text-xs text-night-400">{description}</span>
                    </span>
                  </button>
                </li>
              ))}
            </ul>
          </div>
        )}
      </div>
      <div className="space-y-1">
        {workspaceTree.length ? renderTree(workspaceTree) : (
          <div className="rounded-2xl border border-dashed border-night-200/60 bg-white/60 p-4 text-center text-sm text-night-300">
            暂无台账内容，点击上方“新建”开始创建。
          </div>
        )}
      </div>
    </div>
  );

  const columnSizing = (
    <colgroup>
      <col style={{ width: '48px' }} />
      {columns.map((column) => (
        <col key={column.id} style={{ width: column.width ? `${column.width}px` : undefined }} />
      ))}
      <col style={{ width: '80px' }} />
    </colgroup>
  );

  const tableHeader = (
    <thead className="bg-transparent">
      <tr className="text-left text-xs font-medium text-night-300">
        <th className="w-12 px-2 py-2">
          <input
            ref={selectAllRef}
            type="checkbox"
            onChange={toggleSelectAll}
            className="h-4 w-4 rounded border-night-200/60 text-neon-500 focus:ring-neon-500"
          />
        </th>
        {columns.map((column) => (
          <th
            key={column.id}
            className="relative px-3 py-2"
            style={{ minWidth: `${Math.max(MIN_COLUMN_WIDTH, (column.width ?? DEFAULT_COLUMN_WIDTH) - 32)}px` }}
          >
            <div className="flex items-center gap-2">
              <input
                value={column.title}
                onChange={(event) => updateColumnTitle(column.id, event.target.value)}
                className="w-full rounded-xl border border-transparent bg-transparent px-2 py-1 text-sm text-night-200 focus:border-neon-400 focus:outline-none"
              />
              <button
                type="button"
                onClick={() => removeColumn(column.id)}
                className="rounded-full border border-night-200/40 p-1 text-night-400 transition hover:text-red-500"
                aria-label="删除列"
              >
                <TrashIcon className="h-4 w-4" />
              </button>
            </div>
            <span
              role="separator"
              tabIndex={-1}
              onMouseDown={(event) => handleResizeStart(event, column.id)}
              className="absolute right-0 top-0 h-full w-2 cursor-col-resize select-none rounded-full bg-transparent transition hover:bg-neon-500/20"
            />
          </th>
        ))}
        <th className="px-3 py-2 text-right text-xs text-night-300">操作</th>
      </tr>
    </thead>
  );

  const tableBody = filteredRows.length ? (
    <tbody>
      {filteredRows.map((row) => {
        const isSelected = selectedRowIds.includes(row.id);
        return (
          <tr
            key={row.id}
            className={`align-top border-t border-night-100/10 transition ${
              isSelected ? 'bg-neon-500/5' : 'bg-white/60 hover:bg-white/80'
            }`}
          >
            <td className="px-2 py-3 align-top">
              <input
                type="checkbox"
                checked={isSelected}
                onChange={() => toggleRowSelection(row.id)}
                className="h-4 w-4 rounded border-night-200/60 text-neon-500 focus:ring-neon-500"
              />
            </td>
            {columns.map((column) => (
              <td key={column.id} className="px-3 py-2 align-top">
                <SheetCell
                  value={row.cells[column.id] ?? ''}
                  onChange={(next) => updateCell(row.id, column.id, next)}
                  placeholder={column.title}
                />
              </td>
            ))}
            <td className="px-3 py-2 text-right align-top">
              <button
                type="button"
                onClick={() => removeRow(row.id)}
                className="rounded-full border border-night-200/60 p-2 text-night-300 transition hover:text-red-500"
                aria-label="删除行"
              >
                <TrashIcon className="h-4 w-4" />
              </button>
            </td>
          </tr>
        );
      })}
    </tbody>
  ) : (
    <tbody>
      <tr>
        <td colSpan={columns.length + 2} className="px-6 py-10 text-center text-sm text-night-300">
          {rows.length
            ? '未找到匹配的记录，尝试调整搜索条件或清空筛选。'
            : '暂无记录，使用“新增行”或导入功能开始填写内容。'}
        </td>
      </tr>
    </tbody>
  );

  return (
    <div className="flex h-full flex-1 flex-col overflow-hidden">
      <div className="flex flex-1 flex-col gap-6 overflow-hidden px-6 py-6 lg:flex-row">
        <aside className="w-full max-w-xs space-y-6 rounded-3xl border border-white/60 bg-white/80 p-5 shadow-sm lg:w-72 xl:w-80">
          <h2 className="text-lg font-semibold text-night-100">台账列表</h2>
          {workspaceList}
        </aside>
        <main className="flex-1 overflow-y-auto rounded-3xl border border-white/50 bg-white/95 p-6 shadow-sm">
          {loading && (
            <div className="rounded-3xl border border-white bg-white/80 p-6 text-center text-night-300">
              正在加载台账内容…
            </div>
          )}

          {!loading && currentWorkspace && (
            <Fragment>
              <div className="flex flex-col gap-4 pb-6 border-b border-night-100/20">
                <div className="flex flex-wrap items-center gap-3">
                  <input
                    value={name}
                    onChange={(event) => setName(event.target.value)}
                    className="flex-1 rounded-2xl border border-white bg-white/80 px-4 py-2 text-lg font-semibold text-night-100 focus:border-neon-400 focus:outline-none focus:ring-2 focus:ring-neon-400/30"
                  />
                  <div className="flex flex-wrap items-center gap-2">
                    <ToolbarButton
                      icon={ArrowUpTrayIcon}
                      label="Excel 导入"
                      onClick={() => excelInputRef.current?.click()}
                      disabled={!isSheet}
                    />
                    <ToolbarButton
                      icon={ClipboardDocumentListIcon}
                      label="粘贴导入"
                      onClick={() => setShowPasteModal(true)}
                      disabled={!isSheet}
                    />
                    <ToolbarButton
                      icon={ArrowDownTrayIcon}
                      label="导出 Excel"
                      onClick={handleExport}
                      disabled={!isSheet}
                    />
                  </div>
                  <div className="flex items-center gap-2">
                    <button
                      type="button"
                      className="rounded-2xl border border-neon-500/60 bg-neon-500/10 px-4 py-2 text-sm font-medium text-neon-500 hover:bg-neon-500/20"
                      onClick={handleSave}
                    >
                      保存
                    </button>
                    <button
                      type="button"
                      className="rounded-2xl border border-red-400/60 bg-red-50 px-4 py-2 text-sm text-red-500 hover:bg-red-100"
                      onClick={handleDeleteWorkspace}
                    >
                      删除内容
                    </button>
                  </div>
                </div>
                {(status || error) && (
                  <div
                    className={`rounded-2xl border px-4 py-2 text-sm ${
                      error
                        ? 'border-red-400/60 bg-red-100/70 text-red-600'
                        : 'border-emerald-300/60 bg-emerald-100/70 text-emerald-700'
                    }`}
                  >
                    {error || status}
                  </div>
                )}
              </div>

              {isSheet && (
                <section className="mt-6 space-y-4">
                  <div className="flex flex-wrap items-center justify-between gap-3">
                    <div className="flex flex-wrap items-center gap-3">
                      <h3 className="text-base font-semibold text-night-100">表格数据</h3>
                      <button
                        type="button"
                        className="flex items-center gap-1 rounded-xl border border-night-200/50 bg-white/70 px-3 py-2 text-xs text-night-300 hover:text-night-50 disabled:cursor-not-allowed disabled:opacity-60"
                        onClick={addColumn}
                        disabled={!isSheet}
                      >
                        <PlusIcon className="h-4 w-4" />
                        新增列
                      </button>
                      <button
                        type="button"
                        className="flex items-center gap-1 rounded-xl border border-night-200/50 bg-white/70 px-3 py-2 text-xs text-night-300 hover:text-night-50 disabled:cursor-not-allowed disabled:opacity-60"
                        onClick={addRow}
                        disabled={!isSheet}
                      >
                        <PlusIcon className="h-4 w-4" />
                        新增行
                      </button>
                    </div>
                    <div className="flex flex-wrap items-center gap-2 text-xs text-night-300">
                      <div className="relative">
                        <input
                          value={searchTerm}
                          onChange={(event) => setSearchTerm(event.target.value)}
                          placeholder="搜索关键字"
                          className="w-44 rounded-2xl border border-night-200/60 bg-white/80 px-3 py-2 text-sm text-night-400 focus:border-neon-400 focus:outline-none focus:ring-2 focus:ring-neon-400/20"
                        />
                        {searchTerm && (
                          <button
                            type="button"
                            onClick={() => setSearchTerm('')}
                            className="absolute inset-y-0 right-2 flex items-center text-night-300 hover:text-night-50"
                            aria-label="清空搜索"
                          >
                            <XMarkIcon className="h-4 w-4" />
                          </button>
                        )}
                      </div>
                      <button
                        type="button"
                        onClick={() => hasSelection && columns.length && setShowBatchEditModal(true)}
                        className="flex items-center gap-1 rounded-xl border border-night-200/50 bg-white/70 px-3 py-2 text-xs text-night-300 transition hover:text-night-50 disabled:cursor-not-allowed disabled:opacity-60"
                        disabled={!hasSelection || !columns.length}
                      >
                        <PencilSquareIcon className="h-4 w-4" />
                        批量编辑
                      </button>
                      <button
                        type="button"
                        onClick={handleRemoveSelectedRows}
                        className="flex items-center gap-1 rounded-xl border border-night-200/50 bg-white/70 px-3 py-2 text-xs text-night-300 transition hover:text-red-500 disabled:cursor-not-allowed disabled:opacity-60"
                        disabled={!hasSelection}
                      >
                        <TrashIcon className="h-4 w-4" />
                        删除选中
                      </button>
                    </div>
                  </div>
                  {hasSelection && (
                    <div className="rounded-2xl bg-neon-500/10 px-3 py-2 text-xs text-neon-500">
                      已选中 {selectedRowIds.length} 行，支持批量编辑或删除。
                    </div>
                  )}
                  <div className="relative overflow-auto rounded-3xl bg-white/80 shadow-inner ring-1 ring-night-100/30">
                    <table className="min-w-[800px] text-sm leading-snug text-night-500">
                      {columnSizing}
                      {tableHeader}
                      {tableBody}
                    </table>
                  </div>
                </section>
              )}

              {canEditDocument && (
                <section className="mt-10 space-y-4">
                  <div className="flex flex-wrap items-center gap-3">
                    <h3 className="text-base font-semibold text-night-100">在线文档</h3>
                    <select
                      value={selectedBlock}
                      onChange={handleBlockSelect}
                      disabled={!canEditDocument}
                      className="rounded-2xl border border-night-200/60 bg-white/80 px-3 py-2 text-xs text-night-400 focus:border-neon-400 focus:outline-none focus:ring-2 focus:ring-neon-400/20 disabled:cursor-not-allowed"
                    >
                      {DOCUMENT_BLOCK_OPTIONS.map((option) => (
                        <option key={option.value} value={option.value}>
                          {option.label}
                        </option>
                      ))}
                    </select>
                    <div className="flex flex-wrap items-center gap-2 text-xs text-night-300">
                      <DocumentToolbarButton
                        label="B"
                        tooltip="加粗"
                        onClick={() => execCommand('bold')}
                        disabled={!canEditDocument}
                      />
                      <DocumentToolbarButton
                        label="I"
                        tooltip="斜体"
                        onClick={() => execCommand('italic')}
                        disabled={!canEditDocument}
                      />
                      <DocumentToolbarButton
                        label="U"
                        tooltip="下划线"
                        onClick={() => execCommand('underline')}
                        disabled={!canEditDocument}
                      />
                      <DocumentToolbarButton
                        label="S"
                        tooltip="删除线"
                        onClick={() => execCommand('strikeThrough')}
                        disabled={!canEditDocument}
                      />
                      <DocumentToolbarButton
                        label="HL"
                        tooltip="高亮"
                        onClick={applyHighlight}
                        disabled={!canEditDocument}
                      />
                      <DocumentToolbarButton
                        label="Link"
                        tooltip="插入链接"
                        onClick={handleInsertLink}
                        disabled={!canEditDocument}
                      />
                      <DocumentToolbarButton
                        label="•"
                        tooltip="项目符号列表"
                        onClick={() => execCommand('insertUnorderedList')}
                        disabled={!canEditDocument}
                      />
                      <DocumentToolbarButton
                        label="1."
                        tooltip="编号列表"
                        onClick={() => execCommand('insertOrderedList')}
                        disabled={!canEditDocument}
                      />
                      <DocumentToolbarButton
                        label="☑"
                        tooltip="任务清单"
                        onClick={insertChecklist}
                        disabled={!canEditDocument}
                      />
                      <DocumentToolbarButton
                        label="L"
                        tooltip="左对齐"
                        onClick={() => execCommand('justifyLeft')}
                        disabled={!canEditDocument}
                      />
                      <DocumentToolbarButton
                        label="C"
                        tooltip="居中"
                        onClick={() => execCommand('justifyCenter')}
                        disabled={!canEditDocument}
                      />
                      <DocumentToolbarButton
                        label="R"
                        tooltip="右对齐"
                        onClick={() => execCommand('justifyRight')}
                        disabled={!canEditDocument}
                      />
                      <DocumentToolbarButton
                        label="表"
                        tooltip="插入表格"
                        onClick={insertTable}
                        disabled={!canEditDocument}
                      />
                      <DocumentToolbarButton
                        label="<>"
                        tooltip="插入代码块"
                        onClick={insertCodeBlock}
                        disabled={!canEditDocument}
                      />
                      <DocumentToolbarButton
                        label="Call"
                        tooltip="提示块"
                        onClick={insertCallout}
                        disabled={!canEditDocument}
                      />
                      <DocumentToolbarButton
                        label="HR"
                        tooltip="分割线"
                        onClick={insertDivider}
                        disabled={!canEditDocument}
                      />
                      <DocumentToolbarButton
                        label="IMG"
                        tooltip="插入图片"
                        onClick={() => imageInputRef.current?.click()}
                        disabled={!canEditDocument}
                      />
                      <DocumentToolbarButton
                        label="AUD"
                        tooltip="插入音频"
                        onClick={() => audioInputRef.current?.click()}
                        disabled={!canEditDocument}
                      />
                      <DocumentToolbarButton
                        label="VID"
                        tooltip="插入视频"
                        onClick={() => videoInputRef.current?.click()}
                        disabled={!canEditDocument}
                      />
                      <DocumentToolbarButton
                        label="Find"
                        tooltip="搜索并高亮"
                        onClick={handleDocumentSearch}
                        disabled={!canEditDocument}
                      />
                      <DocumentToolbarButton
                        label="CLR"
                        tooltip="清除格式"
                        onClick={clearFormatting}
                        disabled={!canEditDocument}
                      />
                    </div>
                  </div>
                  <div
                    ref={documentRef}
                    contentEditable={canEditDocument}
                    suppressContentEditableWarning
                    data-placeholder="在此记录会议纪要、流程与操作说明…"
                    onInput={canEditDocument ? (event) => setDocumentContent((event.target as HTMLDivElement).innerHTML) : undefined}
                    className={`doc-editor min-h-[320px] rounded-[28px] border border-night-100/40 bg-white/90 p-6 text-sm leading-relaxed text-night-500 shadow-inner transition focus:outline-none ${
                      canEditDocument ? '' : 'pointer-events-none opacity-60'
                    }`}
                  />
                </section>
              )}

              {isFolder && (
                <section className="mt-10 space-y-4">
                  <div className="flex items-center justify-between">
                    <h3 className="text-base font-semibold text-night-100">文件夹内容</h3>
                    <span className="text-xs text-night-300">共 {folderChildren.length} 项</span>
                  </div>
                  {folderChildren.length ? (
                    <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-3">
                      {folderChildren.map((child) => {
                        const Icon =
                          child.kind === 'folder'
                            ? FolderIcon
                            : child.kind === 'document'
                            ? DocumentTextIcon
                            : TableCellsIcon;
                        return (
                          <button
                            key={child.id}
                            type="button"
                            onClick={() => setSelectedId(child.id)}
                            className="flex w-full flex-col gap-1 rounded-2xl border border-night-100/50 bg-white/80 p-4 text-left text-sm text-night-400 transition hover:border-neon-500/60 hover:text-neon-500"
                          >
                            <Icon className="h-5 w-5" />
                            <span className="truncate font-medium text-night-200">{child.name || DEFAULT_TITLE}</span>
                            {formatTimestamp(child.updatedAt) && (
                              <span className="text-[10px] text-night-400">更新于 {formatTimestamp(child.updatedAt)}</span>
                            )}
                          </button>
                        );
                      })}
                    </div>
                  ) : (
                    <div className="rounded-3xl border border-night-100/30 bg-white/80 p-6 text-sm text-night-300">
                      当前文件夹暂无内容，可使用左上角“新建”按钮创建台账或文档。
                    </div>
                  )}
                </section>
              )}
            </Fragment>
          )}

          {!loading && !currentWorkspace && (
            <div className="rounded-3xl border border-white bg-white/80 p-6 text-center text-night-300">
              请选择左侧的台账或新建一个台账以开始编辑。
            </div>
          )}
        </main>
      </div>

      <input
        ref={excelInputRef}
        type="file"
        accept=".xlsx,application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
        className="hidden"
        onChange={handleImportExcel}
      />
      <input ref={imageInputRef} type="file" accept="image/*" className="hidden" onChange={handleImageInsert} />
      <input
        ref={audioInputRef}
        type="file"
        accept="audio/*"
        className="hidden"
        onChange={(event) => handleMediaInsert(event, 'audio')}
      />
      <input
        ref={videoInputRef}
        type="file"
        accept="video/*"
        className="hidden"
        onChange={(event) => handleMediaInsert(event, 'video')}
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
