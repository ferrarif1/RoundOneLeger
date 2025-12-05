import { useEffect, useMemo, useState } from 'react';
import { useNavigate, useParams } from 'react-router-dom';
import { SidebarNav } from '../components/layout/SidebarNav';
import { TopBarLite } from '../components/layout/TopBarLite';
import { RightPanel } from '../components/layout/RightPanel';
import { DataTable } from '../components/table/DataTable';
import { ViewToolbar } from '../components/table/ViewToolbar';
import { ColumnVisibility } from '../components/table/ColumnVisibility';
import { ImportDialog } from '../components/import/ImportDialog';
import { useRoledgerStore } from '../state/roledgerStore';
import type { Table, View, RecordItem, Property } from '../types/roledger';

const WorkspacePage = () => {
  const { tableId } = useParams<{ tableId: string }>();
  const {
    tables,
    views,
    properties,
    loading,
    loadTables,
    loadTableDetail,
    loadPage,
    addRecord,
    addTable,
    removeRecord,
    addView,
    updateView,
    addProperty,
    editLocalCell,
    applyDirty,
    dirty
  } = useRoledgerStore();
  const navigate = useNavigate();
  const [activeViewId, setActiveViewId] = useState<string | undefined>(undefined);
  const [activeRecordId, setActiveRecordId] = useState<string | undefined>(undefined);
  const [newViewName, setNewViewName] = useState('');
  const [newViewLayout, setNewViewLayout] = useState<'table' | 'list' | 'gallery' | 'kanban'>('table');
  const [newPropName, setNewPropName] = useState('');
  const [newPropType, setNewPropType] = useState<'text' | 'number' | 'date' | 'select' | 'multi_select'>('text');
  const [pageOffset, setPageOffset] = useState(0);
  const pageLimit = 20;
  const [filterProp, setFilterProp] = useState<string>('');
  const [filterOp, setFilterOp] = useState<'eq' | 'contains'>('eq');
  const [filterValue, setFilterValue] = useState('');
  const [sortProp, setSortProp] = useState<string>('');
  const [sortDir, setSortDir] = useState<'asc' | 'desc'>('asc');
  const [showImport, setShowImport] = useState(false);

  useEffect(() => {
    loadTables();
  }, [loadTables]);

  useEffect(() => {
    if (tableId) {
      loadTableDetail(tableId, pageLimit);
      setActiveViewId(undefined);
      setActiveRecordId(undefined);
      setPageOffset(0);
    }
  }, [tableId, loadTableDetail]);

  const navTree = tables.map((t: Table) => ({ id: t.id, title: t.name, isActive: t.id === tableId }));
  const currentTable = tables.find((t: Table) => t.id === tableId);
  const viewList = views[tableId || ''] || [];
  const currentView = useMemo(() => {
    if (activeViewId) return viewList.find((v: View) => v.id === activeViewId);
    return viewList[0];
  }, [activeViewId, viewList]);
  const currentColumns = currentView?.columns ?? [];
  const currentProperties = (properties[tableId || ''] as Property[]) || [];
  const recordIds = useRoledgerStore((state) => state.recordIds[tableId || ''] || []);
  const recordMap = useRoledgerStore((state) => state.recordMap[tableId || ''] || {});
  const currentRecords = recordIds.map((id) => recordMap[id]).filter(Boolean) as RecordItem[];
  const activeRecord = currentRecords.find((r: RecordItem) => r.id === activeRecordId);
  const total = tableId ? useRoledgerStore.getState().totals?.[tableId] || currentRecords.length : currentRecords.length;
  const totalPages = Math.max(1, Math.ceil((total || 1) / pageLimit));
  const currentPage = Math.floor(pageOffset / pageLimit) + 1;
  const [jumpPage, setJumpPage] = useState<string>('');

  useEffect(() => {
    if (tableId && currentView) {
      setPageOffset(0);
      loadPage(tableId, pageLimit, 0, currentView.filters, currentView.sort);
    }
  }, [tableId, currentView, loadPage]);

  const hasDirty = tableId && dirty[tableId] && Object.keys(dirty[tableId]).length > 0;

  const handlePageChange = async (nextOffset: number) => {
    if (!tableId || nextOffset < 0) return;
    if (hasDirty) {
      const confirmSave = window.confirm('当前页有未保存更改，是否先保存并继续翻页？');
      if (!confirmSave) return;
      await applyDirty(tableId);
    }
    setPageOffset(nextOffset);
    loadPage(tableId, pageLimit, nextOffset, currentView?.filters, currentView?.sort);
  };

  const handleToggleColumn = (propertyId: string) => {
    if (!currentView || !tableId) return;
    const nextColumns = currentColumns.map((col) =>
      col.propertyId === propertyId ? { ...col, visible: !col.visible } : col
    );
    setActiveViewId(currentView.id);
    const viewIndex = viewList.findIndex((v) => v.id === currentView.id);
    const nextViewList = [...viewList];
    nextViewList[viewIndex] = { ...currentView, columns: nextColumns };
    useRoledgerStore.setState((state) => ({
      views: { ...state.views, [tableId]: nextViewList }
    }));
    updateView(tableId, currentView.id, { columns: nextColumns });
  };

  return (
    <div className="flex min-h-screen bg-[var(--c-bg)] text-[var(--c-text)]">
      <SidebarNav
        tree={navTree}
        onToggle={() => {}}
        onSelect={(id) => navigate(`/workspace/${id}`)}
        onCreate={async () => {
          const created = await addTable({ name: '新建表' });
          if (created?.id) navigate(`/workspace/${created.id}`);
        }}
      />
      <div className="flex min-h-screen flex-1 flex-col">
        <TopBarLite title={currentTable?.name || '数据库'} />
        <main className="flex flex-1 gap-4 p-4">
          <section className="flex-1 rounded-lg border border-[var(--c-line)] bg-white p-4">
            {currentView ? (
              <>
                <ViewToolbar
                  views={viewList}
                  activeViewId={currentView.id}
                  onSwitch={(id) => setActiveViewId(id)}
                  onFilter={() => {}}
                  onLayoutChange={() => {}}
                  onCreateView={() => {
                    if (!tableId) return;
                    addView(tableId, { name: `视图 ${viewList.length + 1}`, layout: 'table', columns: currentColumns });
                  }}
                />
                <div className="mb-3 flex justify-end">
                  <button type="button" className="btn" onClick={() => setShowImport(true)}>
                    导入数据（自动创建表）
                  </button>
                </div>
              <DataTable
                properties={currentProperties}
                columns={currentColumns}
                records={currentRecords}
                activeIds={activeRecordId ? [activeRecordId] : []}
                loading={loading}
                onSelect={(id) => setActiveRecordId(id)}
                onCellEdit={(id, propertyId, value) => editLocalCell(tableId || '', id, propertyId, value)}
                onResizeColumn={(propertyId, width) => {
                  if (!currentView || !tableId) return;
                  const nextColumns = currentColumns.map((col) =>
                    col.propertyId === propertyId ? { ...col, width } : col
                  );
                  const viewIndex = viewList.findIndex((v) => v.id === currentView.id);
                  const nextViewList = [...viewList];
                  nextViewList[viewIndex] = { ...currentView, columns: nextColumns };
                  useRoledgerStore.setState((state) => ({
                    views: { ...state.views, [tableId]: nextViewList }
                  }));
                  updateView(tableId, currentView.id, { columns: nextColumns });
                }}
                onDelete={(id) => removeRecord(tableId || '', id)}
                  onAddRecord={async () => {
                    await addRecord(tableId || '', {});
                  }}
                />
              <div className="mt-3">
                <ColumnVisibility properties={currentProperties} columns={currentColumns} onToggle={handleToggleColumn} />
              </div>
              <div className="mt-3 grid gap-3 md:grid-cols-2">
                  <div className="rounded-lg border border-[var(--c-line)] bg-white p-3 text-sm">
                    <p className="mb-2 text-xs uppercase tracking-[0.2em] text-[var(--c-muted)]">新建视图</p>
                    <div className="space-y-2">
                      <input
                        className="input"
                        placeholder="视图名称"
                        value={newViewName}
                        onChange={(e) => setNewViewName(e.target.value)}
                      />
                      <select
                        className="select"
                        value={newViewLayout}
                        onChange={(e) => setNewViewLayout(e.target.value as typeof newViewLayout)}
                      >
                        <option value="table">表格</option>
                        <option value="list">列表</option>
                        <option value="gallery">图库</option>
                        <option value="kanban">看板</option>
                      </select>
                      <button
                        type="button"
                        className="btn"
                        onClick={() => {
                          if (!tableId || !newViewName.trim()) return;
                          addView(tableId, { name: newViewName.trim(), layout: newViewLayout, columns: currentColumns });
                          setNewViewName('');
                        }}
                      >
                        保存视图
                      </button>
                    </div>
                  </div>
                  <div className="rounded-lg border border-[var(--c-line)] bg-white p-3 text-sm">
                    <p className="mb-2 text-xs uppercase tracking-[0.2em] text-[var(--c-muted)]">新增字段</p>
                    <div className="space-y-2">
                      <input
                        className="input"
                        placeholder="字段名称"
                        value={newPropName}
                        onChange={(e) => setNewPropName(e.target.value)}
                      />
                      <select
                        className="select"
                        value={newPropType}
                        onChange={(e) => setNewPropType(e.target.value as typeof newPropType)}
                      >
                        <option value="text">文本</option>
                        <option value="number">数字</option>
                        <option value="date">日期</option>
                        <option value="select">单选</option>
                        <option value="multi_select">多选</option>
                      </select>
                      <button
                        type="button"
                        className="btn"
                        onClick={() => {
                          if (!tableId || !newPropName.trim()) return;
                          addProperty(tableId, {
                            name: newPropName.trim(),
                            type: newPropType,
                            order: currentProperties.length
                          });
                          setNewPropName('');
                        }}
                      >
                        添加字段
                      </button>
                    </div>
                  </div>
                  <div className="rounded-lg border border-[var(--c-line)] bg-white p-3 text-sm">
                    <p className="mb-2 text-xs uppercase tracking-[0.2em] text-[var(--c-muted)]">过滤与排序</p>
                    <div className="space-y-2">
                      <select
                        className="select"
                        value={filterProp}
                        onChange={(e) => setFilterProp(e.target.value)}
                      >
                        <option value="">选择字段</option>
                        {currentProperties.map((p) => (
                          <option key={p.id} value={p.id}>
                            {p.name}
                          </option>
                        ))}
                      </select>
                      <select
                        className="select"
                        value={filterOp}
                        onChange={(e) => setFilterOp(e.target.value as typeof filterOp)}
                      >
                        <option value="eq">等于</option>
                        <option value="contains">包含</option>
                      </select>
                      <input
                        className="input"
                        placeholder="过滤值"
                        value={filterValue}
                        onChange={(e) => setFilterValue(e.target.value)}
                      />
                      <select
                        className="select"
                        value={sortProp}
                        onChange={(e) => setSortProp(e.target.value)}
                      >
                        <option value="">排序字段</option>
                        {currentProperties.map((p) => (
                          <option key={p.id} value={p.id}>
                            {p.name}
                          </option>
                        ))}
                      </select>
                      <select
                        className="select"
                        value={sortDir}
                        onChange={(e) => setSortDir(e.target.value as typeof sortDir)}
                      >
                        <option value="asc">升序</option>
                        <option value="desc">降序</option>
                      </select>
                      <button
                        type="button"
                        className="btn"
                        onClick={() => {
                          if (!tableId || !currentView) return;
                          const filters = filterProp && filterValue ? [{ propertyId: filterProp, op: filterOp, value: filterValue }] : [];
                          const sort = sortProp ? [{ propertyId: sortProp, direction: sortDir }] : [];
                          updateView(tableId, currentView.id, { filters, sort });
                          loadPage(tableId, pageLimit, 0, filters, sort);
                          setPageOffset(0);
                        }}
                      >
                        应用
                      </button>
                    </div>
                  </div>
                </div>
              </>
            ) : (
              <p className="text-sm text-[var(--c-muted)]">请选择或创建视图。</p>
            )}
            {tableId && (
              <div className="mt-3 flex items-center justify-between gap-2 text-sm">
                <span className="text-[var(--c-muted)]">
                  显示 {pageOffset + 1}-{Math.min(pageOffset + pageLimit, total)} / {total || '...'} （共 {totalPages} 页）
                </span>
                <button
                  type="button"
                  className="btn"
                  disabled={pageOffset === 0}
                  onClick={() => handlePageChange(Math.max(0, pageOffset - pageLimit))}
                >
                  上一页
                </button>
                <button
                  type="button"
                  className="btn"
                  disabled={pageOffset + pageLimit >= total}
                  onClick={() => handlePageChange(pageOffset + pageLimit)}
                >
                  下一页
                </button>
                <div className="flex items-center gap-2">
                  <span className="text-[var(--c-muted)]">
                    第 {currentPage} 页 / 共 {totalPages} 页
                  </span>
                  <input
                    type="number"
                    min={1}
                    max={totalPages}
                    className="w-20 rounded-md border border-[var(--c-line)] bg-white px-2 py-1 text-sm"
                    value={jumpPage}
                    onChange={(e) => setJumpPage(e.target.value)}
                    onKeyDown={(e) => {
                      if (e.key === 'Enter') {
                        const val = Number(jumpPage);
                        if (!Number.isNaN(val) && val >= 1 && val <= totalPages) {
                          handlePageChange((val - 1) * pageLimit);
                          setJumpPage('');
                        }
                      }
                    }}
                    placeholder="跳转页"
                  />
                </div>
              </div>
            )}
          </section>
          <RightPanel
            title={activeRecord ? '属性' : '未选择记录'}
            properties={currentProperties.map((p) => ({ id: p.id, name: p.name, type: p.type }))}
            values={activeRecord?.properties}
            onChange={(propId, value) => {
              if (!activeRecord) return;
              editLocalCell(tableId || '', activeRecord.id, propId, value);
            }}
          />
        </main>
            {tableId && hasDirty && (
              <div className="fixed bottom-6 right-6 flex items-center gap-3 rounded-full border border-[var(--c-line)] bg-white px-4 py-2 shadow-[var(--shadow-popover)]">
                <span className="text-sm text-[var(--c-text)]">有 {Object.keys(dirty[tableId]).length} 条未保存更改</span>
                <button
                  type="button"
                  className="btn"
                  onClick={async () => {
                    await applyDirty(tableId);
                    if (currentView) {
                      loadPage(tableId, pageLimit, pageOffset, currentView.filters, currentView.sort);
                    }
                  }}
                >
                  保存
                </button>
              </div>
            )}
        <ImportDialog
          open={showImport}
          onClose={() => setShowImport(false)}
          onUploaded={(id) => {
            setShowImport(false);
            navigate(`/workspace/${id}`);
            loadTableDetail(id);
          }}
        />
      </div>
    </div>
  );
};

export default WorkspacePage;
