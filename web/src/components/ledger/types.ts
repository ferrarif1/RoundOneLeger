export type WorkspaceKind = 'sheet' | 'document' | 'folder';

export type WorkspaceColumn = {
  id: string;
  title: string;
  width?: number;
};

export type WorkspaceRow = {
  id: string;
  cells: Record<string, string>;
};

export type WorkspaceNode = {
  id: string;
  name: string;
  kind: WorkspaceKind;
  parentId?: string | null;
  createdAt?: string;
  updatedAt?: string;
  children?: WorkspaceNode[];
  rowCount?: number;
};

export type Workspace = {
  id: string;
  name: string;
  kind: WorkspaceKind;
  parentId?: string | null;
  columns: WorkspaceColumn[];
  rows: WorkspaceRow[];
  document?: string;
  createdAt?: string;
  updatedAt?: string;
};
