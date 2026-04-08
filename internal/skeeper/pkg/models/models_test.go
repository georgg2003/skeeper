package models

import (
	"testing"
	"time"

	"github.com/google/uuid"
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
	if err != nil {
		t.Fatal(err)
	}
	if got.UUID != e.UUID || got.Type != e.Type || got.Version != e.Version || got.IsDeleted != e.IsDeleted {
		t.Fatalf("mismatch %+v vs %+v", got, e)
	}
	if !got.UpdatedAt.Equal(ts) {
		t.Fatal("time mismatch")
	}
}

func TestNewEntryFromProto_InvalidUUID(t *testing.T) {
	p := (&Entry{UUID: uuid.New(), UpdatedAt: time.Now()}).ToProto()
	p.SetUuid("bad")
	_, err := NewEntryFromProto(p)
	if err == nil {
		t.Fatal("expected validation error")
	}
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
	if err != nil {
		t.Fatal(err)
	}
	if !got.LastSyncAt.Equal(ts) || len(got.Updates) != 1 {
		t.Fatalf("%+v", got)
	}
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
	if p.GetCurrentSyncAt() == nil || len(p.GetUpdates()) != 1 {
		t.Fatal("bad proto")
	}
	ap := p.GetAppliedUpdateUuids()
	if len(ap) != 1 || ap[0] != id.String() {
		t.Fatalf("applied uuids %+v", ap)
	}
}
