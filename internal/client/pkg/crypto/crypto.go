// Package crypto derives the master key (Argon2id) and does AES-GCM for the local vault.
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
	// KeyLength is the AES-256 key size in bytes for master keys and DEKs.
	KeyLength = 32
	// SaltSize is the KDF salt length in bytes for vault Argon2id derivation.
	SaltSize = 16
)

// DeriveMasterKey returns a 32-byte key from the vault master password and salt using Argon2id.
func DeriveMasterKey(password string, salt []byte) []byte {
	return argon2.IDKey([]byte(password), salt, 3, 64*1024, 4, KeyLength)
}

// MasterKeyVerifier is SHA-256 of the derived master key—stored so we can check the password
// without keeping the key itself on disk or sending it over the wire.
func MasterKeyVerifier(masterKey []byte) []byte {
	h := sha256.Sum256(masterKey)
	out := make([]byte, len(h))
	copy(out, h[:])
	return out
}

// EncryptAESGCM encrypts data with AES-GCM; the ciphertext prefix is the random nonce.
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

// DecryptAESGCM decrypts a blob produced by EncryptAESGCM.
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
