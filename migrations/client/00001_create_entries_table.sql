-- +goose Up
CREATE TABLE entries (
  uuid TEXT PRIMARY KEY,
  user_id INTEGER NOT NULL,
  type TEXT NOT NULL,
  encrypted_dek BLOB NOT NULL,
  payload BLOB NOT NULL,
  meta BLOB,
  version INTEGER NOT NULL DEFAULT 1,
  is_deleted BOOLEAN NOT NULL DEFAULT 0,
  updated_at DATETIME NOT NULL,
  is_dirty BOOLEAN NOT NULL DEFAULT 0
);
CREATE INDEX IF NOT EXISTS idx_entries_dirty_user ON entries(user_id) WHERE is_dirty = 1;
CREATE INDEX IF NOT EXISTS idx_entries_user_time ON entries(user_id, updated_at DESC);
CREATE INDEX IF NOT EXISTS idx_entries_updated_global ON entries(updated_at DESC);
-- +goose Down
DROP TABLE IF EXISTS entries;