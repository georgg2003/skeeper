-- +goose Up
CREATE TABLE entries (
  uuid TEXT PRIMARY KEY,
  type TEXT NOT NULL,
  encrypted_dek BLOB NOT NULL,
  payload BLOB NOT NULL,
  meta BLOB,
  version INTEGER NOT NULL DEFAULT 1,
  is_deleted BOOLEAN NOT NULL DEFAULT 0,
  updated_at DATETIME NOT NULL,
  is_dirty BOOLEAN NOT NULL DEFAULT 0
);
CREATE INDEX IF NOT EXISTS idx_entries_dirty ON entries(is_dirty)
WHERE is_dirty = 1;
-- +goose Down
DROP TABLE IF EXISTS entries;