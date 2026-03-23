package models

import (
	"net/mail"

	"github.com/georgg2003/skeeper/pkg/errors"
)

type UserCredentials struct {
	Email    string
	Password string
}

var ErrEmptyPassword = errors.New("password should not be empty")

func (creds *UserCredentials) validateEmail() error {
	if _, err := mail.ParseAddress(creds.Email); err != nil {
		return errors.NewValidationError("email", err.Error())
	}
	return nil
}

func (creds *UserCredentials) validatePassword() error {
	if creds.Password == "" {
		return errors.NewValidationError("email", "password must not be empty")
	}
	return nil
}

func (creds *UserCredentials) Validate() error {
	return errors.Join(creds.validateEmail(), creds.validatePassword())
}

type UserInfo struct {
	ID    int64
	Email string
}
