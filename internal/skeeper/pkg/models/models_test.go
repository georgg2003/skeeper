package models

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/georgg2003/skeeper/api"
)

func TestEntryProtoRoundTrip(t *testing.T) {
	id := uuid.MustParse("6ba7b810-9dad-11d1-80b4-00c04fd430c8")
	ts := time.Unix(1700000000, 0).UTC()
	e := Entry{
		UUID:         id,
		Type:         "PASSWORD",
		EncryptedDek: []byte{9},
		Payload:      []byte{8},
		Meta:         []byte{7},
		Version:      3,
		IsDeleted:    true,
		UpdatedAt:    ts,
	}
	p := e.ToProto()
	got, err := NewEntryFromProto(p)
	require.NoError(t, err)
	assert.Equal(t, e.UUID, got.UUID)
	assert.Equal(t, e.Type, got.Type)
	assert.Equal(t, e.Version, got.Version)
	assert.Equal(t, e.IsDeleted, got.IsDeleted)
	assert.True(t, got.UpdatedAt.Equal(ts), "time mismatch")
}

func TestNewEntryFromProto_InvalidUUID(t *testing.T) {
	p := (&Entry{UUID: uuid.New(), UpdatedAt: time.Now()}).ToProto()
	p.SetUuid("bad")
	_, err := NewEntryFromProto(p)
	require.Error(t, err, "expected validation error")
}

func TestNewSyncRequestFromProto(t *testing.T) {
	id := uuid.New()
	ts := time.Unix(123, 0).UTC()
	e := Entry{UUID: id, Type: "T", UpdatedAt: ts}
	p := api.SyncRequest_builder{
		LastSyncAt: timestamppb.New(ts),
		Updates:    []*api.Entry{e.ToProto()},
	}.Build()
	got, err := NewSyncRequestFromProto(p)
	require.NoError(t, err)
	assert.True(t, got.LastSyncAt.Equal(ts))
	assert.Len(t, got.Updates, 1)
}

func TestSyncResponse_ToProto(t *testing.T) {
	ts := time.Unix(999, 0).UTC()
	id := uuid.New()
	s := SyncResponse{
		CurrentSyncAt:      ts,
		AppliedUpdateUUIDs: []uuid.UUID{id},
		Updates: []Entry{
			{UUID: id, Type: "P", UpdatedAt: ts},
		},
	}
	p := s.ToProto()
	require.NotNil(t, p.GetCurrentSyncAt())
	assert.Len(t, p.GetUpdates(), 1)
	ap := p.GetAppliedUpdateUuids()
	require.Len(t, ap, 1)
	assert.Equal(t, id.String(), ap[0])
}
