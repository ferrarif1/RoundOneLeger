import { ChangeEvent, FormEvent, KeyboardEvent, useEffect, useMemo, useRef, useState } from 'react';
import {
  ArrowDownIcon,
  ArrowUpIcon,
  ArrowUturnLeftIcon,
  ArrowUturnRightIcon,
  PencilSquareIcon,
  PlusIcon,
  TagIcon,
  TrashIcon,
  ClipboardDocumentCheckIcon,
  ArrowDownTrayIcon
} from '@heroicons/react/24/outline';
import type { AxiosError } from 'axios';

import api from '../api/client';

type LedgerType = 'ips' | 'devices' | 'personnel' | 'systems';

type FieldSource = 'name' | 'description' | 'attribute';

interface FieldConfig {
  key: string;
  label: string;
  placeholder: string;
  required?: boolean;
  source: FieldSource;
  attributeKey?: string;
}

interface LedgerConfig {
  type: LedgerType;
  label: string;
  endpoint: string;
  accent: string;
  fields: FieldConfig[];
  nameFieldKey: string;
  descriptionFieldKey?: string;
}

interface LedgerRecord {
  id: string;
  name: string;
  description?: string;
  attributes?: Record<string, string> | null;
  tags?: string[];
  links?: Record<string, string[]> | null;
  order: number;
  created_at?: string;
  updated_at?: string;
}

interface HistoryCounters {
  undoSteps: number;
  redoSteps: number;
}

const API_PREFIX = '/api/v1';

const LEDGER_CONFIGS: Record<LedgerType, LedgerConfig> = {
  ips: {
    type: 'ips',
    label: 'IP 白名单',
    endpoint: `${API_PREFIX}/ledgers/ips`,
    accent: 'text-neon-500',
    nameFieldKey: 'address',
    descriptionFieldKey: 'description',
    fields: [
      { key: 'address', label: 'IP 地址', placeholder: '10.0.0.12/24', required: true, source: 'attribute', attributeKey: 'address' },
      { key: 'description', label: '说明', placeholder: '办公网络出入口', source: 'description' }
    ]
  },
  devices: {
    type: 'devices',
    label: '终端设备',
    endpoint: `${API_PREFIX}/ledgers/devices`,
    accent: 'text-indigo-300',
    nameFieldKey: 'identifier',
    fields: [
      { key: 'identifier', label: '设备标识', placeholder: 'MacBook Pro SN', required: true, source: 'name' },
      { key: 'type', label: '类型', placeholder: 'Laptop', source: 'attribute', attributeKey: 'type' },
      { key: 'owner', label: '责任人', placeholder: '张三', source: 'attribute', attributeKey: 'owner' }
    ]
  },
  personnel: {
    type: 'personnel',
    label: '人员',
    endpoint: `${API_PREFIX}/ledgers/personnel`,
    accent: 'text-amber-300',
    nameFieldKey: 'name',
    fields: [
      { key: 'name', label: '姓名', placeholder: '李四', required: true, source: 'name' },
      { key: 'role', label: '角色', placeholder: '安全负责人', source: 'attribute', attributeKey: 'role' },
      { key: 'contact', label: '联系方式', placeholder: 'lisa@example.com', source: 'attribute', attributeKey: 'contact' }
    ]
  },
  systems: {
    type: 'systems',
    label: '系统',
    endpoint: `${API_PREFIX}/ledgers/systems`,
    accent: 'text-cyan-300',
    nameFieldKey: 'name',
    descriptionFieldKey: 'environment',
    fields: [
      { key: 'name', label: '系统名称', placeholder: '核心交易平台', required: true, source: 'name' },
      { key: 'environment', label: '环境', placeholder: '生产/预发', source: 'attribute', attributeKey: 'environment' },
      { key: 'owner', label: '系统负责人', placeholder: '王五', source: 'attribute', attributeKey: 'owner' }
    ]
  }
};

const HISTORY_LIMIT = 10;

const initialValues = (config: LedgerConfig) =>
  config.fields.reduce<Record<string, string>>((acc, field) => {
    acc[field.key] = '';
    return acc;
  }, {});

const normalizeRecords = (items: LedgerRecord[] = []) =>
  items.map((item) => ({
    ...item,
    attributes: item.attributes && typeof item.attributes === 'object' ? item.attributes : {},
    tags: Array.isArray(item.tags) ? item.tags : []
  }));

const getFieldValue = (record: LedgerRecord, field: FieldConfig): string => {
  switch (field.source) {
    case 'name':
      return record.name ?? '';
    case 'description':
      return record.description ?? '';
    case 'attribute':
    default:
      return record.attributes?.[field.attributeKey ?? field.key] ?? '';
  }
};

const buildPayload = (values: Record<string, string>, tags: string[], config: LedgerConfig) => {
  const payload: {
    name: string;
    description?: string;
    attributes?: Record<string, string>;
    tags: string[];
  } = {
    name: values[config.nameFieldKey]?.trim() ?? '',
    tags
  };

  if (config.descriptionFieldKey) {
    const descriptionValue = values[config.descriptionFieldKey]?.trim();
    if (descriptionValue) {
      payload.description = descriptionValue;
    }
  }

  config.fields.forEach((field) => {
    const value = values[field.key]?.trim();
    if (!value) {
      return;
    }
    if (field.source === 'name' && !payload.name) {
      payload.name = value;
      return;
    }
    if (field.source === 'description' && !payload.description) {
      payload.description = value;
      return;
    }
    if (field.source === 'attribute') {
      if (!payload.attributes) {
        payload.attributes = {};
      }
      payload.attributes[field.attributeKey ?? field.key] = value;
    }
  });

  return payload;
};

const readFileAsBase64 = (file: File) =>
  new Promise<string>((resolve, reject) => {
    const reader = new FileReader();
    reader.onload = () => {
      try {
        const result = reader.result;
        if (result instanceof ArrayBuffer) {
          const bytes = new Uint8Array(result);
          let binary = '';
          bytes.forEach((byte) => {
            binary += String.fromCharCode(byte);
          });
          resolve(btoa(binary));
          return;
        }
        if (typeof result === 'string') {
          const base64 = result.split(',').pop() ?? '';
          resolve(base64);
          return;
        }
        reject(new Error('无法读取文件内容'));
      } catch (error) {
        reject(error as Error);
      }
    };
    reader.onerror = () => {
      reject(new Error('读取文件失败，请重试。'));
    };
    reader.readAsArrayBuffer(file);
  });

const Assets = () => {
  const [activeType, setActiveType] = useState<LedgerType>('ips');
  const [records, setRecords] = useState<Record<LedgerType, LedgerRecord[]>>({
    ips: [],
    devices: [],
    personnel: [],
    systems: []
  });
  const [loading, setLoading] = useState(false);
  const [editingId, setEditingId] = useState<string | null>(null);
  const [formValues, setFormValues] = useState<Record<string, string>>(initialValues(LEDGER_CONFIGS.ips));
  const [formTags, setFormTags] = useState<string[]>([]);
  const [tagDraft, setTagDraft] = useState('');
  const [history, setHistory] = useState<HistoryCounters>({ undoSteps: 0, redoSteps: 0 });
  const [showPasteModal, setShowPasteModal] = useState(false);
  const [pasteText, setPasteText] = useState('');
  const [importing, setImporting] = useState(false);
  const [exporting, setExporting] = useState(false);
  const fileInputRef = useRef<HTMLInputElement | null>(null);

  const activeConfig = useMemo(() => LEDGER_CONFIGS[activeType], [activeType]);

  const refreshHistory = async () => {
    try {
      const { data } = await api.get(`${API_PREFIX}/history`);
      setHistory({
        undoSteps: typeof data?.undo === 'number' ? data.undo : 0,
        redoSteps: typeof data?.redo === 'number' ? data.redo : 0
      });
    } catch (error) {
      console.error('Failed to load history counters', error);
    }
  };

  const fetchLedger = async (type: LedgerType) => {
    const config = LEDGER_CONFIGS[type];
    try {
      if (type === activeType) {
        setLoading(true);
      }
      const { data } = await api.get(config.endpoint);
      const items: LedgerRecord[] = normalizeRecords(Array.isArray(data?.items) ? (data.items as LedgerRecord[]) : []);
      setRecords((prev) => ({ ...prev, [type]: items }));
    } catch (error) {
      console.error('Failed to load ledger', error);
    } finally {
      if (type === activeType) {
        setLoading(false);
      }
    }
    await refreshHistory();
  };

  useEffect(() => {
    fetchLedger(activeType);
  }, [activeType]);

  useEffect(() => {
    setFormValues(initialValues(activeConfig));
    setFormTags([]);
    setTagDraft('');
    setEditingId(null);
  }, [activeConfig]);

  const handleValueChange = (key: string, value: string) => {
    setFormValues((prev) => ({ ...prev, [key]: value }));
  };

  const handleTagKeyDown = (event: KeyboardEvent<HTMLInputElement>) => {
    if (event.key !== 'Enter' && event.key !== ',') return;
    event.preventDefault();
    const trimmed = tagDraft.trim();
    if (!trimmed) return;
    if (!formTags.includes(trimmed)) {
      setFormTags((prev) => [...prev, trimmed]);
    }
    setTagDraft('');
  };

  const handleTagRemove = (tag: string) => {
    setFormTags((prev) => prev.filter((item) => item !== tag));
  };

  const resetForm = () => {
    setFormValues(initialValues(activeConfig));
    setFormTags([]);
    setTagDraft('');
    setEditingId(null);
  };

  const handleEdit = (entry: LedgerRecord) => {
    const nextValues = initialValues(activeConfig);
    activeConfig.fields.forEach((field) => {
      nextValues[field.key] = getFieldValue(entry, field);
    });
    setFormValues(nextValues);
    setFormTags(Array.isArray(entry.tags) ? entry.tags : []);
    setEditingId(entry.id);
  };

  const handleDelete = async (id: string) => {
    const confirmed = window.confirm('确定删除这条台账记录吗？');
    if (!confirmed) return;
    try {
      await api.delete(`${activeConfig.endpoint}/${id}`);
      await fetchLedger(activeType);
      if (editingId === id) {
        resetForm();
      }
    } catch (error) {
      console.error('删除失败', error);
      window.alert('删除失败，请稍后重试。');
    }
  };

  const handleUndo = async () => {
    try {
      await api.post(`${API_PREFIX}/history/undo`);
      await fetchLedger(activeType);
    } catch (error) {
      console.error('回退失败', error);
      await refreshHistory();
      window.alert('暂时没有可回退的记录');
    }
  };

  const handleRedo = async () => {
    try {
      await api.post(`${API_PREFIX}/history/redo`);
      await fetchLedger(activeType);
    } catch (error) {
      console.error('前进失败', error);
      await refreshHistory();
      window.alert('暂时没有可前进的记录');
    }
  };

  const handleExportExcel = async () => {
    try {
      setExporting(true);
      const response = await api.get(`${API_PREFIX}/ledgers/export`, { responseType: 'blob' });
      const blob = new Blob([response.data], {
        type: 'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet'
      });
      const url = URL.createObjectURL(blob);
      const link = document.createElement('a');
      link.href = url;
      link.download = 'ledger.xlsx';
      document.body.appendChild(link);
      link.click();
      document.body.removeChild(link);
      URL.revokeObjectURL(url);
    } catch (error) {
      console.error('导出失败', error);
      window.alert('导出 Excel 失败，请稍后重试。');
    } finally {
      setExporting(false);
    }
  };

  const handleExcelFileChange = async (event: ChangeEvent<HTMLInputElement>) => {
    const file = event.target.files?.[0];
    if (!file) {
      return;
    }
    setImporting(true);
    try {
      const base64 = await readFileAsBase64(file);
      await api.post(`${API_PREFIX}/ledgers/import`, { data: base64 });
      await fetchLedger(activeType);
      window.alert('Excel 导入成功。');
    } catch (error) {
      console.error('导入失败', error);
      window.alert('导入 Excel 失败，请检查文件格式。');
    } finally {
      setImporting(false);
      if (fileInputRef.current) {
        fileInputRef.current.value = '';
      }
    }
  };

  const openPasteModal = () => {
    setPasteText('');
    setShowPasteModal(true);
  };

  const closePasteModal = () => {
    setShowPasteModal(false);
    setPasteText('');
  };

  const parseBulkRecords = (value: string) => {
    const rows = value
      .split(/\r?\n/)
      .map((line) => line.trim())
      .filter(Boolean);
    const keys = activeConfig.fields.map((field) => field.key);
    return rows.map((row) => {
      let parts = row.split('\t');
      if (parts.length === 1) {
        parts = row.split(',');
      }
      const payload: Record<string, string> = {};
      keys.forEach((key, index) => {
        payload[key] = parts[index]?.trim() ?? '';
      });
      return payload;
    });
  };

  const handlePasteImportSubmit = async (event: FormEvent) => {
    event.preventDefault();
    if (!pasteText.trim()) {
      closePasteModal();
      return;
    }
    const recordsToCreate = parseBulkRecords(pasteText);
    if (!recordsToCreate.length) {
      closePasteModal();
      return;
    }
    try {
      setImporting(true);
      for (const recordValues of recordsToCreate) {
        const hasRequired = activeConfig.fields.every((field) => {
          if (!field.required) return true;
          const value = recordValues[field.key];
          return typeof value === 'string' && value.trim().length > 0;
        });
        if (!hasRequired) {
          throw new Error('存在缺少必填字段的记录');
        }
        const payload = buildPayload(recordValues, [], activeConfig);
        await api.post(activeConfig.endpoint, payload);
      }
      await fetchLedger(activeType);
      window.alert('批量粘贴导入完成。');
      closePasteModal();
    } catch (error) {
      console.error('批量导入失败', error);
      const axiosError = error as AxiosError<{ error?: string }>;
      const message =
        axiosError.response?.data?.error || axiosError.message || '批量导入失败，请确认内容格式与字段数量匹配。';
      window.alert(message);
    } finally {
      setImporting(false);
    }
  };

  const handleSubmit = async (event: FormEvent) => {
    event.preventDefault();
    const hasRequired = activeConfig.fields.every((field) => {
      if (!field.required) return true;
      const value = formValues[field.key];
      return typeof value === 'string' && value.trim().length > 0;
    });
    if (!hasRequired) {
      window.alert('请填写必填字段');
      return;
    }
    const payload = buildPayload(formValues, formTags, activeConfig);
    try {
      if (editingId) {
        await api.put(`${activeConfig.endpoint}/${editingId}`, payload);
      } else {
        await api.post(activeConfig.endpoint, payload);
      }
      await fetchLedger(activeType);
      resetForm();
    } catch (error) {
      console.error('保存失败', error);
      window.alert('保存失败，请稍后重试。');
    }
  };

  const reorderEntries = async (index: number, direction: -1 | 1) => {
    const items = records[activeType];
    const targetIndex = index + direction;
    if (targetIndex < 0 || targetIndex >= items.length) return;
    const reordered = [...items];
    const [moved] = reordered.splice(index, 1);
    reordered.splice(targetIndex, 0, moved);
    setRecords((prev) => ({ ...prev, [activeType]: reordered }));
    try {
      const order = reordered.map((item) => item.id);
      await api.post(`${activeConfig.endpoint}/reorder`, { ids: order });
      await fetchLedger(activeType);
    } catch (error) {
      console.error('排序失败', error);
      await fetchLedger(activeType);
    }
  };

  const currentRecords = records[activeType];
  const undoDisabled = history.undoSteps === 0;
  const redoDisabled = history.redoSteps === 0;

  return (
    <div className="space-y-6">
      <input
        ref={fileInputRef}
        type="file"
        accept=".xlsx,application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
        className="sr-only"
        onChange={handleExcelFileChange}
      />

      <div className="flex flex-wrap items-center justify-between gap-4">
        <div className="max-w-3xl">
          <h2 className="section-title">台账编排</h2>
          <p className="mt-1 text-sm leading-relaxed text-night-300 break-words">
            参考 Eidos 的霓虹层次，将 IP、设备、人员、系统四大维度统筹管理，可添加标签并自由排序。
          </p>
        </div>
        <div className="flex flex-col items-stretch gap-3 sm:flex-row sm:items-center">
          <div className="flex gap-2 rounded-full bg-night-900/70 p-1">
            {(Object.values(LEDGER_CONFIGS) as LedgerConfig[]).map((config) => (
              <button
                key={config.type}
                onClick={() => setActiveType(config.type)}
                className={`rounded-full px-4 py-2 text-sm transition-all ${
                  activeType === config.type
                    ? 'glass-panel border-neon-500/50 text-neon-500 shadow-glow'
                    : 'text-night-300 hover:text-neon-500'
                }`}
              >
                {config.label}
              </button>
            ))}
          </div>
          <div className="flex flex-wrap items-center gap-2">
            <button
              onClick={handleUndo}
              disabled={undoDisabled}
              title={`剩余 ${history.undoSteps} 步可回退`}
              className="glass-panel inline-flex min-w-[88px] items-center gap-2 rounded-full px-3 py-2 text-xs font-medium text-night-200 transition-colors hover:text-neon-500 disabled:cursor-not-allowed disabled:opacity-40"
            >
              <ArrowUturnLeftIcon className="h-4 w-4 shrink-0" />
              <span className="truncate">回退</span>
            </button>
            <button
              onClick={handleRedo}
              disabled={redoDisabled}
              title={`剩余 ${history.redoSteps} 步可前进`}
              className="glass-panel inline-flex min-w-[88px] items-center gap-2 rounded-full px-3 py-2 text-xs font-medium text-night-200 transition-colors hover:text-neon-500 disabled:cursor-not-allowed disabled:opacity-40"
            >
              <ArrowUturnRightIcon className="h-4 w-4 shrink-0" />
              <span className="truncate">前进</span>
            </button>
            <span className="text-[11px] text-night-500 whitespace-nowrap">
              回退 {history.undoSteps}/{HISTORY_LIMIT} · 前进 {history.redoSteps}/{HISTORY_LIMIT}
            </span>
          </div>
        </div>
      </div>

      <div className="flex flex-wrap items-center justify-end gap-2">
        <button
          type="button"
          onClick={() => fileInputRef.current?.click()}
          disabled={importing}
          className="glass-panel inline-flex items-center gap-2 rounded-full px-4 py-2 text-xs font-medium text-night-200 transition hover:text-neon-500 disabled:cursor-not-allowed disabled:opacity-50"
        >
          <ClipboardDocumentCheckIcon className="h-4 w-4" />
          {importing ? '处理中…' : '导入 Excel'}
        </button>
        <button
          type="button"
          onClick={handleExportExcel}
          disabled={exporting}
          className="glass-panel inline-flex items-center gap-2 rounded-full px-4 py-2 text-xs font-medium text-night-200 transition hover:text-neon-500 disabled:cursor-not-allowed disabled:opacity-50"
        >
          <ArrowDownTrayIcon className="h-4 w-4" />
          {exporting ? '导出中…' : '导出 Excel'}
        </button>
        <button
          type="button"
          onClick={openPasteModal}
          className="glass-panel inline-flex items-center gap-2 rounded-full px-4 py-2 text-xs font-medium text-night-200 transition hover:text-neon-500"
        >
          <PlusIcon className="h-4 w-4" />
          批量粘贴导入
        </button>
      </div>

      <div className="grid gap-6 lg:grid-cols-[2fr_1fr]">
        <div className="space-y-4">
          {loading ? (
            <div className="glass-panel rounded-3xl p-6 text-center text-night-300 break-words">正在加载 {activeConfig.label}...</div>
          ) : currentRecords.length === 0 ? (
            <div className="glass-panel rounded-3xl p-6 text-center text-night-300 break-words">
              暂无 {activeConfig.label} 记录，右侧可手动创建。
            </div>
          ) : (
            currentRecords.map((entry, index) => (
              <div
                key={entry.id}
                className={`glass-panel rounded-3xl border border-ink-200/80 p-5 transition-all hover:border-neon-500/30 overflow-hidden ${
                  editingId === entry.id ? 'ring-2 ring-neon-400/60' : ''
                }`}
              >
                <div className="flex flex-col gap-4 md:flex-row md:items-start md:justify-between">
                  <div className="min-w-0 space-y-3">
                    <h3 className={`text-base font-semibold text-night-100 ${activeConfig.accent} break-all`}>
                      {getFieldValue(entry, activeConfig.fields.find((field) => field.key === activeConfig.nameFieldKey) ?? activeConfig.fields[0]) || '未命名'}
                    </h3>
                    <div className="space-y-2 text-sm text-night-300 break-words">
                      {activeConfig.fields
                        .filter((field) => field.key !== activeConfig.nameFieldKey)
                        .map((field) => {
                          const value = getFieldValue(entry, field);
                          if (!value) return null;
                          return (
                            <p key={field.key} className="flex flex-wrap items-center gap-2 text-left break-words">
                              <span className="text-night-500 whitespace-nowrap">{field.label}：</span>
                              <span className="break-all text-night-100">{value}</span>
                            </p>
                          );
                        })}
                    </div>
                    {entry.tags && entry.tags.length > 0 && (
                      <div className="flex flex-wrap gap-2 pt-1">
                        {entry.tags.map((tag) => (
                          <span
                            key={tag}
                            className="inline-flex max-w-full flex-wrap items-center gap-1 rounded-full border border-night-700/70 bg-white px-3 py-1 text-xs text-neon-500"
                            title={tag}
                          >
                            <TagIcon className="h-3 w-3 shrink-0" />
                            <span className="break-all">{tag}</span>
                          </span>
                        ))}
                      </div>
                    )}
                  </div>
                  <div className="flex shrink-0 flex-row items-center gap-2 text-night-400 md:flex-col md:items-end">
                    <button
                      onClick={() => handleEdit(entry)}
                      className="glass-panel rounded-full p-2 transition-colors hover:text-neon-500"
                      aria-label="编辑"
                    >
                      <PencilSquareIcon className="h-5 w-5" />
                    </button>
                    <button
                      onClick={() => handleDelete(entry.id)}
                      className="glass-panel rounded-full p-2 transition-colors hover:text-red-400"
                      aria-label="删除"
                    >
                      <TrashIcon className="h-5 w-5" />
                    </button>
                    <div className="mt-2 flex flex-col items-center gap-1">
                      <button
                        onClick={() => reorderEntries(index, -1)}
                        disabled={index === 0}
                        className="glass-panel rounded-full p-1 disabled:cursor-not-allowed disabled:opacity-40"
                        aria-label="上移"
                      >
                        <ArrowUpIcon className="h-4 w-4" />
                      </button>
                      <button
                        onClick={() => reorderEntries(index, 1)}
                        disabled={index === currentRecords.length - 1}
                        className="glass-panel rounded-full p-1 disabled:cursor-not-allowed disabled:opacity-40"
                        aria-label="下移"
                      >
                        <ArrowDownIcon className="h-4 w-4" />
                      </button>
                    </div>
                  </div>
                </div>
              </div>
            ))
          )}
        </div>

        <div className="glass-panel rounded-3xl border border-ink-200/80 p-6 overflow-hidden">
          <div className="flex flex-wrap items-center justify-between gap-3">
            <h3 className="text-lg font-semibold text-night-100 break-words">
              {editingId ? '编辑台账' : '新增台账'} · {activeConfig.label}
            </h3>
            {editingId && (
              <button onClick={resetForm} className="text-xs text-night-400 hover:text-neon-500 whitespace-nowrap">
                取消编辑
              </button>
            )}
          </div>

          <form className="mt-6 space-y-4" onSubmit={handleSubmit}>
            {activeConfig.fields.map((field) => (
              <div key={field.key}>
                <label className="block text-xs uppercase tracking-[0.18em] text-night-400 break-words">
                  {field.label}
                  {field.required && <span className="text-red-400"> *</span>}
                </label>
                <input
                  value={formValues[field.key] ?? ''}
                  onChange={(event) => handleValueChange(field.key, event.target.value)}
                  placeholder={field.placeholder}
                  className="mt-2 w-full rounded-2xl border border-ink-200 bg-white px-4 py-3 text-sm text-night-100 placeholder-night-400 focus:border-neon-500 focus:outline-none focus:ring-2 focus:ring-neon-500/30"
                />
              </div>
            ))}

            <div>
              <label className="block text-xs uppercase tracking-[0.18em] text-night-400">标签</label>
              <div className="mt-2 flex flex-wrap gap-2 rounded-2xl border border-ink-200 bg-white p-3">
                {formTags.map((tag) => (
                  <button
                    key={tag}
                    type="button"
                    onClick={() => handleTagRemove(tag)}
                    className="inline-flex max-w-full flex-wrap items-center gap-1 rounded-full border border-night-700/70 px-3 py-1 text-xs text-neon-500 hover:border-neon-500/60"
                    title={`点击移除 ${tag}`}
                  >
                    <span className="break-all">{tag}</span>
                    <span className="text-night-500">×</span>
                  </button>
                ))}
                <input
                  value={tagDraft}
                  onChange={(event) => setTagDraft(event.target.value)}
                  onKeyDown={handleTagKeyDown}
                  placeholder="输入后回车添加标签"
                  className="flex-1 min-w-[120px] bg-transparent text-sm text-night-200 placeholder-night-600 focus:outline-none"
                />
              </div>
              <p className="mt-1 text-xs text-night-500 break-words">
                为每条记录附加 Eidos 风格的标签，如“核心”、“外包”、“高风险”等，用于多维筛选。
              </p>
            </div>

            <button type="submit" className="button-primary flex w-full items-center justify-center gap-2 whitespace-nowrap">
              <PlusIcon className="h-4 w-4" />
              {editingId ? '保存更改' : '创建记录'}
            </button>
          </form>
        </div>
      </div>

      {showPasteModal && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-night-950/70 px-4">
          <div className="w-full max-w-2xl space-y-5 rounded-3xl bg-white p-6 text-night-900 shadow-2xl">
            <div className="space-y-1">
              <h3 className="text-lg font-semibold">批量粘贴导入</h3>
              <p className="text-sm text-night-500">
                每行代表一条记录，可使用制表符或逗号分隔字段，顺序需与当前表单字段一致。
              </p>
            </div>
            <form className="space-y-4" onSubmit={handlePasteImportSubmit}>
              <textarea
                value={pasteText}
                onChange={(event) => setPasteText(event.target.value)}
                rows={8}
                placeholder="示例：\n10.0.0.24/24\t办公区访问\n10.0.0.25/24\t内测网络"
                className="w-full rounded-2xl border border-night-300/60 bg-white px-4 py-3 text-sm text-night-900 placeholder-night-400 focus:border-neon-500 focus:outline-none focus:ring-2 focus:ring-neon-500/30"
              />
              <div className="flex justify-end gap-3">
                <button
                  type="button"
                  onClick={closePasteModal}
                  className="rounded-full border border-night-300 px-5 py-2 text-sm text-night-500 hover:border-night-400 hover:text-night-700"
                  disabled={importing}
                >
                  取消
                </button>
                <button
                  type="submit"
                  className="button-primary inline-flex items-center gap-2"
                  disabled={importing}
                >
                  <PlusIcon className="h-4 w-4" />
                  {importing ? '导入中…' : '确认导入'}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  );
};

export default Assets;
