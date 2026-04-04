package vaulterror

import "errors"

var (
	ErrNotFound = errors.New("vault crypto not found")
	ErrConflict = errors.New("vault crypto already initialized with different salt or verifier")
)
