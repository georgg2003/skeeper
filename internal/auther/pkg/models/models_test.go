package models

import (
	"errors"
	"testing"

	"golang.org/x/crypto/bcrypt"

	pkgerrors "github.com/georgg2003/skeeper/pkg/errors"
)

func TestUserCredentials_Validate_EmptyPasswordField(t *testing.T) {
	err := (&UserCredentials{Email: "a@b.c", Password: ""}).Validate()
	if err == nil {
		t.Fatal("expected error")
	}
	valErr, ok := pkgerrors.AsType[*pkgerrors.ValidationError](err)
	if !ok || valErr.Field != "password" {
		t.Fatalf("got %v", err)
	}
}

func TestUserCredentials_HashPassword_CheckPassword_RoundTrip(t *testing.T) {
	c := UserCredentials{Email: "x@y.z", Password: "correct horse battery staple"}
	hash, err := c.HashPassword()
	if err != nil {
		t.Fatal(err)
	}
	if len(hash) == 0 {
		t.Fatal("empty hash")
	}
	if err := c.CheckPassword(hash); err != nil {
		t.Fatal(err)
	}
}

func TestUserCredentials_CheckPassword_WrongPassword(t *testing.T) {
	c := UserCredentials{Email: "x@y.z", Password: "secret"}
	hash, err := c.HashPassword()
	if err != nil {
		t.Fatal(err)
	}
	wrong := UserCredentials{Email: "x@y.z", Password: "other"}
	cmpErr := wrong.CheckPassword(hash)
	if cmpErr == nil {
		t.Fatal("expected compare error")
	}
	if !errors.Is(cmpErr, bcrypt.ErrMismatchedHashAndPassword) {
		t.Fatalf("want bcrypt.ErrMismatchedHashAndPassword, got %v", cmpErr)
	}
}
