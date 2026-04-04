package utils

import "testing"

func TestHashToken_Deterministic(t *testing.T) {
	a := HashToken("refresh-me")
	b := HashToken("refresh-me")
	if a != b {
		t.Fatal("hash not stable")
	}
	if a == HashToken("other") {
		t.Fatal("collision")
	}
}
