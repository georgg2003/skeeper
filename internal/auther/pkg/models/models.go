// Package models holds credentials validation, password hashing, and DTOs for the Auther usecase.
package models

import (
	"net/mail"
	"strings"
	"unicode/utf8"

	"golang.org/x/crypto/bcrypt"

	"github.com/georgg2003/skeeper/pkg/errors"
	"github.com/georgg2003/skeeper/pkg/jwthelper"
)

type UserCredentials struct {
	Email    string
	Password string
}

const (
	// MinPasswordLength is the minimum accepted password length (bcrypt truncates beyond 72 bytes).
	MinPasswordLength = 8
	// MaxPasswordBytes is the bcrypt input limit; longer passwords are rejected to match hashing behavior.
	MaxPasswordBytes = 72
)

var ErrEmptyPassword = errors.New("password should not be empty")

func normalizeEmail(raw string) (string, error) {
	addr, err := mail.ParseAddress(strings.TrimSpace(raw))
	if err != nil {
		return "", err
	}
	return strings.ToLower(strings.TrimSpace(addr.Address)), nil
}

func (creds *UserCredentials) validateEmail() error {
	norm, err := normalizeEmail(creds.Email)
	if err != nil {
		return errors.NewValidationError("email", err.Error())
	}
	creds.Email = norm
	return nil
}

func (creds *UserCredentials) validatePassword() error {
	if creds.Password == "" {
		return errors.NewValidationError("password", "password must not be empty")
	}
	if utf8.RuneCountInString(creds.Password) < MinPasswordLength {
		return errors.NewValidationError("password", "password is too short")
	}
	if len(creds.Password) > MaxPasswordBytes {
		return errors.NewValidationError("password", "password is too long")
	}
	return nil
}

func (creds *UserCredentials) Validate() error {
	if err := creds.validatePassword(); err != nil {
		return err
	}
	return creds.validateEmail()
}

// ValidateForLogin checks email and non-empty password only (no minimum length), so existing
// accounts created before password policy tightening can still sign in.
func (creds *UserCredentials) ValidateForLogin() error {
	if creds.Password == "" {
		return errors.NewValidationError("password", "password must not be empty")
	}
	if len(creds.Password) > MaxPasswordBytes {
		return errors.NewValidationError("password", "password is too long")
	}
	return creds.validateEmail()
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

type LoginResponse struct {
	TokenPair jwthelper.TokenPair
	User      UserInfo
}
