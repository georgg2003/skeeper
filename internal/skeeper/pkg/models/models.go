package models

import (
	"time"

	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/georgg2003/skeeper/api"
	"github.com/georgg2003/skeeper/pkg/errors"
)

type Entry struct {
	UUID         uuid.UUID
	Type         string // "credentials", "card", ...
	EncryptedDek []byte // Data encryption key, зашифрованный мастер-ключом.
	Payload      []byte // Зашифрованные данные (логин:пароль, текст и т.д.).
	Meta         []byte // Зашифрованные метаданные.
	Version      int64
	IsDeleted    bool
	UpdatedAt    time.Time
}

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

type SyncRequest struct {
	LastSyncAt time.Time
	Updates    []Entry
}

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

type SyncResponse struct {
	CurrentSyncAt time.Time
	Updates       []Entry
}

func (s *SyncResponse) ToProto() *api.SyncResponse {
	protoUpdates := make([]*api.Entry, 0, len(s.Updates))

	for i := range s.Updates {
		protoUpdates = append(protoUpdates, s.Updates[i].ToProto())
	}

	return api.SyncResponse_builder{
		CurrentSyncAt: timestamppb.New(s.CurrentSyncAt),
		Updates:       protoUpdates,
	}.Build()
}
