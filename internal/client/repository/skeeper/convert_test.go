package skeeper

import (
	"testing"
	"time"

	"github.com/georgg2003/skeeper/internal/client/pkg/models"
	"github.com/google/uuid"
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
	if err != nil {
		t.Fatal(err)
	}
	if got.UUID != e.UUID || got.Type != e.Type || got.Version != e.Version {
		t.Fatalf("mismatch: %+v vs %+v", got, e)
	}
	if !got.UpdatedAt.Equal(ts) {
		t.Fatalf("time %v vs %v", got.UpdatedAt, ts)
	}
}

func TestProtoToClientEntry_InvalidUUID(t *testing.T) {
	p := clientEntryToProto(&models.Entry{
		UUID:      uuid.Nil,
		Type:      "X",
		UpdatedAt: time.Time{},
	})
	p.SetUuid("not-a-uuid")
	_, err := protoToClientEntry(p)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestClientEntryToProto_SetsTimestamp(t *testing.T) {
	id := uuid.New()
	ts := time.Unix(1, 0).UTC()
	p := clientEntryToProto(&models.Entry{
		UUID: id, Type: "T", UpdatedAt: ts,
	})
	if p.GetUpdatedAt() == nil || !p.GetUpdatedAt().AsTime().Equal(ts) {
		t.Fatal("timestamp not set")
	}
}

