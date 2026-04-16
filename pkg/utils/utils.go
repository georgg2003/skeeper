// Package utils is a few tiny helpers shared by services (e.g. hashing refresh tokens for storage).
package utils

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"

	"golang.org/x/crypto/bcrypt"
)

func HashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}

// NewDummyHash creates a random, but valid bcrypt hash.
// It is used to simulate database work when the user is not found.
func NewDummyBcryptHash() ([]byte, error) {
	randomBytes := make([]byte, 32)
	if _, err := rand.Read(randomBytes); err != nil {
		return nil, err
	}

	dummyPassword := base64.StdEncoding.EncodeToString(randomBytes)

	hash, err := bcrypt.GenerateFromPassword([]byte(dummyPassword), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	return hash, nil
}
