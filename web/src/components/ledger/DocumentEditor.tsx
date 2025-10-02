import {
  ChangeEvent,
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState
} from 'react';

const DOCUMENT_BLOCK_OPTIONS = [
  { value: 'paragraph', label: '正文', command: 'P' },
  { value: 'h1', label: '标题 1', command: 'H1' },
  { value: 'h2', label: '标题 2', command: 'H2' },
  { value: 'h3', label: '标题 3', command: 'H3' },
  { value: 'quote', label: '引用', command: 'BLOCKQUOTE' },
  { value: 'code', label: '代码块', command: 'PRE' }
] as const;

interface DocumentEditorProps {
  value: string;
  editable: boolean;
  onChange: (html: string) => void;
  onStatus: (message: string) => void;
}

type DocumentBlockValue = (typeof DOCUMENT_BLOCK_OPTIONS)[number]['value'];

const ToolbarButton = ({
  label,
  tooltip,
  onClick,
  disabled = false
}: {
  label: string;
  tooltip?: string;
  onClick: () => void;
  disabled?: boolean;
}) => (
  <button
    type="button"
    onClick={onClick}
    disabled={disabled}
    title={tooltip ?? label}
    aria-label={tooltip ?? label}
    className="rounded-full border border-white/60 bg-white/90 px-2 py-1 text-[11px] font-semibold uppercase tracking-wide text-[var(--muted)] shadow-[var(--shadow-sm)] transition hover:border-[var(--accent)]/60 hover:text-[var(--text)] disabled:cursor-not-allowed disabled:opacity-60"
  >
    {label}
  </button>
);

export const DocumentEditor = ({ value, editable, onChange, onStatus }: DocumentEditorProps) => {
  const documentRef = useRef<HTMLDivElement | null>(null);
  const imageInputRef = useRef<HTMLInputElement | null>(null);
  const audioInputRef = useRef<HTMLInputElement | null>(null);
  const videoInputRef = useRef<HTMLInputElement | null>(null);
  const [selectedBlock, setSelectedBlock] = useState<DocumentBlockValue>('paragraph');

  useEffect(() => {
    const container = documentRef.current;
    if (!container) {
      return;
    }
    if (container.innerHTML !== value) {
      container.innerHTML = value || '';
    }
  }, [value]);

  const execCommand = useCallback(
    (command: string, commandValue?: string) => {
      if (!editable || !documentRef.current) return;
      documentRef.current.focus();
      try {
        document.execCommand(command, false, commandValue);
        onChange(documentRef.current.innerHTML);
      } catch (error) {
        console.warn('执行编辑命令失败', command, error);
      }
    },
    [editable, onChange]
  );

  const insertHtml = useCallback(
    (html: string) => {
      execCommand('insertHTML', html);
    },
    [execCommand]
  );

  const applyBlock = useCallback(
    (block: DocumentBlockValue) => {
      const found = DOCUMENT_BLOCK_OPTIONS.find((option) => option.value === block);
      execCommand('formatBlock', found?.command ?? 'P');
    },
    [execCommand]
  );

  const handleBlockSelect = useCallback(
    (event: ChangeEvent<HTMLSelectElement>) => {
      const next = (event.target.value as DocumentBlockValue) ?? 'paragraph';
      setSelectedBlock(next);
      applyBlock(next);
    },
    [applyBlock]
  );

  const insertChecklist = useCallback(() => {
    insertHtml('<ul class="doc-checklist"><li><label><input type="checkbox" /> 待办事项</label></li></ul>');
  }, [insertHtml]);

  const insertCallout = useCallback(() => {
    insertHtml('<div class="doc-callout"><strong>提示：</strong>请在此补充说明。</div>');
  }, [insertHtml]);

  const insertCodeBlock = useCallback(() => {
    insertHtml('<pre class="doc-code"><code>// 在此编写示例代码</code></pre>');
  }, [insertHtml]);

  const insertTable = useCallback(() => {
    const rowsInput = window.prompt('请输入表格行数 (1-20)', '3');
    const colsInput = window.prompt('请输入表格列数 (1-10)', '3');
    let rowsCount = Number.parseInt(rowsInput ?? '0', 10);
    let colsCount = Number.parseInt(colsInput ?? '0', 10);
    if (Number.isNaN(rowsCount) || rowsCount <= 0) rowsCount = 3;
    if (Number.isNaN(colsCount) || colsCount <= 0) colsCount = 3;
    rowsCount = Math.min(rowsCount, 20);
    colsCount = Math.min(colsCount, 10);
    const rows = Array.from({ length: rowsCount })
      .map(
        () =>
          `<tr>${Array.from({ length: colsCount })
            .map(() => '<td>内容</td>')
            .join('')}</tr>`
      )
      .join('');
    insertHtml(`<table>${rows}</table>`);
  }, [insertHtml]);

  const insertDivider = useCallback(() => {
    insertHtml('<hr />');
  }, [insertHtml]);

  const handleMediaInsert = useCallback(
    (event: ChangeEvent<HTMLInputElement>, type: 'image' | 'audio' | 'video') => {
      const file = event.target.files?.[0];
      event.target.value = '';
      if (!file || !editable) return;
      const reader = new FileReader();
      reader.onload = () => {
        const result = reader.result;
        if (typeof result === 'string') {
          if (type === 'image') {
            insertHtml(
              `<figure class="doc-figure"><img src="${result}" alt="插入图片" /><figcaption>图片说明</figcaption></figure>`
            );
            return;
          }
          const tag = type === 'audio' ? 'audio' : 'video';
          const extra = type === 'video' ? ' class="doc-video"' : '';
          insertHtml(`<${tag}${extra} controls src="${result}"></${tag}>`);
        }
      };
      reader.readAsDataURL(file);
    },
    [editable, insertHtml]
  );

  const applyHighlight = useCallback(() => {
    execCommand('hiliteColor', '#FFF9C4');
  }, [execCommand]);

  const clearFormatting = useCallback(() => {
    if (documentRef.current) {
      const normalized = documentRef.current.innerHTML.replace(/<mark class="doc-search-highlight">(.*?)<\/mark>/g, '$1');
      documentRef.current.innerHTML = normalized;
      onChange(normalized);
    }
    execCommand('removeFormat');
    execCommand('unlink');
  }, [execCommand, onChange]);

  const handleSearchHighlight = useCallback(() => {
    const term = window.prompt('请输入需要高亮的内容');
    if (!term || !documentRef.current) {
      return;
    }
    const container = documentRef.current;
    const resetHtml = container.innerHTML.replace(/<mark class="doc-search-highlight">(.*?)<\/mark>/g, '$1');
    const escaped = term.replace(/[-/\\^$*+?.()|[\]{}]/g, '\\$&');
    const regex = new RegExp(escaped, 'gi');
    const hasMatch = regex.test(resetHtml);
    regex.lastIndex = 0;
    if (!hasMatch) {
      container.innerHTML = resetHtml;
      onChange(resetHtml);
      onStatus(`未找到 “${term}” 的匹配内容。`);
      return;
    }
    const highlighted = resetHtml.replace(regex, (match) => `<mark class="doc-search-highlight">${match}</mark>`);
    container.innerHTML = highlighted;
    onChange(highlighted);
    onStatus(`已高亮显示 “${term}” 的匹配内容。`);
  }, [onChange, onStatus]);

  const handleInput = useCallback(() => {
    if (!documentRef.current) {
      return;
    }
    onChange(documentRef.current.innerHTML);
  }, [onChange]);

  const toolbar = useMemo(
    () => (
      <div className="flex flex-wrap items-center gap-2 text-xs text-[var(--muted)]">
        <select
          value={selectedBlock}
          onChange={handleBlockSelect}
          disabled={!editable}
          className="rounded-full border border-[var(--muted)]/30 bg-white/80 px-3 py-2 text-xs text-[var(--muted)] focus:border-[var(--accent)] focus:outline-none focus:ring-2 focus:ring-[var(--accent)]/20 disabled:cursor-not-allowed"
        >
          {DOCUMENT_BLOCK_OPTIONS.map((option) => (
            <option key={option.value} value={option.value}>
              {option.label}
            </option>
          ))}
        </select>
        <ToolbarButton label="B" tooltip="加粗" onClick={() => execCommand('bold')} disabled={!editable} />
        <ToolbarButton label="I" tooltip="斜体" onClick={() => execCommand('italic')} disabled={!editable} />
        <ToolbarButton label="U" tooltip="下划线" onClick={() => execCommand('underline')} disabled={!editable} />
        <ToolbarButton label="S" tooltip="删除线" onClick={() => execCommand('strikeThrough')} disabled={!editable} />
        <ToolbarButton label="HL" tooltip="高亮选区" onClick={applyHighlight} disabled={!editable} />
        <ToolbarButton label="Link" tooltip="插入链接" onClick={() => execCommand('createLink', window.prompt('请输入链接地址') ?? '')} disabled={!editable} />
        <ToolbarButton label="•" tooltip="项目符号列表" onClick={() => execCommand('insertUnorderedList')} disabled={!editable} />
        <ToolbarButton label="1." tooltip="编号列表" onClick={() => execCommand('insertOrderedList')} disabled={!editable} />
        <ToolbarButton label="☑" tooltip="任务清单" onClick={insertChecklist} disabled={!editable} />
        <ToolbarButton label="L" tooltip="左对齐" onClick={() => execCommand('justifyLeft')} disabled={!editable} />
        <ToolbarButton label="C" tooltip="居中" onClick={() => execCommand('justifyCenter')} disabled={!editable} />
        <ToolbarButton label="R" tooltip="右对齐" onClick={() => execCommand('justifyRight')} disabled={!editable} />
        <ToolbarButton label="表" tooltip="插入表格" onClick={insertTable} disabled={!editable} />
        <ToolbarButton label="<>" tooltip="插入代码块" onClick={insertCodeBlock} disabled={!editable} />
        <ToolbarButton label="Call" tooltip="提示块" onClick={insertCallout} disabled={!editable} />
        <ToolbarButton label="HR" tooltip="分割线" onClick={insertDivider} disabled={!editable} />
        <ToolbarButton label="IMG" tooltip="插入图片" onClick={() => imageInputRef.current?.click()} disabled={!editable} />
        <ToolbarButton label="AUD" tooltip="插入音频" onClick={() => audioInputRef.current?.click()} disabled={!editable} />
        <ToolbarButton label="VID" tooltip="插入视频" onClick={() => videoInputRef.current?.click()} disabled={!editable} />
        <ToolbarButton label="Find" tooltip="搜索并高亮" onClick={handleSearchHighlight} disabled={!editable} />
        <ToolbarButton label="CLR" tooltip="清除格式" onClick={clearFormatting} disabled={!editable} />
      </div>
    ), [
      applyHighlight,
      clearFormatting,
      editable,
      execCommand,
      handleBlockSelect,
      handleSearchHighlight,
      insertCallout,
      insertChecklist,
      insertCodeBlock,
      insertDivider,
      insertTable,
      selectedBlock
    ]
  );

  return (
    <section className="mt-10 space-y-4">
      <div className="flex flex-wrap items-center gap-3">
        <h3 className="text-base font-semibold text-[var(--text)]">在线文档</h3>
        {toolbar}
      </div>
      <div className="rounded-[var(--radius-lg)] border border-white/60 bg-white/90 p-4 shadow-[var(--shadow-sm)]">
        <div
          ref={documentRef}
          className="doc-editor min-h-[320px] outline-none"
          contentEditable={editable}
          data-placeholder={editable ? '在此撰写文档内容…' : undefined}
          suppressContentEditableWarning
          onInput={handleInput}
        />
      </div>
      <input
        ref={imageInputRef}
        type="file"
        accept="image/*"
        className="hidden"
        onChange={(event) => handleMediaInsert(event, 'image')}
      />
      <input
        ref={audioInputRef}
        type="file"
        accept="audio/*"
        className="hidden"
        onChange={(event) => handleMediaInsert(event, 'audio')}
      />
      <input
        ref={videoInputRef}
        type="file"
        accept="video/*"
        className="hidden"
        onChange={(event) => handleMediaInsert(event, 'video')}
      />
    </section>
  );
};
