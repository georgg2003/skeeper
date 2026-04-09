// Package models is the CLI’s view of sessions, encrypted entries, and small payloads (e.g. cards).
package models

import (
	"time"

	"github.com/google/uuid"
)

const (
	EntryTypePassword = "PASSWORD"
	EntryTypeText     = "TEXT"
	EntryTypeFile     = "FILE"
	EntryTypeCard     = "CARD"
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

	UserID *int64 // nil on old rows before per-user scoping
}

type Session struct {
	AccessToken      string
	RefreshToken     string
	ExpiresAt        time.Time
	RefreshExpiresAt time.Time
	UserID           *int64 // parsed from JWT when we can; nil on legacy sessions
}

type User struct {
	ID    int64
	Email string
}

type CardPayload struct {
	Holder string `json:"holder,omitempty"`
	Number string `json:"number,omitempty"`
	Expiry string `json:"expiry,omitempty"`
	CVC    string `json:"cvc,omitempty"`
}
