BEGIN;

-- Records table indexes to speed pagination/filters
CREATE INDEX IF NOT EXISTS idx_records_table_updated ON records(table_id, updated_at DESC);
CREATE INDEX IF NOT EXISTS idx_records_table_trash ON records(table_id, trashed_at);
-- Generic JSONB GIN index to accelerate property filters
CREATE INDEX IF NOT EXISTS idx_records_properties_gin ON records USING gin (properties);
-- Example expression index for common property (adjust as needed)
-- 请根据实际字段启用或新增以下索引
-- CREATE INDEX IF NOT EXISTS idx_records_prop_name ON records ((properties->>'name'));
-- CREATE INDEX IF NOT EXISTS idx_records_prop_type ON records ((properties->>'type'));

-- Views/properties/import_tasks helper indexes
CREATE INDEX IF NOT EXISTS idx_views_table ON views(table_id);
CREATE INDEX IF NOT EXISTS idx_properties_table ON properties(table_id);
CREATE INDEX IF NOT EXISTS idx_import_tasks_table ON import_tasks(table_id);
CREATE INDEX IF NOT EXISTS idx_import_tasks_status ON import_tasks(status);

COMMIT;
