-- +goose Up
CREATE TABLE IF NOT EXISTS entries (
  uuid UUID PRIMARY KEY,
  user_id BIGINT NOT NULL,
  type VARCHAR(50) NOT NULL,
  encrypted_dek BYTEA NOT NULL,
  payload BYTEA NOT NULL,
  meta BYTEA,
  version BIGINT NOT NULL DEFAULT 1,
  is_deleted BOOLEAN NOT NULL DEFAULT FALSE,
  updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_entries_user_sync ON entries(user_id, updated_at ASC);
-- +goose Down
DROP TABLE IF EXISTS entries;