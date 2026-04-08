// Package utils is a few tiny helpers shared by services (e.g. hashing refresh tokens for storage).
package utils

import (
	"crypto/sha256"
	"encoding/hex"
)

func HashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}
