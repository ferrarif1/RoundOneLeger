export type PropertyType =
  | 'text'
  | 'number'
  | 'date'
  | 'select'
  | 'multi_select'
  | 'relation'
  | 'checkbox'
  | 'url'
  | 'email'
  | 'phone'
  | 'user'
  | 'formula'
  | 'rollup';

export type SelectOption = { id: string; label: string; color?: string };

export type Property = {
  id: string;
  tableId: string;
  name: string;
  type: PropertyType;
  options?: SelectOption[];
  relation?: { targetTableId: string; relationType: 'm2m' | 'o2m' | 'o2o' };
  formula?: string;
  rollup?: { sourcePropertyId: string; relationPropertyId: string; aggregation: string };
  order: number;
};

export type ViewLayout = 'table' | 'list' | 'gallery' | 'kanban';

export type ViewColumn = {
  propertyId: string;
  visible: boolean;
  width?: number;
  order: number;
};

export type ViewFilter = {
  propertyId: string;
  op: string;
  value: unknown;
};

export type ViewSort = {
  propertyId: string;
  direction: 'asc' | 'desc';
};

export type View = {
  id: string;
  tableId: string;
  name: string;
  layout: ViewLayout;
  filters: ViewFilter[];
  sort: ViewSort[];
  group?: { propertyId: string; direction: 'asc' | 'desc' };
  columns: ViewColumn[];
};

export type RecordItem = {
  id: string;
  tableId: string;
  properties: Record<string, unknown>;
  createdBy?: string;
  updatedBy?: string;
  updatedAt?: string;
  version?: number;
  trashedAt?: string | null;
};

export type Table = {
  id: string;
  name: string;
  description?: string;
  properties: Property[];
  views: View[];
  createdBy?: string;
  updatedAt?: string;
  version?: number;
};

export type Block = {
  id: string;
  type: string;
  props: Record<string, unknown>;
  children?: Block[];
  order: number;
  pageId: string;
};
