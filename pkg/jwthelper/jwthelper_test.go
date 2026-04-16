package jwthelper

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func rsaTestKeys(t *testing.T) (privPEM, pubPEM []byte) {
	t.Helper()
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	privDER := x509.MarshalPKCS1PrivateKey(priv)
	privPEM = pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: privDER})
	pubDER, err := x509.MarshalPKIXPublicKey(&priv.PublicKey)
	require.NoError(t, err)
	pubPEM = pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pubDER})
	return privPEM, pubPEM
}

func TestNew_NewTokenPair_ValidateToken(t *testing.T) {
	privPEM, pubPEM := rsaTestKeys(t)
	h, err := New(privPEM, pubPEM, time.Minute, time.Hour, "")
	require.NoError(t, err)
	pair, err := h.NewTokenPair(99)
	require.NoError(t, err)
	assert.NotEmpty(t, pair.AccessToken.Token)
	assert.NotEmpty(t, pair.RefreshToken.Token)
	claims, err := h.ValidateToken(pair.AccessToken.Token)
	require.NoError(t, err)
	assert.Equal(t, int64(99), claims.UserID)
}

func TestNew_DefaultLifetimes(t *testing.T) {
	privPEM, pubPEM := rsaTestKeys(t)
	h, err := New(privPEM, pubPEM, 0, 0, "")
	require.NoError(t, err)
	assert.Positive(t, h.atLifetime)
	assert.Positive(t, h.rtLifetime)
}

func TestNew_PublicKeyOnly_ValidatesTokens(t *testing.T) {
	privPEM, pubPEM := rsaTestKeys(t)
	sign, err := New(privPEM, pubPEM, time.Minute, time.Hour, "")
	require.NoError(t, err)
	pair, err := sign.NewTokenPair(42)
	require.NoError(t, err)
	validate, err := New(nil, pubPEM, time.Minute, time.Hour, "")
	require.NoError(t, err)
	claims, err := validate.ValidateToken(pair.AccessToken.Token)
	require.NoError(t, err)
	assert.Equal(t, int64(42), claims.UserID)
}

func TestNewTokenPair_RequiresPrivateKey(t *testing.T) {
	_, pubPEM := rsaTestKeys(t)
	h, err := New(nil, pubPEM, time.Minute, time.Hour, "")
	require.NoError(t, err)
	_, err = h.NewTokenPair(1)
	require.Error(t, err, "expected error when signing key is missing")
}

func TestNewFromConfig(t *testing.T) {
	privPEM, pubPEM := rsaTestKeys(t)
	dir := t.TempDir()
	privPath := filepath.Join(dir, "priv.pem")
	pubPath := filepath.Join(dir, "pub.pem")
	require.NoError(t, os.WriteFile(privPath, privPEM, 0o600))
	require.NoError(t, os.WriteFile(pubPath, pubPEM, 0o600))
	h, err := NewFromConfig(JWTConfig{
		PrivateKeyFile:       privPath,
		PublicKeyFile:        pubPath,
		AccessTokenLifetime:  time.Minute,
		RefreshTokenLifetime: time.Hour,
	})
	require.NoError(t, err)
	require.NotNil(t, h)
}

func TestNewFromConfig_PublicKeyOnly(t *testing.T) {
	_, pubPEM := rsaTestKeys(t)
	dir := t.TempDir()
	pubPath := filepath.Join(dir, "pub.pem")
	require.NoError(t, os.WriteFile(pubPath, pubPEM, 0o600))
	h, err := NewFromConfig(JWTConfig{
		PublicKeyFile:        pubPath,
		AccessTokenLifetime:  time.Minute,
		RefreshTokenLifetime: time.Hour,
	})
	require.NoError(t, err)
	_, err = h.NewTokenPair(1)
	require.Error(t, err, "expected signing to fail without private key file")
}

func TestValidateToken_Invalid(t *testing.T) {
	privPEM, pubPEM := rsaTestKeys(t)
	h, err := New(privPEM, pubPEM, time.Minute, time.Hour, "")
	require.NoError(t, err)
	_, err = h.ValidateToken("not-a-jwt")
	require.Error(t, err, "expected error")
}

func TestValidateToken_RejectsWrongIssuer(t *testing.T) {
	privPEM, pubPEM := rsaTestKeys(t)
	h, err := New(privPEM, pubPEM, time.Minute, time.Hour, "")
	require.NoError(t, err)
	priv, err := jwt.ParseRSAPrivateKeyFromPEM(privPEM)
	require.NoError(t, err)
	claims := TokenClaims{
		UserID: 1,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "other",
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		},
	}
	tok, err := jwt.NewWithClaims(jwt.SigningMethodRS256, claims).SignedString(priv)
	require.NoError(t, err)
	_, err = h.ValidateToken(tok)
	require.Error(t, err, "expected wrong issuer to be rejected")
}

func TestValidateToken_RejectsNonRS256(t *testing.T) {
	privPEM, pubPEM := rsaTestKeys(t)
	h, err := New(privPEM, pubPEM, time.Minute, time.Hour, "")
	require.NoError(t, err)
	priv, err := jwt.ParseRSAPrivateKeyFromPEM(privPEM)
	require.NoError(t, err)
	claims := TokenClaims{
		UserID: 7,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    DefaultIssuer,
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		},
	}
	rs512Token, err := jwt.NewWithClaims(jwt.SigningMethodRS512, claims).SignedString(priv)
	require.NoError(t, err)
	_, err = h.ValidateToken(rs512Token)
	require.Error(t, err, "expected RS512 token to be rejected")
}

func TestNew_WithAudience_ValidateRequiresMatch(t *testing.T) {
	privPEM, pubPEM := rsaTestKeys(t)
	sign, err := New(privPEM, pubPEM, time.Minute, time.Hour, "skeeper-api")
	require.NoError(t, err)
	pair, err := sign.NewTokenPair(1)
	require.NoError(t, err)
	wrongAud, err := New(nil, pubPEM, time.Minute, time.Hour, "other")
	require.NoError(t, err)
	_, err = wrongAud.ValidateToken(pair.AccessToken.Token)
	require.Error(t, err, "expected audience mismatch")
	rightAud, err := New(nil, pubPEM, time.Minute, time.Hour, "skeeper-api")
	require.NoError(t, err)
	_, err = rightAud.ValidateToken(pair.AccessToken.Token)
	require.NoError(t, err)
}
