package crypto

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMasterKeyVerifier_Deterministic(t *testing.T) {
	key := make([]byte, KeyLength)
	for i := range key {
		key[i] = byte(i)
	}
	v1 := MasterKeyVerifier(key)
	v2 := MasterKeyVerifier(key)
	assert.True(t, bytes.Equal(v1, v2))
	assert.Len(t, v1, 32)
}

func TestDeriveMasterKey_Deterministic(t *testing.T) {
	salt := []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16}
	k1 := DeriveMasterKey("same-password", salt)
	k2 := DeriveMasterKey("same-password", salt)
	assert.True(t, bytes.Equal(k1, k2), "derived keys differ for same input")
	k3 := DeriveMasterKey("other-password", salt)
	assert.False(t, bytes.Equal(k1, k3), "different passwords should not yield same key")
}

func TestEncryptDecryptAESGCM_RoundTrip(t *testing.T) {
	key := make([]byte, KeyLength)
	for i := range key {
		key[i] = byte(i + 1)
	}
	plain := []byte("hello skeeper")

	ct, err := EncryptAESGCM(plain, key)
	require.NoError(t, err)
	got, err := DecryptAESGCM(ct, key)
	require.NoError(t, err)
	assert.Equal(t, plain, got)
}

func TestDecryptAESGCM_WrongKey(t *testing.T) {
	key := make([]byte, KeyLength)
	for i := range key {
		key[i] = 7
	}
	ct, err := EncryptAESGCM([]byte("x"), key)
	require.NoError(t, err)
	wrong := make([]byte, KeyLength)
	wrong[0] = 1
	_, err = DecryptAESGCM(ct, wrong)
	require.Error(t, err, "expected error decrypting with wrong key")
}
