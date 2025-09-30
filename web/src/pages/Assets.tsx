import { FormEvent, KeyboardEvent, useEffect, useMemo, useState } from 'react';
import api from '../api/client';
import {
  ArrowDownIcon,
  ArrowUpIcon,
  ArrowUturnLeftIcon,
  ArrowUturnRightIcon,
  PencilSquareIcon,
  PlusIcon,
  TagIcon,
  TrashIcon
} from '@heroicons/react/24/outline';

type LedgerType = 'ips' | 'devices' | 'personnel' | 'systems';

interface LedgerRecord {
  id: number;
  tags?: string[];
  [key: string]: unknown;
}

interface HistoryCounters {
  undoSteps: number;
  redoSteps: number;
}

type LedgerStateResponse = Partial<Record<LedgerType, LedgerRecord[]>>;

interface FieldConfig {
  key: string;
  label: string;
  placeholder: string;
  required?: boolean;
}

interface LedgerConfig {
  type: LedgerType;
  label: string;
  endpoint: string;
  responseKey: string;
  fields: FieldConfig[];
  accent: string;
}

const LEDGER_CONFIGS: Record<LedgerType, LedgerConfig> = {
  ips: {
    type: 'ips',
    label: 'IP 白名单',
    endpoint: '/ledger/ips',
    responseKey: 'ips',
    accent: 'text-neon-500',
    fields: [
      { key: 'address', label: 'IP 地址', placeholder: '10.0.0.12/24', required: true },
      { key: 'description', label: '说明', placeholder: '办公网络出入口' }
    ]
  },
  devices: {
    type: 'devices',
    label: '终端设备',
    endpoint: '/ledger/devices',
    responseKey: 'devices',
    accent: 'text-indigo-300',
    fields: [
      { key: 'identifier', label: '设备标识', placeholder: 'MacBook Pro SN', required: true },
      { key: 'type', label: '类型', placeholder: 'Laptop' },
      { key: 'owner', label: '责任人', placeholder: '张三' }
    ]
  },
  personnel: {
    type: 'personnel',
    label: '人员',
    endpoint: '/ledger/personnel',
    responseKey: 'personnel',
    accent: 'text-amber-300',
    fields: [
      { key: 'name', label: '姓名', placeholder: '李四', required: true },
      { key: 'role', label: '角色', placeholder: '安全负责人' },
      { key: 'contact', label: '联系方式', placeholder: 'lisa@example.com' }
    ]
  },
  systems: {
    type: 'systems',
    label: '系统',
    endpoint: '/ledger/systems',
    responseKey: 'systems',
    accent: 'text-cyan-300',
    fields: [
      { key: 'name', label: '系统名称', placeholder: '核心交易平台', required: true },
      { key: 'environment', label: '环境', placeholder: '生产/预发' },
      { key: 'owner', label: '系统负责人', placeholder: '王五' }
    ]
  }
};

const initialValues = (config: LedgerConfig) =>
  config.fields.reduce<Record<string, string>>((acc, field) => {
    acc[field.key] = '';
    return acc;
  }, {});

const HISTORY_LIMIT = 10;

const normalizeRecords = (items: LedgerRecord[] = []) =>
  items.map((item) => ({ ...item, tags: Array.isArray(item.tags) ? item.tags : [] }));

const Assets = () => {
  const [activeType, setActiveType] = useState<LedgerType>('ips');
  const [records, setRecords] = useState<Record<LedgerType, LedgerRecord[]>>({
    ips: [],
    devices: [],
    personnel: [],
    systems: []
  });
  const [loading, setLoading] = useState(false);
  const [editingId, setEditingId] = useState<number | null>(null);
  const [formValues, setFormValues] = useState<Record<string, string>>(initialValues(LEDGER_CONFIGS.ips));
  const [formTags, setFormTags] = useState<string[]>([]);
  const [tagDraft, setTagDraft] = useState('');
  const [history, setHistory] = useState<HistoryCounters>({ undoSteps: 0, redoSteps: 0 });

  const activeConfig = useMemo(() => LEDGER_CONFIGS[activeType], [activeType]);

  const refreshHistory = async () => {
    try {
      const { data } = await api.get('/ledger/history');
      const counters = data?.history ?? {};
      setHistory({
        undoSteps: typeof counters.undoSteps === 'number' ? counters.undoSteps : 0,
        redoSteps: typeof counters.redoSteps === 'number' ? counters.redoSteps : 0
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
      const responseKey = config.responseKey;
      const items: LedgerRecord[] = normalizeRecords(data[responseKey] ?? []);
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

  const syncStateFromPayload = (state?: LedgerStateResponse, counters?: HistoryCounters) => {
    if (state) {
      const nextState: Record<LedgerType, LedgerRecord[]> = {
        ips: normalizeRecords(state.ips ?? []),
        devices: normalizeRecords(state.devices ?? []),
        personnel: normalizeRecords(state.personnel ?? []),
        systems: normalizeRecords(state.systems ?? [])
      };
      setRecords(nextState);
      if (editingId !== null) {
        const activeList = nextState[activeType];
        if (!activeList.some((item) => item.id === editingId)) {
          resetForm();
        }
      }
    }
    if (counters) {
      setHistory(counters);
    }
  };

  const handleEdit = (entry: LedgerRecord) => {
    const nextValues = initialValues(activeConfig);
    activeConfig.fields.forEach((field) => {
      const value = entry[field.key];
      if (typeof value === 'string') {
        nextValues[field.key] = value;
      }
    });
    setFormValues(nextValues);
    setFormTags(Array.isArray(entry.tags) ? entry.tags : []);
    setEditingId(entry.id);
  };

  const handleDelete = async (id: number) => {
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
    }
  };

  const handleUndo = async () => {
    try {
      const { data } = await api.post('/ledger/history/undo');
      if (data?.state) {
        syncStateFromPayload(data.state, data.history);
      } else {
        await refreshHistory();
      }
    } catch (error) {
      console.error('回退失败', error);
      await refreshHistory();
      window.alert('暂时没有可回退的记录');
    }
  };

  const handleRedo = async () => {
    try {
      const { data } = await api.post('/ledger/history/redo');
      if (data?.state) {
        syncStateFromPayload(data.state, data.history);
      } else {
        await refreshHistory();
      }
    } catch (error) {
      console.error('前进失败', error);
      await refreshHistory();
      window.alert('暂时没有可前进的记录');
    }
  };

  const handleSubmit = async (event: FormEvent) => {
    event.preventDefault();
    const payload = { ...formValues, tags: formTags };
    const hasRequired = activeConfig.fields.every((field) => {
      if (!field.required) return true;
      const value = (payload as Record<string, unknown>)[field.key];
      if (typeof value === 'string') {
        return value.trim().length > 0;
      }
      return Boolean(value);
    });
    if (!hasRequired) {
      alert('请填写必填字段');
      return;
    }
    try {
      if (editingId !== null) {
        await api.put(`${activeConfig.endpoint}/${editingId}`, payload);
      } else {
        await api.post(activeConfig.endpoint, payload);
      }
      await fetchLedger(activeType);
      resetForm();
    } catch (error) {
      console.error('保存失败', error);
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
      await api.put(`${activeConfig.endpoint}/order`, { order });
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
      <div className="flex flex-wrap items-center justify-between gap-4">
        <div className="max-w-3xl">
          <h2 className="section-title">台账编排</h2>
          <p className="mt-1 text-sm leading-relaxed text-night-300 break-words">
            参考 Eidos 的霓虹层次，将 IP、设备、人员、系统四大维度统筹管理，可添加标签并自由排序。
          </p>
        </div>
        <div className="flex flex-wrap items-center gap-3">
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
                      {activeConfig.fields[0] && typeof entry[activeConfig.fields[0].key] === 'string'
                        ? (entry[activeConfig.fields[0].key] as string)
                        : '未命名'}
                    </h3>
                    <div className="space-y-2 text-sm text-night-300 break-words">
                      {activeConfig.fields.slice(1).map((field) => {
                        const value = entry[field.key];
                        if (!value) return null;
                        return (
                          <p key={field.key} className="flex flex-wrap items-center gap-2 text-left break-words">
                            <span className="text-night-500 whitespace-nowrap">{field.label}：</span>
                            <span className="break-all text-night-100">{String(value)}</span>
                          </p>
                        );
                      })}
                    </div>
                    {entry.tags && entry.tags.length > 0 && (
                      <div className="mt-4 flex flex-wrap gap-2">
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
    </div>
  );
};

export default Assets;
