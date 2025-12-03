import { ArrowDownTrayIcon, ArrowUpTrayIcon, ClipboardDocumentListIcon, TrashIcon } from '@heroicons/react/24/outline';
import { clsx } from 'clsx';

interface ToolbarActionsProps {
  onSave: () => void;
  onDelete: () => void;
  onExcelImport?: () => void;
  onPasteImport?: () => void;
  onExport?: () => void;
  onExportAll?: () => void;
  onImportAll?: () => void;
  onDocumentImport?: () => void;
  onDocumentExport?: () => void;
  sheetActionsEnabled?: boolean;
  documentActionsEnabled?: boolean;
  busy?: boolean;
  dirty?: boolean;
}

const ActionButton = ({
  icon: Icon,
  label,
  onClick,
  disabled
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
    className={clsx(
      'roleger-btn roleger-btn--ghost text-xs font-medium',
      disabled && 'opacity-60'
    )}
  >
    <Icon className="h-4 w-4" />
    <span>{label}</span>
  </button>
);

export const ToolbarActions = ({
  onSave,
  onDelete,
  onExcelImport,
  onPasteImport,
  onExport,
  onExportAll,
  onImportAll,
  onDocumentImport,
  onDocumentExport,
  sheetActionsEnabled = true,
  documentActionsEnabled = false,
  busy = false,
  dirty = false
}: ToolbarActionsProps) => {
  return (
    <div className="flex flex-wrap items-center gap-2">
      {sheetActionsEnabled && (
        <>
          {onExcelImport && (
            <ActionButton icon={ArrowUpTrayIcon} label="Excel 导入" onClick={onExcelImport} />
          )}
          {onPasteImport && (
            <ActionButton icon={ClipboardDocumentListIcon} label="粘贴导入" onClick={onPasteImport} />
          )}
          {onExport && (
            <ActionButton icon={ArrowDownTrayIcon} label="导出 Excel" onClick={onExport} />
          )}
        </>
      )}
      {documentActionsEnabled && (
        <>
          {onDocumentImport && (
            <ActionButton icon={ArrowUpTrayIcon} label="导入文档" onClick={onDocumentImport} />
          )}
          {onDocumentExport && (
            <ActionButton icon={ArrowDownTrayIcon} label="导出文档" onClick={onDocumentExport} />
          )}
        </>
      )}
      {onExportAll && <ActionButton icon={ArrowDownTrayIcon} label="导出全部数据" onClick={onExportAll} />}
      {onImportAll && (
        <button type="button" onClick={onImportAll} className="roleger-btn roleger-btn--ghost text-xs font-medium">
          <ArrowUpTrayIcon className="h-4 w-4" />
          <span>导入全部数据</span>
        </button>
      )}
      <button
        type="button"
        onClick={onSave}
        disabled={busy}
        className={clsx('roleger-btn', busy && 'is-busy')}
      >
        {busy ? '保存中…' : dirty ? '保存更改' : '保存'}
      </button>
      <button
        type="button"
        onClick={onDelete}
        className="roleger-btn roleger-btn--danger"
      >
        <TrashIcon className="h-4 w-4" />
        删除内容
      </button>
    </div>
  );
};
