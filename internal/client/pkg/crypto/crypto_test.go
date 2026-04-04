package crypto

import (
	"bytes"
	"testing"
)

func TestDeriveMasterKey_Deterministic(t *testing.T) {
	salt := []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16}
	k1 := DeriveMasterKey("same-password", salt)
	k2 := DeriveMasterKey("same-password", salt)
	if !bytes.Equal(k1, k2) {
		t.Fatal("derived keys differ for same input")
	}
	k3 := DeriveMasterKey("other-password", salt)
	if bytes.Equal(k1, k3) {
		t.Fatal("different passwords should not yield same key")
	}
}

func TestEncryptDecryptAESGCM_RoundTrip(t *testing.T) {
	key := make([]byte, KeyLength)
	for i := range key {
		key[i] = byte(i + 1)
	}
	plain := []byte("hello gophkeeper")

	ct, err := EncryptAESGCM(plain, key)
	if err != nil {
		t.Fatal(err)
	}
	got, err := DecryptAESGCM(ct, key)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, plain) {
		t.Fatalf("got %q want %q", got, plain)
	}
}

func TestDecryptAESGCM_WrongKey(t *testing.T) {
	key := make([]byte, KeyLength)
	for i := range key {
		key[i] = 7
	}
	ct, err := EncryptAESGCM([]byte("x"), key)
	if err != nil {
		t.Fatal(err)
	}
	wrong := make([]byte, KeyLength)
	wrong[0] = 1
	_, err = DecryptAESGCM(ct, wrong)
	if err == nil {
		t.Fatal("expected error decrypting with wrong key")
	}
}
