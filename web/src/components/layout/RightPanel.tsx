/* eslint-disable @typescript-eslint/no-unused-vars, no-unused-vars */
import React, { type JSX } from 'react';
import { XMarkIcon } from '@heroicons/react/24/outline';

type RightPanelProps = {
  title?: string;
  properties: { id: string; name: string; type: string; options?: Array<{ id?: string; label: string }> }[];
  values?: Record<string, unknown>;
  onChange: (id: string, value: unknown) => void;
  onClose?: () => void;
};

export const RightPanel = ({ title = '属性', properties, values, onChange, onClose }: RightPanelProps) => (
  <aside className="w-80 shrink-0 border-l border-[var(--c-line)] bg-white px-4 py-4">
    <div className="mb-3 flex items-center justify-between">
      <div>
        <p className="text-xs uppercase tracking-[0.2em] text-[var(--c-muted)]">属性</p>
        <h3 className="text-sm font-semibold text-[var(--c-text)]">{title}</h3>
      </div>
      {onClose && (
        <button
          type="button"
          aria-label="关闭属性面板"
          className="rounded-full border border-[var(--c-line)] bg-white p-1 text-[var(--c-muted)]"
          onClick={onClose}
        >
          <XMarkIcon className="h-4 w-4" />
        </button>
      )}
    </div>
    <div className="space-y-3">
      {properties.map((prop) => {
        const value = values?.[prop.id];
        const type = prop.type;
        const opts = prop.options || [];

        let input: JSX.Element;
        if (type === 'select') {
          input = (
            <select
              className="select"
              value={(value as string) ?? ''}
              onChange={(e) => onChange(prop.id, e.target.value)}
            >
              <option value="">请选择</option>
              {Array.isArray(opts) &&
                opts.map((opt) => (
                  <option key={opt.id || opt.label} value={opt.id || opt.label}>
                    {opt.label}
                  </option>
                ))}
            </select>
          );
        } else if (type === 'checkbox') {
          input = (
            <label className="inline-flex items-center gap-2 text-sm">
              <input
                type="checkbox"
                checked={Boolean(value)}
                onChange={(e) => onChange(prop.id, e.target.checked)}
                className="h-4 w-4 rounded border-[var(--c-line)] text-[var(--c-accent)]"
              />
              <span>是/否</span>
            </label>
          );
        } else if (type === 'number') {
          input = (
            <input
              type="number"
              className="input"
              value={value === undefined || value === null ? '' : (value as number)}
              onChange={(e) => onChange(prop.id, e.target.valueAsNumber)}
            />
          );
        } else if (type === 'multi_select') {
          const selected = Array.isArray(value) ? (value as string[]) : [];
          input = (
            <div className="space-y-1 rounded-[6px] border border-[var(--c-line)] bg-white px-2 py-2">
              {Array.isArray(opts) &&
                opts.map((opt) => {
                  const checked = selected.includes(opt.id || opt.label);
                  return (
                    <label key={opt.id || opt.label} className="flex items-center justify-between text-sm">
                      <span>{opt.label}</span>
                      <input
                        type="checkbox"
                        checked={checked}
                        onChange={() => {
                          const next = checked
                            ? selected.filter((v) => v !== (opt.id || opt.label))
                            : [...selected, opt.id || opt.label];
                          onChange(prop.id, next);
                        }}
                        className="h-4 w-4 rounded border-[var(--c-line)] text-[var(--c-accent)]"
                      />
                    </label>
                  );
                })}
            </div>
          );
        } else if (type === 'date') {
          const dateVal = value ? String(value).slice(0, 10) : '';
          input = (
            <input
              type="date"
              className="input"
              value={dateVal}
              onChange={(e) => onChange(prop.id, e.target.value)}
            />
          );
        } else {
          input = (
            <input
              className="input"
              value={(value as string) ?? ''}
              onChange={(e) => onChange(prop.id, e.target.value)}
            />
          );
        }

        return (
          <label key={prop.id} className="block space-y-1 text-sm">
            <span className="text-[var(--c-muted)]">{prop.name}</span>
            {input}
          </label>
        );
      })}
      {properties.length === 0 && <p className="text-xs text-[var(--c-muted)]">选择一条记录以编辑属性。</p>}
    </div>
  </aside>
);
