import type { Property, RecordItem, Table, View } from '../types/roledger';
import api from './client';

export const listTables = async () => {
  const { data } = await api.get<{ tables: Table[] }>('/api/v1/tables');
  return data.tables;
};

export const createTable = async (payload: Partial<Table>) => {
  const { data } = await api.post<{ table: Table }>('/api/v1/tables', payload);
  return data.table;
};

export const createView = async (tableId: string, payload: Partial<View>) => {
  const { data } = await api.post<{ view: View }>(`/api/v1/tables/${tableId}/views`, payload);
  return data.view;
};

export const updateView = async (tableId: string, viewId: string, payload: Partial<View>) => {
  const { data } = await api.put<{ view: View }>(`/api/v1/tables/${tableId}/views/${viewId}`, payload);
  return data.view;
};

export const createProperty = async (tableId: string, payload: Partial<Property>) => {
  const { data } = await api.post<{ property: Property }>(`/api/v1/tables/${tableId}/properties`, payload);
  return data.property;
};

export const listViews = async (tableId: string) => {
  const { data } = await api.get<{ views: View[] }>(`/api/v1/tables/${tableId}/views`);
  return data.views;
};

export const listProperties = async (tableId: string) => {
  const { data } = await api.get<{ properties: Property[] }>(`/api/v1/tables/${tableId}/properties`);
  return data.properties;
};

export const listRecords = async (
  tableId: string,
  limit?: number,
  offset?: number,
  filters?: Array<{ propertyId: string; op: string; value: unknown }>,
  sorts?: Array<{ propertyId: string; direction: 'asc' | 'desc' }>
) => {
  const params: Record<string, unknown> = {};
  if (limit) params.limit = limit;
  if (offset) params.offset = offset;
  if (filters?.length) params.filters = JSON.stringify(filters.map((f) => ({ property: f.propertyId, op: f.op, value: f.value })));
  if (sorts?.length) params.sort = JSON.stringify(sorts.map((s) => ({ property: s.propertyId, direction: s.direction })));
  const { data } = await api.get<{ records: RecordItem[]; total: number; limit: number; offset: number }>(
    `/api/v1/tables/${tableId}/records`,
    { params }
  );
  return data;
};

export const updateRecord = async (tableId: string, recordId: string, properties: Record<string, unknown>) => {
  const { data } = await api.put<{ record: RecordItem }>(`/api/v1/tables/${tableId}/records/${recordId}`, {
    properties
  });
  return data.record;
};

export const createRecord = async (tableId: string, properties: Record<string, unknown>) => {
  const { data } = await api.post<{ record: RecordItem }>(`/api/v1/tables/${tableId}/records`, {
    properties
  });
  return data.record;
};

export const deleteRecord = async (tableId: string, recordId: string) => {
  await api.delete(`/api/v1/tables/${tableId}/records/${recordId}`);
};

export const saveRecordsBulk = async (tableId: string, updates: { id: string; properties: Record<string, unknown> }[]) => {
  const { data } = await api.post<{ records: RecordItem[] }>(`/api/v1/tables/${tableId}/records/bulk`, { updates });
  return data.records;
};

export type ImportTask = {
  id: string;
  tableId: string;
  status: string;
  progress: number;
  error?: string;
};

export const startImport = async (form: FormData): Promise<{ taskId: string; tableId: string }> => {
  // 让浏览器自动设置 multipart 边界，避免手动设置 Header 触发编码问题
  const { data } = await api.post('/api/v1/import', form);
  return data as { taskId: string; tableId: string };
};

export const getImportTask = async (id: string): Promise<ImportTask> => {
  const { data } = await api.get<{ task: ImportTask }>(`/api/v1/import/${id}`);
  return data.task;
};
