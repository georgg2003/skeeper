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
	if MustGetUserID(ctx) != 42 {
		t.Fatal("must get")
	}
}

func TestMustGetUserID_Panic(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("expected panic")
		}
	}()
	_ = MustGetUserID(context.Background())
}
