import { ArrowDownTrayIcon, ArrowUpTrayIcon, ClipboardDocumentListIcon, TrashIcon } from '@heroicons/react/24/outline';
import { clsx } from 'clsx';

interface ToolbarActionsProps {
  onSave: () => void;
  onDelete: () => void;
  onExcelImport: () => void;
  onPasteImport: () => void;
  onExport: () => void;
  disabledSheetActions?: boolean;
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
      'eidos-btn eidos-btn--ghost text-xs font-medium',
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
  disabledSheetActions = false,
  busy = false,
  dirty = false
}: ToolbarActionsProps) => {
  return (
    <div className="flex flex-wrap items-center gap-2">
      <ActionButton icon={ArrowUpTrayIcon} label="Excel 导入" onClick={onExcelImport} disabled={disabledSheetActions} />
      <ActionButton icon={ClipboardDocumentListIcon} label="粘贴导入" onClick={onPasteImport} disabled={disabledSheetActions} />
      <ActionButton icon={ArrowDownTrayIcon} label="导出 Excel" onClick={onExport} disabled={disabledSheetActions} />
      <button
        type="button"
        onClick={onSave}
        disabled={busy}
        className={clsx('eidos-btn', busy && 'is-busy')}
      >
        {busy ? '保存中…' : dirty ? '保存更改' : '保存'}
      </button>
      <button
        type="button"
        onClick={onDelete}
        className="eidos-btn eidos-btn--danger"
      >
        <TrashIcon className="h-4 w-4" />
        删除内容
      </button>
    </div>
  );
};
