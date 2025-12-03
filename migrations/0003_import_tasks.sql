BEGIN;

CREATE TABLE IF NOT EXISTS import_tasks (
    id TEXT PRIMARY KEY,
    table_id TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending',
    progress INTEGER NOT NULL DEFAULT 0,
    error TEXT,
    payload_path TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_import_tasks_status ON import_tasks(status);
CREATE INDEX IF NOT EXISTS idx_import_tasks_table ON import_tasks(table_id);

COMMIT;
