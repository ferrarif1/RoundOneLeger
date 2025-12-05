/* eslint-disable @typescript-eslint/no-unused-vars, no-unused-vars */
import { useEffect, useState } from 'react';
import { ArrowUpTrayIcon, XMarkIcon } from '@heroicons/react/24/outline';
import { createTable, type ImportTask } from '../../api/roledger';

type ImportDialogProps = {
  open: boolean;
  onClose: () => void;
  onUploaded: (tableId: string) => void;
};

export const ImportDialog = ({ open, onClose, onUploaded }: ImportDialogProps) => {
  const [tableName, setTableName] = useState('');
  const [file, setFile] = useState<File | null>(null);
  const [status, setStatus] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);
  const [uploading, setUploading] = useState(false);
  const [taskId, setTaskId] = useState<string | null>(null);
  const [progress, setProgress] = useState<number>(0);
  const [tableId, setTableId] = useState<string | null>(null);

  useEffect(() => {
    if (!taskId) return;
    const timer = setInterval(async () => {
      try {
        const task = await import('../../api/roledger').then((m) => m.getImportTask(taskId));
        setProgress(task.progress || 0);
        if (task.status === 'success') {
          setStatus('导入完成');
          clearInterval(timer);
          if (tableId) {
            onUploaded(tableId);
          }
        } else if (task.status === 'failed') {
          setStatus(task.error || '导入失败');
          clearInterval(timer);
        } else {
          setStatus(`正在导入... ${task.progress}%`);
        }
      } catch (err: any) {
        setStatus(err?.message || '查询任务失败');
      }
    }, 1500);
    return () => clearInterval(timer);
  }, [taskId, tableId, onUploaded]);

  if (!open) return null;

  const handleSubmit = async () => {
    if (!file || !tableName.trim()) {
      setStatus('请选择文件并输入表名');
      return;
    }
    setBusy(true);
    setUploading(true);
    setStatus(null);
    try {
      // 先创建表，再上传任务（此处仅创建表，上传留给后续异步 API）
      const table = await createTable({ name: tableName.trim() });
      setTableId(table.id);
      setStatus('表已创建，开始导入...');
      const form = new FormData();
      form.append('tableName', table.name || tableName.trim());
      form.append('file', file);
      const { taskId: createdTaskId } = await import('../../api/roledger').then((m) => m.startImport(form));
      setTaskId(createdTaskId);
      setStatus('任务已提交，正在后台导入...');
    } catch (err: any) {
      setStatus(err?.message || '导入失败');
    } finally {
      setBusy(false);
    }
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-[var(--c-overlay)] px-4">
      <div className="w-full max-w-lg rounded-lg border border-[var(--c-line)] bg-white p-4 shadow-[var(--shadow-popover)]">
        <div className="mb-3 flex items-center justify-between">
          <h3 className="text-base font-semibold text-[var(--c-text)]">导入数据</h3>
          <button type="button" onClick={onClose} className="rounded-full border border-[var(--c-line)] p-1">
            <XMarkIcon className="h-4 w-4" />
          </button>
        </div>
        <div className="space-y-2">
          <input
            className="input"
            placeholder="新表名称"
            value={tableName}
            onChange={(e) => setTableName(e.target.value)}
          />
          <label className="flex items-center justify-between rounded-lg border border-[var(--c-line)] bg-white px-3 py-2 text-sm">
            <span>{file ? file.name : '选择 CSV/Excel 文件'}</span>
            <ArrowUpTrayIcon className="h-5 w-5 text-[var(--c-muted)]" />
            <input
              type="file"
              accept=".csv,application/vnd.openxmlformats-officedocument.spreadsheetml.sheet,application/vnd.ms-excel"
              className="hidden"
              onChange={(e) => setFile(e.target.files?.[0] || null)}
            />
          </label>
          <button type="button" className="btn" disabled={busy} onClick={handleSubmit}>
            {busy ? '上传中...' : '开始导入'}
          </button>
          {status && <p className="text-sm text-[var(--c-muted)]">{status}</p>}
          {taskId && (
            <div className="rounded-lg border border-[var(--c-line)] bg-white px-3 py-2 text-sm">
              任务 {taskId} 进度：{progress}%
            </div>
          )}
        </div>
      </div>
    </div>
  );
};
