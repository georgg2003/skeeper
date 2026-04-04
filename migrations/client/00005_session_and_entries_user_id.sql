-- +goose Up
ALTER TABLE session ADD COLUMN user_id INTEGER;
ALTER TABLE entries ADD COLUMN user_id INTEGER;
CREATE INDEX IF NOT EXISTS idx_entries_user_id ON entries(user_id);
-- +goose Down
DROP INDEX IF EXISTS idx_entries_user_id;
ALTER TABLE entries DROP COLUMN user_id;
ALTER TABLE session DROP COLUMN user_id;
