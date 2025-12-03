/* eslint-disable @typescript-eslint/no-unused-vars, no-unused-vars */
import { FunnelIcon, Squares2X2Icon, SwatchIcon, PlusIcon } from '@heroicons/react/24/outline';
import type { View } from '../../types/roleger';

type ViewToolbarProps = {
  views: View[];
  activeViewId?: string;
  onSwitch: (viewId: string) => void;
  onFilter?: () => void;
  onLayoutChange?: () => void;
  onCreateView?: () => void;
};

export const ViewToolbar = ({ views, activeViewId, onSwitch, onFilter, onLayoutChange, onCreateView }: ViewToolbarProps) => {
  return (
    <div className="mb-3 flex flex-wrap items-center gap-2 text-sm">
      <div className="flex items-center gap-1 rounded-full border border-[var(--c-line)] bg-white px-2 py-1">
        {views.map((view) => (
          <button
            key={view.id}
            type="button"
            onClick={() => onSwitch(view.id)}
            className={`rounded-full px-3 py-1 text-sm ${
              view.id === activeViewId ? 'bg-[var(--c-hover)] text-[var(--c-text)]' : 'text-[var(--c-muted)]'
            }`}
          >
            {view.name}
          </button>
        ))}
      </div>
      <button
        type="button"
          onClick={onFilter}
          className="flex items-center gap-1 rounded-full border border-[var(--c-line)] bg-white px-3 py-1 text-[var(--c-text)]"
        >
          <FunnelIcon className="h-4 w-4 text-[var(--c-muted)]" />
          筛选
        </button>
      <button
        type="button"
        onClick={onLayoutChange}
        className="flex items-center gap-1 rounded-full border border-[var(--c-line)] bg-white px-3 py-1 text-[var(--c-text)]"
      >
        <Squares2X2Icon className="h-4 w-4 text-[var(--c-muted)]" />
        视图
      </button>
      <button
        type="button"
        onClick={onCreateView}
        className="flex items-center gap-1 rounded-full border border-[var(--c-line)] bg-white px-3 py-1 text-[var(--c-text)]"
      >
        <PlusIcon className="h-4 w-4 text-[var(--c-muted)]" />
        新建视图
      </button>
      <div className="ml-auto flex items-center gap-1 rounded-full border border-[var(--c-line)] bg-white px-3 py-1 text-[var(--c-muted)]">
        <SwatchIcon className="h-4 w-4" />
        轻量模式
      </div>
    </div>
  );
};
