package postgres

import (
	"time"

	"github.com/google/uuid"

	"github.com/georgg2003/skeeper/internal/skeeper/pkg/models"
)

type entryDB struct {
	UUID         uuid.UUID `db:"uuid"`
	UserID       int64     `db:"user_id"`
	Type         string    `db:"type"`
	EncryptedDek []byte    `db:"encrypted_dek"`
	Payload      []byte    `db:"payload"`
	Meta         []byte    `db:"meta"`
	Version      int64     `db:"version"`
	IsDeleted    bool      `db:"is_deleted"`
	UpdatedAt    time.Time `db:"updated_at"`
}

func (e entryDB) toDomain() models.Entry {
	return models.Entry{
		UUID:         e.UUID,
		Type:         e.Type,
		EncryptedDek: e.EncryptedDek,
		Payload:      e.Payload,
		Meta:         e.Meta,
		Version:      e.Version,
		IsDeleted:    e.IsDeleted,
		UpdatedAt:    e.UpdatedAt,
	}
}
