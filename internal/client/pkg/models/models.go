// Package models defines client-side domain types for sessions and encrypted entries.
package models

import (
	"time"

	"github.com/google/uuid"
)

// Entry type constants sent to Skeeper (cleartext type label only; payload stays encrypted).
const (
	EntryTypePassword = "PASSWORD"
	EntryTypeText     = "TEXT"
	EntryTypeBinary   = "BINARY"
	EntryTypeCard     = "CARD"
)

// Entry is one ciphertext blob cached locally and synchronized with the server.
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

// Session holds JWT material returned by Auther.
type Session struct {
	AccessToken      string
	RefreshToken     string
	ExpiresAt        time.Time // access token expiry
	RefreshExpiresAt time.Time // refresh token expiry
}

// User is a minimal account projection (used after registration).
type User struct {
	ID    int64
	Email string
}

// CardPayload is serialized into the ciphertext payload for CARD entries.
type CardPayload struct {
	Holder string `json:"holder,omitempty"`
	Number string `json:"number,omitempty"`
	Expiry string `json:"expiry,omitempty"`
	CVC    string `json:"cvc,omitempty"`
}
