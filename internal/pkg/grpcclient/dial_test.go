package grpcclient

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"math/big"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDialOptions_InsecureWhenAllowed(t *testing.T) {
	opts, err := DialOptions(TLSConfig{Enabled: false})
	require.NoError(t, err)
	assert.Len(t, opts, 1)
}

func TestDialOptions_TLSMissingCA(t *testing.T) {
	_, err := DialOptions(TLSConfig{Enabled: true, CAFile: ""})
	require.Error(t, err, "expected error")
}

func TestDialOptions_TLSWithPEMFile(t *testing.T) {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	tpl := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(24 * time.Hour),
		KeyUsage:              x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		DNSNames:              []string{"localhost"},
	}
	der, err := x509.CreateCertificate(rand.Reader, tpl, tpl, &priv.PublicKey, priv)
	require.NoError(t, err)
	pemData := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	dir := t.TempDir()
	path := filepath.Join(dir, "ca.pem")
	require.NoError(t, os.WriteFile(path, pemData, 0o600))
	opts, err := DialOptions(TLSConfig{Enabled: true, CAFile: path})
	require.NoError(t, err)
	assert.Len(t, opts, 1)
}
