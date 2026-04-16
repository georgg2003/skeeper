// Package db is the local SQLite vault: entries, session row, crypto metadata, goose migrations.
package db

import (
	"context"
	"crypto/rand"
	"database/sql"
	"fmt"
	"path/filepath"
	"time"

	clientmigrate "github.com/georgg2003/skeeper/migrations/client"
	"github.com/georgg2003/skeeper/pkg/errors"

	"github.com/google/uuid"
	"github.com/pressly/goose/v3"
	_ "modernc.org/sqlite"

	"github.com/georgg2003/skeeper/internal/client/pkg/models"
)

type sqlExecer interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

type Repository struct {
	db         *sql.DB
	sessionKey []byte
}

func (r *Repository) SaveEntry(ctx context.Context, e models.Entry, isDirty bool) error {
	return r.saveEntry(ctx, r.db, e, isDirty)
}

func (r *Repository) saveEntry(ctx context.Context, ex sqlExecer, e models.Entry, isDirty bool) error {
	dirtyInt := 0
	if isDirty {
		dirtyInt = 1
	}

	query := `
		INSERT INTO entries (
			uuid, type, encrypted_dek, payload, meta, version, is_deleted, updated_at, is_dirty, user_id
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(uuid) DO UPDATE SET
			type = excluded.type,
			encrypted_dek = excluded.encrypted_dek,
			payload = excluded.payload,
			meta = excluded.meta,
			version = excluded.version,
			is_deleted = excluded.is_deleted,
			updated_at = excluded.updated_at,
			is_dirty = excluded.is_dirty,
			user_id = COALESCE(excluded.user_id, entries.user_id);
	`
	var uid any
	if e.UserID != nil {
		uid = *e.UserID
	}
	_, err := ex.ExecContext(ctx, query,
		e.UUID.String(),
		e.Type,
		e.EncryptedDek,
		e.Payload,
		e.Meta,
		e.Version,
		e.IsDeleted,
		e.UpdatedAt,
		dirtyInt,
		uid,
	)
	return err
}

// PersistSyncResult applies remote updates and marks applied dirty rows clean in one transaction.
func (r *Repository) PersistSyncResult(
	ctx context.Context,
	userID int64,
	updates []models.Entry,
	dirty []models.Entry,
	applied map[uuid.UUID]struct{},
) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	for i := range updates {
		e := updates[i]
		u := userID
		e.UserID = &u
		if err := r.saveEntry(ctx, tx, e, false); err != nil {
			return err
		}
	}
	for _, e := range dirty {
		if _, ok := applied[e.UUID]; !ok {
			continue
		}
		if _, err := tx.ExecContext(ctx, `UPDATE entries SET is_dirty = 0 WHERE uuid = ? AND user_id = ?`,
			e.UUID.String(), userID); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (r *Repository) GetDirtyEntries(ctx context.Context, forUserID *int64) ([]models.Entry, error) {
	query := `SELECT uuid, type, encrypted_dek, payload, meta, version, is_deleted, updated_at, user_id
	          FROM entries WHERE is_dirty = 1`
	var args []any
	if forUserID != nil {
		query += ` AND user_id = ?`
		args = append(args, *forUserID)
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var entries []models.Entry
	for rows.Next() {
		var e models.Entry
		var uStr string
		var userID sql.NullInt64
		err := rows.Scan(&uStr, &e.Type, &e.EncryptedDek, &e.Payload, &e.Meta, &e.Version, &e.IsDeleted, &e.UpdatedAt, &userID)
		if err != nil {
			return nil, errors.Wrap(err, "entries scan error")
		}
		e.UUID, err = uuid.Parse(uStr)
		if err != nil {
			return nil, errors.Wrapf(err, "entries uuid %q parse error", uStr)
		}
		if userID.Valid {
			v := userID.Int64
			e.UserID = &v
		}
		entries = append(entries, e)
	}
	if err = rows.Err(); err != nil {
		return nil, errors.Wrap(err, "entries rows error")
	}
	return entries, nil
}

func (r *Repository) MarkAsSynced(ctx context.Context, u uuid.UUID, userID int64) error {
	_, err := r.db.ExecContext(ctx, `UPDATE entries SET is_dirty = 0 WHERE uuid = ? AND user_id = ?`, u.String(), userID)
	return err
}

func (r *Repository) GetEntry(ctx context.Context, id uuid.UUID, forUserID *int64) (models.Entry, error) {
	query := `SELECT uuid, type, encrypted_dek, payload, meta, version, is_deleted, updated_at, is_dirty, user_id
	          FROM entries WHERE uuid = ?`
	var args []any
	args = append(args, id.String())
	if forUserID != nil {
		query += ` AND user_id = ?`
		args = append(args, *forUserID)
	}
	var e models.Entry
	var uStr string
	var dirtyInt int
	var userID sql.NullInt64
	err := r.db.QueryRowContext(ctx, query, args...).Scan(
		&uStr, &e.Type, &e.EncryptedDek, &e.Payload, &e.Meta, &e.Version, &e.IsDeleted, &e.UpdatedAt, &dirtyInt, &userID,
	)
	if err != nil {
		return models.Entry{}, err
	}
	e.UUID, err = uuid.Parse(uStr)
	if err != nil {
		return models.Entry{}, err
	}
	e.IsDirty = dirtyInt != 0
	if userID.Valid {
		v := userID.Int64
		e.UserID = &v
	}
	return e, nil
}

const kdfSaltSize = 16

// EnsureLocalVaultCrypto loads salt (+ verifier) for this Auther user, or creates a new row.
func (r *Repository) EnsureLocalVaultCrypto(ctx context.Context, userID int64) (salt []byte, masterVerifier []byte, err error) {
	err = r.db.QueryRowContext(ctx, `SELECT kdf_salt, master_verifier FROM crypto_meta WHERE user_id = ?`, userID).
		Scan(&salt, &masterVerifier)
	if err == nil {
		return salt, masterVerifier, nil
	}
	if err != sql.ErrNoRows {
		return nil, nil, err
	}
	salt = make([]byte, kdfSaltSize)
	if _, err := rand.Read(salt); err != nil {
		return nil, nil, fmt.Errorf("generate kdf salt: %w", err)
	}
	_, err = r.db.ExecContext(ctx, `INSERT INTO crypto_meta (user_id, kdf_salt, master_verifier) VALUES (?, ?, NULL)`, userID, salt)
	if err != nil {
		return nil, nil, err
	}
	return salt, nil, nil
}

func (r *Repository) ReplaceLocalVaultCrypto(ctx context.Context, userID int64, salt, masterVerifier []byte) error {
	var ver any
	if len(masterVerifier) > 0 {
		ver = masterVerifier
	}
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO crypto_meta (user_id, kdf_salt, master_verifier) VALUES (?, ?, ?)
		ON CONFLICT(user_id) DO UPDATE SET
			kdf_salt = excluded.kdf_salt,
			master_verifier = excluded.master_verifier
	`, userID, salt, ver)
	return err
}

func (r *Repository) SetLocalMasterVerifier(ctx context.Context, userID int64, masterVerifier []byte) error {
	_, err := r.db.ExecContext(ctx, `UPDATE crypto_meta SET master_verifier = ? WHERE user_id = ?`, masterVerifier, userID)
	return err
}

func (r *Repository) ListEntries(ctx context.Context, forUserID *int64) ([]models.Entry, error) {
	query := `SELECT uuid, type, encrypted_dek, payload, meta, version, is_deleted, updated_at, is_dirty, user_id
	          FROM entries WHERE is_deleted = 0`
	var args []any
	if forUserID != nil {
		query += ` AND user_id = ?`
		args = append(args, *forUserID)
	}
	query += ` ORDER BY updated_at DESC`
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var out []models.Entry
	for rows.Next() {
		var e models.Entry
		var uStr string
		var dirtyInt int
		var userID sql.NullInt64
		if err := rows.Scan(&uStr, &e.Type, &e.EncryptedDek, &e.Payload, &e.Meta, &e.Version, &e.IsDeleted, &e.UpdatedAt, &dirtyInt, &userID); err != nil {
			return nil, err
		}
		e.UUID, err = uuid.Parse(uStr)
		if err != nil {
			return nil, err
		}
		e.IsDirty = dirtyInt != 0
		if userID.Valid {
			v := userID.Int64
			e.UserID = &v
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

func (r *Repository) GetLastUpdate(ctx context.Context, forUserID *int64) (time.Time, error) {
	var lastUpdateInt int64
	query := `SELECT COALESCE(unixepoch(MAX(updated_at)), 0) FROM entries`
	var args []any
	if forUserID != nil {
		query += ` WHERE user_id = ?`
		args = append(args, *forUserID)
	}

	err := r.db.QueryRowContext(ctx, query, args...).Scan(&lastUpdateInt)
	if err != nil {
		return time.Time{}, err
	}

	return time.Unix(lastUpdateInt, 0), nil
}

func New(dsn string) (*Repository, error) {
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	keyPath := filepath.Join(filepath.Dir(dsn), ".session-key")
	key, err := loadOrCreateSessionKey(keyPath)
	if err != nil {
		_ = db.Close()
		return nil, err
	}
	return &Repository{db: db, sessionKey: key}, nil
}

func (r *Repository) Close() error {
	return r.db.Close()
}

func (r *Repository) RunMigrations(ctx context.Context) error {
	goose.SetBaseFS(clientmigrate.ClientMigrationsFS)

	if err := goose.SetDialect("sqlite3"); err != nil {
		return err
	}

	if err := goose.UpContext(ctx, r.db, "."); err != nil {
		return fmt.Errorf("goose up: %w", err)
	}

	return nil
}
