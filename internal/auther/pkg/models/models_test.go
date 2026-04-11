package models

import (
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"

	pkgerrors "github.com/georgg2003/skeeper/pkg/errors"
)

func TestUserCredentials_Validate_NormalizesEmail(t *testing.T) {
	c := UserCredentials{Email: "  Foo@EXAMPLE.com  ", Password: "long-password-ok"}
	require.NoError(t, c.Validate())
	assert.Equal(t, "foo@example.com", c.Email)
}

func TestUserCredentials_Validate_EmptyPasswordField(t *testing.T) {
	err := (&UserCredentials{Email: "a@b.c", Password: ""}).Validate()
	require.Error(t, err)
	valErr, ok := pkgerrors.AsType[*pkgerrors.ValidationError](err)
	require.True(t, ok)
	assert.Equal(t, "password", valErr.Field)
}

func TestUserCredentials_Validate_PasswordTooShort(t *testing.T) {
	err := (&UserCredentials{Email: "a@b.c", Password: "short"}).Validate()
	require.Error(t, err)
	valErr, ok := pkgerrors.AsType[*pkgerrors.ValidationError](err)
	require.True(t, ok)
	assert.Equal(t, "password", valErr.Field)
}

func TestUserCredentials_ValidateForLogin_AllowsShortPassword(t *testing.T) {
	c := UserCredentials{Email: "a@b.c", Password: "short"}
	require.NoError(t, c.ValidateForLogin())
	assert.Equal(t, "a@b.c", c.Email)
}

func TestUserCredentials_Validate_PasswordTooLong(t *testing.T) {
	long := strings.Repeat("a", MaxPasswordBytes+1)
	err := (&UserCredentials{Email: "a@b.c", Password: long}).Validate()
	require.Error(t, err)
	valErr, ok := pkgerrors.AsType[*pkgerrors.ValidationError](err)
	require.True(t, ok)
	assert.Equal(t, "password", valErr.Field)
}

func TestUserCredentials_HashPassword_CheckPassword_RoundTrip(t *testing.T) {
	c := UserCredentials{Email: "x@y.z", Password: "correct horse battery staple"}
	hash, err := c.HashPassword()
	require.NoError(t, err)
	assert.NotEmpty(t, hash)
	require.NoError(t, c.CheckPassword(hash))
}

func TestUserCredentials_CheckPassword_WrongPassword(t *testing.T) {
	c := UserCredentials{Email: "x@y.z", Password: "secret-long-12"}
	hash, err := c.HashPassword()
	require.NoError(t, err)
	wrong := UserCredentials{Email: "x@y.z", Password: "other-long-12x"}
	cmpErr := wrong.CheckPassword(hash)
	require.Error(t, cmpErr)
	assert.True(t, errors.Is(cmpErr, bcrypt.ErrMismatchedHashAndPassword))
}
