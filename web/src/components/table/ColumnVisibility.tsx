/* eslint-disable @typescript-eslint/no-unused-vars, no-unused-vars */
import { useMemo } from 'react';
import type { Property, ViewColumn } from '../../types/roledger';

type ColumnVisibilityProps = {
  properties: Property[];
  columns: ViewColumn[];
  onToggle: (propertyId: string) => void;
};

export const ColumnVisibility = ({ properties, columns, onToggle }: ColumnVisibilityProps) => {
  const merged = useMemo(() => {
    return properties.map((prop) => {
      const col = columns.find((c) => c.propertyId === prop.id);
      return { ...prop, visible: col ? col.visible : false };
    });
  }, [properties, columns]);

  return (
    <div className="rounded-lg border border-[var(--c-line)] bg-white p-3 text-sm text-[var(--c-text)]">
      <p className="mb-2 text-xs uppercase tracking-[0.2em] text-[var(--c-muted)]">显示列</p>
      <div className="space-y-2">
        {merged.map((item) => (
          <label key={item.id} className="flex items-center justify-between gap-2">
            <span>{item.name}</span>
            <input
              type="checkbox"
              checked={item.visible}
              onChange={() => onToggle(item.id)}
              className="h-4 w-4 rounded border-[var(--c-line)] text-[var(--c-accent)]"
            />
          </label>
        ))}
      </div>
    </div>
  );
};
