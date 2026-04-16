package db

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"os"

	"github.com/georgg2003/skeeper/internal/client/pkg/crypto"
)

func loadOrCreateSessionKey(keyPath string) ([]byte, error) {
	b, err := os.ReadFile(keyPath)
	if err == nil {
		if len(b) != crypto.KeyLength {
			return nil, fmt.Errorf("session key file %s: want %d bytes, got %d", keyPath, crypto.KeyLength, len(b))
		}
		return b, nil
	}
	if !os.IsNotExist(err) {
		return nil, fmt.Errorf("read session key: %w", err)
	}
	key := make([]byte, crypto.KeyLength)
	if _, err := io.ReadFull(rand.Reader, key); err != nil {
		return nil, fmt.Errorf("generate session key: %w", err)
	}
	if err := os.WriteFile(keyPath, key, 0o600); err != nil {
		return nil, fmt.Errorf("write session key: %w", err)
	}
	return key, nil
}

// encryptSessionToken stores ciphertext as raw std base64 (no version prefix).
func encryptSessionToken(plaintext string, key []byte) (string, error) {
	if plaintext == "" {
		return "", nil
	}
	out, err := crypto.EncryptAESGCM([]byte(plaintext), key)
	if err != nil {
		return "", err
	}
	return base64.RawStdEncoding.EncodeToString(out), nil
}

func decryptSessionToken(encoded string, key []byte) (string, error) {
	if encoded == "" {
		return "", nil
	}
	raw, err := base64.RawStdEncoding.DecodeString(encoded)
	if err != nil {
		return "", fmt.Errorf("session token decode: %w", err)
	}
	plain, err := crypto.DecryptAESGCM(raw, key)
	if err != nil {
		return "", fmt.Errorf("session token decrypt: %w", err)
	}
	return string(plain), nil
}
