// Package models is the domain layer between protobuf and Skeeper usecase (entries, sync DTOs).
package models

import (
	"time"

	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/georgg2003/skeeper/api"
	"github.com/georgg2003/skeeper/pkg/errors"
)

// Entry is the server-side mirror of a client vault row (opaque ciphertext).
type Entry struct {
	UUID         uuid.UUID
	Type         string
	EncryptedDek []byte // DEK, encrypted with the user's master key.
	Payload      []byte // Ciphertext (password blob, text, etc.).
	Meta         []byte // Encrypted metadata JSON.
	Version      int64
	IsDeleted    bool
	UpdatedAt    time.Time
}

// ToProto maps the domain entry to the protobuf wire type.
func (e *Entry) ToProto() *api.Entry {
	return api.Entry_builder{
		Uuid:         e.UUID.String(),
		Type:         e.Type,
		EncryptedDek: e.EncryptedDek,
		Payload:      e.Payload,
		Meta:         e.Meta,
		Version:      e.Version,
		IsDeleted:    e.IsDeleted,
		UpdatedAt:    timestamppb.New(e.UpdatedAt),
	}.Build()
}

// NewEntryFromProto parses and validates a protobuf Entry into the domain model.
func NewEntryFromProto(e *api.Entry) (Entry, error) {
	parsedUUID, err := uuid.Parse(e.GetUuid())
	if err != nil {
		return Entry{}, errors.NewValidationError("uuid", "invalid uuid format")
	}

	out := Entry{
		UUID:         parsedUUID,
		Type:         e.GetType(),
		EncryptedDek: e.GetEncryptedDek(),
		Payload:      e.GetPayload(),
		Meta:         e.GetMeta(),
		Version:      e.GetVersion(),
		IsDeleted:    e.GetIsDeleted(),
	}
	if ts := e.GetUpdatedAt(); ts != nil {
		out.UpdatedAt = ts.AsTime()
	}
	return out, nil
}

// SyncRequest is the usecase input for one sync round-trip.
type SyncRequest struct {
	LastSyncAt time.Time
	Updates    []Entry
}

// NewSyncRequestFromProto converts the gRPC request into validated domain entries.
func NewSyncRequestFromProto(r *api.SyncRequest) (SyncRequest, error) {
	var lastSyncAt time.Time
	if r.GetLastSyncAt() != nil {
		lastSyncAt = r.GetLastSyncAt().AsTime()
	}

	protoChanges := r.GetUpdates()
	updates := make([]Entry, 0, len(protoChanges))

	for _, p := range protoChanges {
		entry, err := NewEntryFromProto(p)
		if err != nil {
			return SyncRequest{}, errors.Wrap(err, "parse entry in sync request")
		}
		updates = append(updates, entry)
	}

	return SyncRequest{
		LastSyncAt: lastSyncAt,
		Updates:    updates,
	}, nil
}

// SyncResponse is returned to the delivery layer to build SyncResponse protobuf.
type SyncResponse struct {
	CurrentSyncAt      time.Time
	Updates            []Entry
	AppliedUpdateUUIDs []uuid.UUID
}

// ToProto maps the sync result to the protobuf wire type.
func (s *SyncResponse) ToProto() *api.SyncResponse {
	protoUpdates := make([]*api.Entry, 0, len(s.Updates))

	for i := range s.Updates {
		protoUpdates = append(protoUpdates, s.Updates[i].ToProto())
	}

	applied := make([]string, 0, len(s.AppliedUpdateUUIDs))
	for i := range s.AppliedUpdateUUIDs {
		applied = append(applied, s.AppliedUpdateUUIDs[i].String())
	}

	return api.SyncResponse_builder{
		CurrentSyncAt:      timestamppb.New(s.CurrentSyncAt),
		Updates:            protoUpdates,
		AppliedUpdateUuids: applied,
	}.Build()
}
