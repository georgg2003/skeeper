-- +goose Up
CREATE TABLE IF NOT EXISTS session (
  id INTEGER PRIMARY KEY CHECK (id = 1),
  user_id INTEGER NOT NULL,
  access_token TEXT NOT NULL,
  refresh_token TEXT NOT NULL,
  refresh_expires_at DATETIME NOT NULL,
  expires_at DATETIME NOT NULL
);
-- +goose Down
DROP TABLE IF EXISTS session;