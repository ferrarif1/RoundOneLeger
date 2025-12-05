/* eslint-disable @typescript-eslint/no-unused-vars, no-unused-vars */
import { useMemo, useState } from 'react';
import { PlusIcon } from '@heroicons/react/24/outline';
import type { Block } from '../../types/roleger';
import { BlockRenderer } from './BlockRenderer';

type BlockEditorProps = {
  blocks: Block[];
  onChange: (next: Block[]) => void;
};

const createBlock = (): Block => ({
  id: crypto.randomUUID(),
  type: 'paragraph',
  props: { text: '' },
  children: [],
  order: Date.now(),
  pageId: ''
});

export const BlockEditor = ({ blocks, onChange }: BlockEditorProps) => {
  const [activeId, setActiveId] = useState<string | null>(null);

  const sorted = useMemo(() => [...blocks].sort((a, b) => a.order - b.order), [blocks]);

  const handleTextChange = (id: string, text: string) => {
    const next = blocks.map((b) => (b.id === id ? { ...b, props: { ...b.props, text } } : b));
    onChange(next);
  };

  const handleAdd = () => {
    const nextBlock = { ...createBlock(), order: sorted.length ? sorted[sorted.length - 1].order + 1 : 0 };
    onChange([...blocks, nextBlock]);
    setActiveId(nextBlock.id);
  };

  return (
    <div className="space-y-2">
      {sorted.map((block) => (
        <div
          key={block.id}
          className="rounded-md border border-transparent px-2 py-1.5 hover:border-[var(--c-line)]"
          onClick={() => setActiveId(block.id)}
        >
          {activeId === block.id ? (
            <textarea
              className="w-full resize-none rounded-[6px] border border-[var(--c-line)] bg-white px-2 py-1.5 text-sm text-[var(--c-text)] focus:border-[var(--c-line-strong)] focus:outline-none"
              value={(block.props?.text as string) ?? ''}
              onChange={(e) => handleTextChange(block.id, e.target.value)}
              placeholder="输入内容，使用 / 插入组件"
              rows={1}
            />
          ) : (
            <BlockRenderer block={block} />
          )}
        </div>
      ))}
      <button
        type="button"
        onClick={handleAdd}
        className="flex items-center gap-2 rounded-md border border-[var(--c-line)] bg-white px-2 py-2 text-sm text-[var(--c-text)]"
      >
        <PlusIcon className="h-4 w-4" />
        新建块
      </button>
    </div>
  );
};
