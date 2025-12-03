import type { Block } from '../../types/roleger';

type BlockRendererProps = {
  block: Block;
};

export const BlockRenderer = ({ block }: BlockRendererProps) => {
  switch (block.type) {
    case 'heading':
      return <h3 className="text-base font-semibold text-[var(--c-text)]">{block.props?.text as string}</h3>;
    case 'quote':
      return (
        <blockquote className="border-l-2 border-[var(--c-line)] pl-3 text-sm text-[var(--c-muted)]">
          {block.props?.text as string}
        </blockquote>
      );
    case 'code':
      return (
        <pre className="rounded-[6px] border border-[var(--c-line)] bg-[var(--c-surface-subtle)] px-3 py-2 text-[13px] text-[var(--c-text)]">
          <code>{block.props?.text as string}</code>
        </pre>
      );
    default:
      return <p className="text-sm text-[var(--c-text)]">{block.props?.text as string}</p>;
  }
};
