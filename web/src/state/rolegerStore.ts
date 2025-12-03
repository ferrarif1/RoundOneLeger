/* eslint-disable @typescript-eslint/no-unused-vars, no-unused-vars */
import { create, type StateCreator } from 'zustand';
import type { Property, RecordItem, Table, View } from '../types/roleger';
import {
  createProperty,
  createRecord,
  createTable,
  createView,
  deleteRecord,
  listProperties,
  listRecords,
  listTables,
  listViews,
  updateRecord,
  saveRecordsBulk,
  updateView
} from '../api/roleger';

type RolegerState = {
  tables: Table[];
  views: Record<string, View[]>;
  properties: Record<string, Property[]>;
  recordIds: Record<string, string[]>;
  recordMap: Record<string, Record<string, RecordItem>>;
  totals: Record<string, number>;
  dirty: Record<string, Record<string, Record<string, unknown>>>;
  autoSaving: Record<string, boolean>;
  loading: boolean;
  error: string | null;
  loadTables: () => Promise<void>;
  loadTableDetail: (tableId: string, limit?: number, filters?: any[], sorts?: any[]) => Promise<void>;
  loadPage: (tableId: string, limit?: number, offset?: number, filters?: any[], sorts?: any[]) => Promise<void>;
  saveRecord: (tableId: string, recordId: string, props: Record<string, unknown>) => Promise<void>;
  addRecord: (tableId: string, props: Record<string, unknown>) => Promise<RecordItem | null>;
  addTable: (payload: Partial<Table>) => Promise<Table | null>;
  addView: (tableId: string, payload: Partial<View>) => Promise<View | null>;
  updateView: (tableId: string, viewId: string, payload: Partial<View>) => Promise<View | null>;
  addProperty: (tableId: string, payload: Partial<Property>) => Promise<Property | null>;
  removeRecord: (tableId: string, recordId: string) => Promise<void>;
  editLocalCell: (tableId: string, recordId: string, propertyId: string, value: unknown) => void;
  applyDirty: (tableId: string) => Promise<void>;
};

const AUTO_SAVE_DELAY = 1800;

const creator: StateCreator<RolegerState> = (set, get) => {
  const autoSaveTimers: Record<string, number | undefined> = {};

  const scheduleAutoSave = (tableId: string) => {
    const existing = autoSaveTimers[tableId];
    if (existing) {
      window.clearTimeout(existing);
    }
    autoSaveTimers[tableId] = window.setTimeout(() => {
      void get().applyDirty(tableId);
    }, AUTO_SAVE_DELAY);
  };

  return {
    tables: [],
    views: {},
    properties: {},
    recordIds: {},
    recordMap: {},
    totals: {},
    dirty: {},
    autoSaving: {},
    loading: false,
    error: null,
    loadTables: async () => {
      set({ loading: true, error: null });
      try {
        const tables = await listTables();
        set({ tables });
      } catch (error: any) {
        set({ error: error?.message || '加载表失败' });
      } finally {
        set({ loading: false });
      }
    },
    loadTableDetail: async (tableId: string, limit = 20, filters?: any[], sorts?: any[]) => {
      set({ loading: true, error: null });
      try {
        const [views, properties] = await Promise.all([listViews(tableId), listProperties(tableId)]);
        const { records, total } = await listRecords(tableId, limit, 0, filters, sorts);
        const ids = records.map((r) => r.id);
        const map = records.reduce<Record<string, RecordItem>>((acc, r) => {
          acc[r.id] = r;
          return acc;
        }, {});
        set((state) => ({
          views: { ...state.views, [tableId]: views },
          properties: { ...state.properties, [tableId]: properties },
          recordIds: { ...state.recordIds, [tableId]: ids },
          recordMap: { ...state.recordMap, [tableId]: map },
          totals: { ...state.totals, [tableId]: total },
          dirty: { ...state.dirty, [tableId]: {} }
        }));
      } catch (error: any) {
        set({ error: error?.message || '加载表详情失败' });
      } finally {
        set({ loading: false });
      }
    },
    loadPage: async (tableId, limit = 20, offset = 0, filters?: any[], sorts?: any[]) => {
      set({ loading: true, error: null });
      try {
      const { records, total } = await listRecords(tableId, limit, offset, filters, sorts);
      const dirtyTable = get().dirty[tableId] || {};
      const map = records.reduce<Record<string, RecordItem>>((acc, r) => {
        const dirtyRow = dirtyTable[r.id];
        const merged = dirtyRow ? { ...r, properties: { ...r.properties, ...dirtyRow } } : r;
        acc[r.id] = merged;
        return acc;
      }, {});
      const ids = records.map((r) => r.id);
      set((state) => ({
        recordIds: { ...state.recordIds, [tableId]: ids },
        recordMap: { ...state.recordMap, [tableId]: map },
        totals: { ...state.totals, [tableId]: total }
      }));
    } catch (error: any) {
      set({ error: error?.message || '加载记录失败' });
    } finally {
        set({ loading: false });
      }
    },
    saveRecord: async (tableId, recordId, props) => {
      try {
        const updated = await updateRecord(tableId, recordId, props);
        set((state) => {
          const map = { ...(state.recordMap[tableId] || {}) };
          if (map[recordId]) map[recordId] = updated;
          return { recordMap: { ...state.recordMap, [tableId]: map } };
        });
      } catch (error: any) {
        set({ error: error?.message || '保存记录失败' });
      }
    },
    addRecord: async (tableId, props) => {
      try {
        const record = await createRecord(tableId, props);
        set((state) => {
          const ids = state.recordIds[tableId] || [];
          const map = state.recordMap[tableId] || {};
          return {
            recordIds: { ...state.recordIds, [tableId]: [record.id, ...ids] },
            recordMap: { ...state.recordMap, [tableId]: { ...map, [record.id]: record } }
          };
        });
        return record;
      } catch (error: any) {
        set({ error: error?.message || '创建记录失败' });
        return null;
      }
    },
    addTable: async (payload) => {
      try {
        const table = await createTable(payload);
        set((state) => ({ tables: [...state.tables, table] }));
        return table;
      } catch (error: any) {
        set({ error: error?.message || '创建表失败' });
        return null;
      }
    },
    addView: async (tableId, payload) => {
      try {
        const view = await createView(tableId, payload);
        set((state) => {
          const list = state.views[tableId] || [];
          return { views: { ...state.views, [tableId]: [...list, view] } };
        });
        return view;
      } catch (error: any) {
        set({ error: error?.message || '创建视图失败' });
        return null;
      }
    },
    updateView: async (tableId, viewId, payload) => {
      try {
        const view = await updateView(tableId, viewId, payload);
        set((state) => {
          const list = state.views[tableId] || [];
          const next = list.map((v) => (v.id === viewId ? { ...v, ...view } : v));
          return { views: { ...state.views, [tableId]: next } };
        });
        return view;
      } catch (error: any) {
        set({ error: error?.message || '更新视图失败' });
        return null;
      }
    },
    addProperty: async (tableId, payload) => {
      try {
        const prop = await createProperty(tableId, payload);
        set((state) => {
          const list = state.properties[tableId] || [];
          return { properties: { ...state.properties, [tableId]: [...list, prop] } };
        });
        return prop;
      } catch (error: any) {
        set({ error: error?.message || '创建字段失败' });
        return null;
      }
    },
    removeRecord: async (tableId, recordId) => {
      try {
        await deleteRecord(tableId, recordId);
        set((state) => {
          const ids = (state.recordIds[tableId] || []).filter((id) => id !== recordId);
          const map = { ...(state.recordMap[tableId] || {}) };
          delete map[recordId];
          return { recordIds: { ...state.recordIds, [tableId]: ids }, recordMap: { ...state.recordMap, [tableId]: map } };
        });
      } catch (error: any) {
        set({ error: error?.message || '删除记录失败' });
      }
    },
    editLocalCell: (tableId, recordId, propertyId, value) => {
      set((state) => {
        const dirtyForTable = state.dirty[tableId] || {};
        const dirtyRow = dirtyForTable[recordId] || {};
      const nextDirty = {
        ...state.dirty,
        [tableId]: {
          ...dirtyForTable,
          [recordId]: { ...dirtyRow, [propertyId]: value }
        }
      };
      const map = state.recordMap[tableId] || {};
      const existing = map[recordId];
      const nextMap = {
        ...state.recordMap,
        [tableId]: {
          ...map,
          [recordId]: existing ? { ...existing, properties: { ...existing.properties, [propertyId]: value } } : existing
        }
      };
      return { dirty: nextDirty, recordMap: nextMap };
    });
    scheduleAutoSave(tableId);
  },
    applyDirty: async (tableId) => {
      const dirtyTable = get().dirty[tableId];
      if (!dirtyTable) return;
      const updates = Object.entries(dirtyTable).map(([recordId, props]) => ({ id: recordId, properties: props }));
      set((state) => ({ autoSaving: { ...state.autoSaving, [tableId]: true } }));
      try {
        await saveRecordsBulk(tableId, updates);
        set((state) => {
          const nextDirty = { ...state.dirty };
          delete nextDirty[tableId];
          const nextAuto = { ...state.autoSaving };
          delete nextAuto[tableId];
          return { dirty: nextDirty, autoSaving: nextAuto };
        });
      } finally {
        const timer = autoSaveTimers[tableId];
        if (timer) {
          window.clearTimeout(timer);
          delete autoSaveTimers[tableId];
        }
      }
    }
  };
};

export const useRolegerStore = create<RolegerState>(creator);
