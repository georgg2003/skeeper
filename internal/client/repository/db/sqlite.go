package db

import (
	"context"
	"crypto/rand"
	"database/sql"
	"fmt"
	"time"

	"github.com/georgg2003/skeeper/internal/client/pkg/models"
	"github.com/google/uuid"
	"github.com/pressly/goose/v3"
	_ "modernc.org/sqlite"
)

type Repository struct {
	db *sql.DB
}

func (r *Repository) SaveEntry(ctx context.Context, e models.Entry, isDirty bool) error {
	dirtyInt := 0
	if isDirty {
		dirtyInt = 1
	}

	query := `
		INSERT INTO entries (
			uuid, type, encrypted_dek, payload, meta, version, is_deleted, updated_at, is_dirty
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(uuid) DO UPDATE SET
			type = excluded.type,
			encrypted_dek = excluded.encrypted_dek,
			payload = excluded.payload,
			meta = excluded.meta,
			version = excluded.version,
			is_deleted = excluded.is_deleted,
			updated_at = excluded.updated_at,
			is_dirty = excluded.is_dirty;
	`
	_, err := r.db.ExecContext(ctx, query,
		e.UUID.String(),
		e.Type,
		e.EncryptedDek,
		e.Payload,
		e.Meta,
		e.Version,
		e.IsDeleted,
		e.UpdatedAt,
		dirtyInt,
	)
	return err
}

func (r *Repository) GetDirtyEntries(ctx context.Context) ([]models.Entry, error) {
	query := `SELECT uuid, type, encrypted_dek, payload, meta, version, is_deleted, updated_at 
	          FROM entries WHERE is_dirty = 1`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []models.Entry
	for rows.Next() {
		var e models.Entry
		var uStr string
		err := rows.Scan(&uStr, &e.Type, &e.EncryptedDek, &e.Payload, &e.Meta, &e.Version, &e.IsDeleted, &e.UpdatedAt)
		if err != nil {
			return nil, err
		}
		e.UUID, _ = uuid.Parse(uStr)
		entries = append(entries, e)
	}
	return entries, nil
}

func (r *Repository) MarkAsSynced(ctx context.Context, u uuid.UUID) error {
	query := `UPDATE entries SET is_dirty = 0 WHERE uuid = ?`
	_, err := r.db.ExecContext(ctx, query, u.String())
	return err
}

// GetEntry returns a single entry by id.
func (r *Repository) GetEntry(ctx context.Context, id uuid.UUID) (models.Entry, error) {
	query := `SELECT uuid, type, encrypted_dek, payload, meta, version, is_deleted, updated_at, is_dirty
	          FROM entries WHERE uuid = ?`
	var e models.Entry
	var uStr string
	var dirtyInt int
	err := r.db.QueryRowContext(ctx, query, id.String()).Scan(
		&uStr, &e.Type, &e.EncryptedDek, &e.Payload, &e.Meta, &e.Version, &e.IsDeleted, &e.UpdatedAt, &dirtyInt,
	)
	if err != nil {
		return models.Entry{}, err
	}
	e.UUID, err = uuid.Parse(uStr)
	if err != nil {
		return models.Entry{}, err
	}
	e.IsDirty = dirtyInt != 0
	return e, nil
}

const kdfSaltSize = 16

// GetOrCreateKDFSalt returns the persisted Argon2id salt, generating one on first use.
func (r *Repository) GetOrCreateKDFSalt(ctx context.Context) ([]byte, error) {
	var salt []byte
	err := r.db.QueryRowContext(ctx, `SELECT kdf_salt FROM crypto_meta WHERE id = 1`).Scan(&salt)
	if err == nil && len(salt) > 0 {
		return salt, nil
	}
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}

	salt = make([]byte, kdfSaltSize)
	if _, err := rand.Read(salt); err != nil {
		return nil, fmt.Errorf("generate kdf salt: %w", err)
	}
	if err == sql.ErrNoRows {
		_, err = r.db.ExecContext(ctx, `INSERT INTO crypto_meta (id, kdf_salt) VALUES (1, ?)`, salt)
	} else {
		_, err = r.db.ExecContext(ctx, `UPDATE crypto_meta SET kdf_salt = ? WHERE id = 1`, salt)
	}
	if err != nil {
		return nil, err
	}
	return salt, nil
}

// ListEntries returns all non-deleted rows for local browsing (payload remains ciphertext).
func (r *Repository) ListEntries(ctx context.Context) ([]models.Entry, error) {
	query := `SELECT uuid, type, encrypted_dek, payload, meta, version, is_deleted, updated_at, is_dirty
	          FROM entries WHERE is_deleted = 0 ORDER BY updated_at DESC`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []models.Entry
	for rows.Next() {
		var e models.Entry
		var uStr string
		var dirtyInt int
		if err := rows.Scan(&uStr, &e.Type, &e.EncryptedDek, &e.Payload, &e.Meta, &e.Version, &e.IsDeleted, &e.UpdatedAt, &dirtyInt); err != nil {
			return nil, err
		}
		e.UUID, err = uuid.Parse(uStr)
		if err != nil {
			return nil, err
		}
		e.IsDirty = dirtyInt != 0
		out = append(out, e)
	}
	return out, rows.Err()
}

func (r *Repository) GetLastUpdate(ctx context.Context) (time.Time, error) {
	var lastUpdate time.Time
	query := `SELECT COALESCE(MAX(updated_at), '1970-01-01 00:00:00') FROM entries`
	err := r.db.QueryRowContext(ctx, query).Scan(&lastUpdate)
	return lastUpdate, err
}

func New(dsn string) (*Repository, error) {
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	return &Repository{db: db}, nil
}

// Close releases the database handle.
func (r *Repository) Close() error {
	return r.db.Close()
}

func (r *Repository) RunMigrations(ctx context.Context) error {
	goose.SetBaseFS(migrationsFS)

	if err := goose.SetDialect("sqlite3"); err != nil {
		return err
	}

	if err := goose.Up(r.db, "migrations"); err != nil {
		return fmt.Errorf("goose up: %w", err)
	}

	return nil
}
