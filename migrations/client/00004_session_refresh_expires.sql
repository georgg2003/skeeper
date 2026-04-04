-- +goose Up
ALTER TABLE session ADD COLUMN refresh_expires_at DATETIME;
-- Legacy rows: approximate refresh expiry from access expiry (access ~15m, refresh ~30d from issue).
UPDATE session
SET refresh_expires_at = datetime(expires_at, '-15 minutes', '+30 days')
WHERE refresh_expires_at IS NULL AND id = 1;
-- +goose Down
ALTER TABLE session DROP COLUMN refresh_expires_at;
