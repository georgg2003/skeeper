-- +goose Up
CREATE TABLE crypto_meta (
  user_id INTEGER NOT NULL PRIMARY KEY,
  kdf_salt BLOB NOT NULL,
  master_verifier BLOB
);
-- +goose Down
DROP TABLE IF EXISTS crypto_meta;
