package models

import (
	"net/mail"

	"golang.org/x/crypto/bcrypt"

	"github.com/georgg2003/skeeper/pkg/errors"
	"github.com/georgg2003/skeeper/pkg/jwthelper"
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
		return errors.NewValidationError("password", "password must not be empty")
	}
	return nil
}

func (creds *UserCredentials) Validate() error {
	return errors.Join(creds.validateEmail(), creds.validatePassword())
}

func (creds *UserCredentials) HashPassword() ([]byte, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(creds.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}
	return hash, nil
}

func (creds *UserCredentials) CheckPassword(passwordHash []byte) error {
	return bcrypt.CompareHashAndPassword(passwordHash, []byte(creds.Password))
}

type DBUserCredentials struct {
	Email        string
	PasswordHash []byte
}

type UserInfo struct {
	ID           int64
	Email        string
	PasswordHash []byte
}

type RefreshTokenHashed struct {
	jwthelper.Token
	Hash string
}

type LoginReponse struct {
	TokenPair jwthelper.TokenPair
	User      UserInfo
}
