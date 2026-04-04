-- migrate:kind=normal
CREATE TABLE IF NOT EXISTS module_states (
    module_id TEXT PRIMARY KEY,
    enabled INTEGER NOT NULL,
    updated_at INTEGER NOT NULL,
    updated_by INTEGER
);
