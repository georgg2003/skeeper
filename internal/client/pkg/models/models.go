// Package models is the CLI’s view of sessions, encrypted entries, and small payloads (e.g. cards).
package models

import (
	"time"

	"github.com/google/uuid"
)

// Entry type constants stored in Entry.Type and sent to the sync server.
const (
	EntryTypePassword = "PASSWORD"
	EntryTypeText     = "TEXT"
	EntryTypeFile     = "FILE"
	EntryTypeCard     = "CARD"
)

// Entry is a local vault row: ciphertext-only secret material plus metadata needed for sync.
type Entry struct {
	UUID         uuid.UUID // Client-generated id (v7) for offline creation and server upsert.
	Type         string    // One of EntryType* constants.
	EncryptedDek []byte    // Per-entry DEK wrapped with the user’s vault master key.
	Payload      []byte    // Type-specific secret bytes encrypted with the DEK.
	Meta         []byte    // JSON metadata encrypted with the same DEK.
	Version      int64     // Increments on each change; drives last-write-wins sync.
	IsDeleted    bool      // Soft-delete tombstone replicated to other devices.
	UpdatedAt    time.Time // Last modification time (local or merged from server).

	IsDirty bool // True when local changes are not yet confirmed by sync.

	UserID *int64 // Owning account; nil on legacy rows before per-user scoping.
}

// Session holds Auther tokens persisted on disk after login or refresh.
type Session struct {
	AccessToken      string
	RefreshToken     string
	ExpiresAt        time.Time
	RefreshExpiresAt time.Time
	UserID           *int64 // Parsed from the access JWT for local filtering; nil on legacy sessions.
}

// User is a minimal account record returned from registration/login flows.
type User struct {
	ID    int64
	Email string
}

// CardPayload is the cleartext shape marshaled into CARD entry payloads before encryption.
type CardPayload struct {
	Holder string `json:"holder,omitempty"`
	Number string `json:"number,omitempty"`
	Expiry string `json:"expiry,omitempty"`
	CVC    string `json:"cvc,omitempty"`
}
