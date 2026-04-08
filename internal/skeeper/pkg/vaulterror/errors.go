// Package vaulterror holds errors for server-side vault salt and master-key fingerprint storage.
package vaulterror

import "errors"

var (
	ErrNotFound = errors.New("vault crypto not found")
	ErrConflict = errors.New("vault crypto already initialized with different salt or verifier")
)
