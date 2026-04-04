package contextlib

import (
	"context"
	"testing"
)

func TestUserIDRoundTrip(t *testing.T) {
	ctx := SetUserID(context.Background(), 42)
	id, ok := GetUserID(ctx)
	if !ok || id != 42 {
		t.Fatalf("got %d ok=%v", id, ok)
	}
}

func TestGetUserID_Missing(t *testing.T) {
	_, ok := GetUserID(context.Background())
	if ok {
		t.Fatal("expected no user id")
	}
}
