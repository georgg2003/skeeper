-- +goose Up
CREATE TABLE crypto_meta_new (
  user_id INTEGER NOT NULL PRIMARY KEY,
  kdf_salt BLOB NOT NULL,
  master_verifier BLOB
);
INSERT INTO crypto_meta_new (user_id, kdf_salt, master_verifier)
SELECT 0, kdf_salt, master_verifier FROM crypto_meta;
DROP TABLE crypto_meta;
ALTER TABLE crypto_meta_new RENAME TO crypto_meta;
-- +goose Down
CREATE TABLE crypto_meta_legacy (
  id INTEGER PRIMARY KEY CHECK (id = 1),
  kdf_salt BLOB NOT NULL,
  master_verifier BLOB NULL
);
INSERT INTO crypto_meta_legacy (id, kdf_salt, master_verifier)
SELECT 1, kdf_salt, master_verifier FROM crypto_meta WHERE user_id = 0 LIMIT 1;
DROP TABLE crypto_meta;
ALTER TABLE crypto_meta_legacy RENAME TO crypto_meta;
