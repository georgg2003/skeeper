package models

import (
	"time"

	"github.com/google/uuid"
)

type Entry struct {
	UUID         uuid.UUID
	Type         string
	EncryptedDek []byte
	Payload      []byte
	Meta         []byte
	Version      int64
	IsDeleted    bool
	UpdatedAt    time.Time

	IsDirty bool
}

type Session struct {
	AccessToken  string
	RefreshToken string
	ExpiresAt    time.Time
}
