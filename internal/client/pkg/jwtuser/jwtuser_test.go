package jwtuser

import (
	"testing"

	"github.com/golang-jwt/jwt/v4"
)

func TestUserIDFromAccessTokenUnverified_UserIDClaim(t *testing.T) {
	tok := jwt.NewWithClaims(jwt.SigningMethodNone, jwt.MapClaims{
		"UserID": float64(7),
	})
	s, err := tok.SignedString(jwt.UnsafeAllowNoneSignatureType)
	if err != nil {
		t.Fatal(err)
	}
	id, err := UserIDFromAccessTokenUnverified(s)
	if err != nil || id != 7 {
		t.Fatalf("got %d err %v", id, err)
	}
}

func TestUserIDFromAccessTokenUnverified_snakeClaim(t *testing.T) {
	tok := jwt.NewWithClaims(jwt.SigningMethodNone, jwt.MapClaims{
		"user_id": float64(42),
	})
	s, err := tok.SignedString(jwt.UnsafeAllowNoneSignatureType)
	if err != nil {
		t.Fatal(err)
	}
	id, err := UserIDFromAccessTokenUnverified(s)
	if err != nil || id != 42 {
		t.Fatalf("got %d err %v", id, err)
	}
}
