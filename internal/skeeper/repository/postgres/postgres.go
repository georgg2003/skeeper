package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/georgg2003/skeeper/internal/skeeper/pkg/models"
)

type Repository struct {
	pool *pgxpool.Pool
}

func (r *Repository) UpsertEntries(ctx context.Context, userID int64, entries []models.Entry) error {
	if len(entries) == 0 {
		return nil
	}

	uuids := make([]uuid.UUID, len(entries))
	types := make([]string, len(entries))
	deks := make([][]byte, len(entries))
	payloads := make([][]byte, len(entries))
	metas := make([][]byte, len(entries))
	versions := make([]int64, len(entries))
	deleted := make([]bool, len(entries))
	updatedAt := make([]time.Time, len(entries))

	for i, e := range entries {
		uuids[i] = e.UUID
		types[i] = e.Type
		deks[i] = e.EncryptedDek
		payloads[i] = e.Payload
		metas[i] = e.Meta
		versions[i] = e.Version
		deleted[i] = e.IsDeleted
		updatedAt[i] = e.UpdatedAt
	}

	query := `
		INSERT INTO entries (uuid, user_id, type, encrypted_dek, payload, meta, version, is_deleted, updated_at)
		SELECT * FROM UNNEST($1::uuid[], $2::int8[], $3::varchar[], $4::bytea[], $5::bytea[], $6::bytea[], $7::int8[], $8::boolean[], $9::timestamptz[])
		ON CONFLICT (uuid) DO UPDATE SET
			type = EXCLUDED.type,
			encrypted_dek = EXCLUDED.encrypted_dek,
			payload = EXCLUDED.payload,
			meta = EXCLUDED.meta,
			version = EXCLUDED.version,
			is_deleted = EXCLUDED.is_deleted,
			updated_at = EXCLUDED.updated_at
		WHERE entries.version < EXCLUDED.version;
	`

	userIDs := make([]int64, len(entries))
	for i := range userIDs {
		userIDs[i] = userID
	}

	_, err := r.pool.Exec(
		ctx,
		query,
		uuids,
		userIDs,
		types,
		deks,
		payloads,
		metas,
		versions,
		deleted,
		updatedAt,
	)
	return err
}

func (r *Repository) GetUpdatedAfter(ctx context.Context, userID int64, lastSync time.Time) ([]models.Entry, error) {
	query := `
		SELECT
			uuid,
			user_id,
			type,
			encrypted_dek,
			payload,
			meta,
			version,
			is_deleted,
			updated_at
		FROM entries
		WHERE user_id = $1 AND updated_at > $2
		ORDER BY updated_at ASC;
	`

	rows, err := r.pool.Query(ctx, query, userID, lastSync)
	if err != nil {
		return nil, err
	}

	dbEntries, err := pgx.CollectRows(rows, pgx.RowToStructByName[entryDB])
	if err != nil {
		return nil, err
	}
	result := make([]models.Entry, len(dbEntries))
	for i, v := range dbEntries {
		result[i] = v.toDomain()
	}

	return result, nil
}

func (r *Repository) Close() {
	r.pool.Close()
}

type PostgresConfig struct {
	Host     string `mapstructure:"host"`
	Port     uint16 `mapstructure:"port"`
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
	Database string `mapstructure:"database"`
}

func NewFromPool(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func NewFromString(ctx context.Context, connStr string) (*Repository, error) {
	pool, err := pgxpool.New(ctx, connStr)
	if err != nil {
		return nil, err
	}
	return NewFromPool(pool), nil
}

func New(ctx context.Context, cfg PostgresConfig) (*Repository, error) {
	connStr := fmt.Sprintf("postgres://%s:%s@%s:%d/%s", cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.Database)
	return NewFromString(ctx, connStr)
}
