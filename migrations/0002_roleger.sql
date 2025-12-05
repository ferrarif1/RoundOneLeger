-- Draft schema for Roledger-style database/blocks/views system.
-- Adjust types / constraints as needed before applying in production.
BEGIN;

CREATE TABLE IF NOT EXISTS tables (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    description TEXT,
    created_by TEXT,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    version INTEGER NOT NULL DEFAULT 1
);

CREATE TABLE IF NOT EXISTS properties (
    id TEXT PRIMARY KEY,
    table_id TEXT NOT NULL REFERENCES tables(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    type TEXT NOT NULL,
    options JSONB,
    relation JSONB,
    formula TEXT,
    rollup JSONB,
    sort_order INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS views (
    id TEXT PRIMARY KEY,
    table_id TEXT NOT NULL REFERENCES tables(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    layout TEXT NOT NULL,
    filters JSONB NOT NULL DEFAULT '[]'::JSONB,
    sorts JSONB NOT NULL DEFAULT '[]'::JSONB,
    "group" JSONB,
    columns JSONB NOT NULL DEFAULT '[]'::JSONB
);

CREATE TABLE IF NOT EXISTS records (
    id TEXT PRIMARY KEY,
    table_id TEXT NOT NULL REFERENCES tables(id) ON DELETE CASCADE,
    properties JSONB NOT NULL DEFAULT '{}'::JSONB,
    created_by TEXT,
    updated_by TEXT,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    version INTEGER NOT NULL DEFAULT 1,
    trashed_at TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS blocks (
    id TEXT PRIMARY KEY,
    page_id TEXT NOT NULL REFERENCES records(id) ON DELETE CASCADE,
    parent_id TEXT REFERENCES blocks(id) ON DELETE CASCADE,
    type TEXT NOT NULL,
    props JSONB NOT NULL DEFAULT '{}'::JSONB,
    "order" INTEGER NOT NULL DEFAULT 0
);

CREATE INDEX IF NOT EXISTS idx_properties_table_order ON properties(table_id, sort_order);
CREATE INDEX IF NOT EXISTS idx_views_table ON views(table_id);
CREATE INDEX IF NOT EXISTS idx_records_table ON records(table_id);
CREATE INDEX IF NOT EXISTS idx_records_trash ON records(trashed_at);
CREATE INDEX IF NOT EXISTS idx_blocks_page ON blocks(page_id, parent_id, "order");

COMMIT;
