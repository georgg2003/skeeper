-- +goose Up
CREATE TABLE crypto_meta (
  id INTEGER PRIMARY KEY CHECK (id = 1),
  kdf_salt BLOB NOT NULL
);
-- +goose Down
DROP TABLE IF EXISTS crypto_meta;
