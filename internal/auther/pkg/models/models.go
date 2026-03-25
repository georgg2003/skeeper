package models

import (
	"net/mail"
	"time"

	"github.com/georgg2003/skeeper/pkg/errors"
	"golang.org/x/crypto/bcrypt"
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

func (creds *UserCredentials) HashPassword() {
	bcrypt.GenerateFromPassword([]byte(creds.Password), bcrypt.DefaultCost)
}

type UserInfo struct {
	ID           int64
	Email        string
	PasswordHash string
}

type Token struct {
	Data      string
	ExpiresAt time.Time
}

type TokenSet struct {
	AccessToken  Token
	RefreshToken Token
}

type LoginReponse struct {
	TokenSet
	User UserInfo
}
