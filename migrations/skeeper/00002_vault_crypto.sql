-- +goose Up
CREATE TABLE IF NOT EXISTS vault_crypto (
  user_id BIGINT PRIMARY KEY,
  kdf_salt BYTEA NOT NULL,
  master_verifier BYTEA NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
-- +goose Down
DROP TABLE IF EXISTS vault_crypto;
