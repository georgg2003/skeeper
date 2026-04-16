package skeeper

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/georgg2003/skeeper/internal/client/pkg/models"
)

func TestClientEntryProtoRoundTrip(t *testing.T) {
	id := uuid.MustParse("6ba7b810-9dad-11d1-80b4-00c04fd430c8")
	ts := time.Unix(1700000000, 0).UTC()
	e := models.Entry{
		UUID:         id,
		Type:         models.EntryTypePassword,
		EncryptedDek: []byte{1, 2},
		Payload:      []byte{3, 4},
		Meta:         []byte{5},
		Version:      2,
		IsDeleted:    false,
		UpdatedAt:    ts,
	}
	p := clientEntryToProto(&e)
	got, err := protoToClientEntry(p)
	require.NoError(t, err)
	assert.Equal(t, e.UUID, got.UUID)
	assert.Equal(t, e.Type, got.Type)
	assert.Equal(t, e.Version, got.Version)
	assert.True(t, got.UpdatedAt.Equal(ts), "time %v vs %v", got.UpdatedAt, ts)
}

func TestProtoToClientEntry_InvalidUUID(t *testing.T) {
	p := clientEntryToProto(&models.Entry{
		UUID:      uuid.Nil,
		Type:      "X",
		UpdatedAt: time.Time{},
	})
	p.SetUuid("not-a-uuid")
	_, err := protoToClientEntry(p)
	require.Error(t, err, "expected error")
}

func TestClientEntryToProto_SetsTimestamp(t *testing.T) {
	id := uuid.New()
	ts := time.Unix(1, 0).UTC()
	p := clientEntryToProto(&models.Entry{
		UUID: id, Type: "T", UpdatedAt: ts,
	})
	require.NotNil(t, p.GetUpdatedAt())
	assert.True(t, p.GetUpdatedAt().AsTime().Equal(ts), "timestamp not set")
}
