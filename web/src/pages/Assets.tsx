import { ChangeEvent, Fragment, useEffect, useMemo, useRef, useState } from 'react';
import {
  ArrowDownTrayIcon,
  ArrowUpTrayIcon,
  PhotoIcon,
  PlusIcon,
  TrashIcon,
  XMarkIcon,
  ClipboardDocumentListIcon,
  PencilSquareIcon,
  CheckIcon
} from '@heroicons/react/24/outline';
import type { AxiosError } from 'axios';

import api from '../api/client';

type WorkspaceColumn = {
  id: string;
  title: string;
  width?: number;
};

type WorkspaceRow = {
  id: string;
  cells: Record<string, string>;
};

type Workspace = {
  id: string;
  name: string;
  columns: WorkspaceColumn[];
  rows: WorkspaceRow[];
  document?: string;
};

type WorkspaceSummary = {
  id: string;
  name: string;
  updatedAt?: string;
};

const DEFAULT_TITLE = '未命名台账';

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

const ToolbarButton = ({
  icon: Icon,
  label,
  onClick
}: {
  icon: typeof ArrowDownTrayIcon;
  label: string;
  onClick: () => void;
}) => (
  <button
    type="button"
    onClick={onClick}
    className="flex items-center gap-1 rounded-xl border border-white bg-white/80 px-3 py-2 text-xs font-medium text-night-300 transition hover:text-night-50"
  >
    <Icon className="h-4 w-4" />
    {label}
  </button>
);

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

const Assets = () => {
  const [workspaces, setWorkspaces] = useState<WorkspaceSummary[]>([]);
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
  const documentRef = useRef<HTMLDivElement>(null);
  const excelInputRef = useRef<HTMLInputElement>(null);
  const imageInputRef = useRef<HTMLInputElement>(null);

  const selectedWorkspace = useMemo(() => {
    if (!currentWorkspace) return null;
    return {
      ...currentWorkspace,
      columns,
      rows,
      document: documentContent
    };
  }, [currentWorkspace, columns, rows, documentContent]);

  const refreshList = async () => {
    try {
      const { data } = await api.get<{ items: WorkspaceSummary[] }>('/api/v1/workspaces');
      const summaries = Array.isArray(data?.items) ? data.items : [];
      setWorkspaces(summaries);
      if (summaries.length && !selectedId) {
        setSelectedId(summaries[0].id);
      } else if (!summaries.length) {
        setSelectedId(null);
        setCurrentWorkspace(null);
        setColumns([]);
        setRows([]);
        setDocumentContent('');
        setName('');
      }
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
      const normalizedColumns = workspace.columns.length
        ? workspace.columns
        : [
            {
              id: generateId(),
              title: '列 1'
            }
          ];
      const normalizedRows = workspace.rows.length
        ? workspace.rows.map((row) => fillRowCells(row, normalizedColumns))
        : [];

      setCurrentWorkspace(workspace);
      setColumns(normalizedColumns);
      setRows(normalizedRows);
      setName(workspace.name || DEFAULT_TITLE);
      setDocumentContent(workspace.document || '');

      if (documentRef.current) {
        documentRef.current.innerHTML = workspace.document || '';
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

  const updateColumnTitle = (id: string, title: string) => {
    setColumns((prev) =>
      prev.map((column) => (column.id === id ? { ...column, title } : column))
    );
  };

  const removeColumn = (id: string) => {
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
    const title = window.prompt('请输入新列的标题', `列 ${columns.length + 1}`);
    if (!title) {
      return;
    }
    const id = generateId();
    const column: WorkspaceColumn = { id, title };
    setColumns((prev) => [...prev, column]);
    setRows((prev) => prev.map((row) => ({ ...row, cells: { ...row.cells, [id]: '' } })));
  };

  const addRow = () => {
    setRows((prev) => [...prev, createEmptyRow(columns)]);
  };

  const removeRow = (id: string) => {
    setRows((prev) => prev.filter((row) => row.id !== id));
  };

  const updateCell = (rowId: string, columnId: string, value: string) => {
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
    const payload = {
      name: name.trim() || DEFAULT_TITLE,
      document: documentContent,
      columns,
      rows: rows.map((row) => ({ id: row.id, cells: row.cells }))
    };
    try {
      await api.put(`/api/v1/workspaces/${currentWorkspace.id}`, payload);
      setStatus('已保存所有更改。');
      await refreshList();
    } catch (err) {
      console.error('保存失败', err);
      const axiosError = err as AxiosError<{ error?: string }>;
      setError(axiosError.response?.data?.error || axiosError.message || '保存失败，请稍后再试。');
    }
  };

  const handleCreateWorkspace = async () => {
    const title = window.prompt('新台账名称', DEFAULT_TITLE) || DEFAULT_TITLE;
    try {
      const { data } = await api.post<{ workspace: Workspace }>('/api/v1/workspaces', {
        name: title,
        document: '',
        columns: [],
        rows: []
      });
      await refreshList();
      if (data?.workspace?.id) {
        setSelectedId(data.workspace.id);
      }
      setStatus('新台账已创建。');
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
      setStatus('台账已删除。');
      await refreshList();
    } catch (err) {
      console.error('删除台账失败', err);
      const axiosError = err as AxiosError<{ error?: string }>;
      setError(axiosError.response?.data?.error || axiosError.message || '删除失败，请稍后再试。');
    }
  };

  const handleImportExcel = async (event: ChangeEvent<HTMLInputElement>) => {
    if (!currentWorkspace) return;
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
    if (!currentWorkspace) return;
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
    if (!currentWorkspace) return;
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

  const execCommand = (command: string, value?: string) => {
    if (!documentRef.current) return;
    documentRef.current.focus();
    try {
      document.execCommand(command, false, value);
      setDocumentContent(documentRef.current.innerHTML);
    } catch (err) {
      console.warn('执行编辑命令失败', command, err);
    }
  };

  const handleImageInsert = (event: ChangeEvent<HTMLInputElement>) => {
    const file = event.target.files?.[0];
    event.target.value = '';
    if (!file || !documentRef.current) return;
    const reader = new FileReader();
    reader.onload = () => {
      const result = reader.result;
      if (typeof result === 'string') {
        execCommand('insertImage', result);
      }
    };
    reader.readAsDataURL(file);
  };

  const workspaceList = (
    <div className="space-y-2">
      <button
        type="button"
        onClick={handleCreateWorkspace}
        className="flex w-full items-center justify-center gap-2 rounded-2xl border border-neon-500/40 bg-white/70 px-3 py-3 text-sm font-medium text-neon-500 transition hover:bg-neon-500/10"
      >
        <PlusIcon className="h-4 w-4" />
        新建台账
      </button>
      <div className="space-y-1">
        {workspaces.map((item) => (
          <button
            key={item.id}
            type="button"
            onClick={() => setSelectedId(item.id)}
            className={`w-full rounded-2xl border px-4 py-3 text-left text-sm transition ${
              item.id === selectedId
                ? 'border-neon-500/60 bg-neon-500/10 text-neon-500 shadow-glow'
                : 'border-white bg-white/70 text-night-300 hover:text-night-50'
            }`}
          >
            <div className="flex items-center justify-between gap-2">
              <span className="truncate font-medium">{item.name || DEFAULT_TITLE}</span>
              <ClipboardDocumentListIcon className="h-4 w-4" />
            </div>
            {item.updatedAt && (
              <p className="mt-1 text-xs text-night-400">{new Date(item.updatedAt).toLocaleString()}</p>
            )}
          </button>
        ))}
      </div>
    </div>
  );

  const tableHeader = (
    <thead className="bg-night-900/10">
      <tr>
        {columns.map((column) => (
          <th key={column.id} className="px-3 py-2 text-left text-xs font-semibold text-night-400">
            <div className="flex items-center gap-2">
              <input
                value={column.title}
                onChange={(event) => updateColumnTitle(column.id, event.target.value)}
                className="w-full rounded-lg border border-transparent bg-transparent px-2 py-1 text-sm text-night-100 focus:border-neon-400 focus:outline-none"
              />
              <button
                type="button"
                onClick={() => removeColumn(column.id)}
                className="rounded-full border border-night-700/50 p-1 text-night-400 hover:text-red-500"
                aria-label="删除列"
              >
                <TrashIcon className="h-4 w-4" />
              </button>
            </div>
          </th>
        ))}
        <th className="px-3 py-2" />
      </tr>
    </thead>
  );

  const tableBody = (
    <tbody>
      {rows.map((row) => (
        <tr key={row.id} className="border-t border-night-100/20">
          {columns.map((column) => (
            <td key={column.id} className="px-3 py-2">
              <input
                value={row.cells[column.id] ?? ''}
                onChange={(event) => updateCell(row.id, column.id, event.target.value)}
                className="w-full rounded-xl border border-white bg-white/70 px-3 py-2 text-sm text-night-500 focus:border-neon-400 focus:outline-none focus:ring-2 focus:ring-neon-400/30"
              />
            </td>
          ))}
          <td className="px-3 py-2 text-right">
            <button
              type="button"
              onClick={() => removeRow(row.id)}
              className="rounded-full border border-night-200/60 p-2 text-night-300 hover:text-red-500"
              aria-label="删除行"
            >
              <TrashIcon className="h-4 w-4" />
            </button>
          </td>
        </tr>
      ))}
    </tbody>
  );

  return (
    <div className="flex h-full flex-1 flex-col overflow-hidden">
      <div className="flex flex-1 flex-col gap-6 overflow-hidden px-6 py-6 lg:flex-row">
        <aside className="w-full max-w-xs space-y-6 rounded-3xl border border-white bg-white/80 p-5 shadow-sm lg:w-72 xl:w-80">
          <h2 className="text-lg font-semibold text-night-100">台账列表</h2>
          {workspaceList}
        </aside>
        <main className="flex-1 overflow-y-auto rounded-3xl border border-white bg-white/90 p-6 shadow-sm">
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
                    <ToolbarButton icon={ArrowUpTrayIcon} label="Excel 导入" onClick={() => excelInputRef.current?.click()} />
                    <ToolbarButton icon={ClipboardDocumentListIcon} label="粘贴导入" onClick={() => setShowPasteModal(true)} />
                    <ToolbarButton icon={ArrowDownTrayIcon} label="导出 Excel" onClick={handleExport} />
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
                      删除台账
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

              <section className="mt-6 space-y-4">
                <div className="flex flex-wrap items-center gap-3">
                  <h3 className="text-base font-semibold text-night-100">表格数据</h3>
                  <button
                    type="button"
                    className="flex items-center gap-1 rounded-xl border border-night-200/60 bg-white/80 px-3 py-2 text-xs text-night-300 hover:text-night-50"
                    onClick={addColumn}
                  >
                    <PlusIcon className="h-4 w-4" />
                    新增列
                  </button>
                  <button
                    type="button"
                    className="flex items-center gap-1 rounded-xl border border-night-200/60 bg-white/80 px-3 py-2 text-xs text-night-300 hover:text-night-50"
                    onClick={addRow}
                  >
                    <PlusIcon className="h-4 w-4" />
                    新增行
                  </button>
                </div>
                <div className="overflow-x-auto rounded-3xl border border-night-100/30">
                  <table className="min-w-full divide-y divide-night-100/20 text-sm">
                    {tableHeader}
                    {tableBody}
                  </table>
                </div>
              </section>

              <section className="mt-10 space-y-4">
                <div className="flex flex-wrap items-center gap-3">
                  <h3 className="text-base font-semibold text-night-100">在线文档</h3>
                  <div className="flex flex-wrap items-center gap-2 text-xs text-night-300">
                    <ToolbarButton icon={PencilSquareIcon} label="粗体" onClick={() => execCommand('bold')} />
                    <ToolbarButton icon={CheckIcon} label="斜体" onClick={() => execCommand('italic')} />
                    <ToolbarButton icon={PlusIcon} label="下划线" onClick={() => execCommand('underline')} />
                    <ToolbarButton icon={PhotoIcon} label="插入图片" onClick={() => imageInputRef.current?.click()} />
                  </div>
                </div>
                <div
                  ref={documentRef}
                  contentEditable
                  suppressContentEditableWarning
                  onInput={(event) => setDocumentContent((event.target as HTMLDivElement).innerHTML)}
                  className="min-h-[240px] rounded-3xl border border-night-100/40 bg-white/80 p-5 text-sm leading-relaxed text-night-500 focus:outline-none"
                />
              </section>
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

      <PasteModal open={showPasteModal} onClose={() => setShowPasteModal(false)} onSubmit={handleImportText} />
    </div>
  );
};

export default Assets;
