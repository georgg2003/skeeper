// Package crypto provides Argon2id-based key derivation and AES-GCM for the local vault.
package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"io"

	"golang.org/x/crypto/argon2"
)

const (
	KeyLength = 32 // AES-256
	SaltSize  = 16
)

func DeriveMasterKey(password string, salt []byte) []byte {
	return argon2.IDKey([]byte(password), salt, 3, 64*1024, 4, KeyLength)
}

// MasterKeyVerifier returns SHA-256(masterKey) for storing on the server and locally.
// It does not reveal the master password and allows a constant-time check before encrypt/decrypt.
func MasterKeyVerifier(masterKey []byte) []byte {
	h := sha256.Sum256(masterKey)
	out := make([]byte, len(h))
	copy(out, h[:])
	return out
}

func EncryptAESGCM(data []byte, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	return gcm.Seal(nonce, nonce, data, nil), nil
}

func DecryptAESGCM(cipherText []byte, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonceSize := gcm.NonceSize()
	if len(cipherText) < nonceSize {
		return nil, fmt.Errorf("ciphertext too short")
	}

	nonce, actualCiphertext := cipherText[:nonceSize], cipherText[nonceSize:]
	return gcm.Open(nil, nonce, actualCiphertext, nil)
}
