package models

import (
	"testing"

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
