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
)

func rsaTestKeys(t *testing.T) (privPEM, pubPEM []byte) {
	t.Helper()
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	privDER := x509.MarshalPKCS1PrivateKey(priv)
	privPEM = pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: privDER})
	pubDER, err := x509.MarshalPKIXPublicKey(&priv.PublicKey)
	if err != nil {
		t.Fatal(err)
	}
	pubPEM = pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pubDER})
	return privPEM, pubPEM
}

func TestNew_NewTokenPair_ValidateToken(t *testing.T) {
	privPEM, pubPEM := rsaTestKeys(t)
	h, err := New(privPEM, pubPEM, time.Minute, time.Hour, "")
	if err != nil {
		t.Fatal(err)
	}
	pair, err := h.NewTokenPair(99)
	if err != nil {
		t.Fatal(err)
	}
	if pair.AccessToken.Token == "" || pair.RefreshToken.Token == "" {
		t.Fatal("empty tokens")
	}
	claims, err := h.ValidateToken(pair.AccessToken.Token)
	if err != nil {
		t.Fatal(err)
	}
	if claims.UserID != 99 {
		t.Fatalf("user id %d", claims.UserID)
	}
}

func TestNew_DefaultLifetimes(t *testing.T) {
	privPEM, pubPEM := rsaTestKeys(t)
	h, err := New(privPEM, pubPEM, 0, 0, "")
	if err != nil {
		t.Fatal(err)
	}
	if h.atLifetime <= 0 || h.rtLifetime <= 0 {
		t.Fatal("expected positive defaults")
	}
}

func TestNew_PublicKeyOnly_ValidatesTokens(t *testing.T) {
	privPEM, pubPEM := rsaTestKeys(t)
	sign, err := New(privPEM, pubPEM, time.Minute, time.Hour, "")
	if err != nil {
		t.Fatal(err)
	}
	pair, err := sign.NewTokenPair(42)
	if err != nil {
		t.Fatal(err)
	}
	validate, err := New(nil, pubPEM, time.Minute, time.Hour, "")
	if err != nil {
		t.Fatal(err)
	}
	claims, err := validate.ValidateToken(pair.AccessToken.Token)
	if err != nil || claims.UserID != 42 {
		t.Fatalf("claims %+v err %v", claims, err)
	}
}

func TestNewTokenPair_RequiresPrivateKey(t *testing.T) {
	_, pubPEM := rsaTestKeys(t)
	h, err := New(nil, pubPEM, time.Minute, time.Hour, "")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := h.NewTokenPair(1); err == nil {
		t.Fatal("expected error when signing key is missing")
	}
}

func TestNewFromConfig(t *testing.T) {
	privPEM, pubPEM := rsaTestKeys(t)
	dir := t.TempDir()
	privPath := filepath.Join(dir, "priv.pem")
	pubPath := filepath.Join(dir, "pub.pem")
	if err := os.WriteFile(privPath, privPEM, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(pubPath, pubPEM, 0o600); err != nil {
		t.Fatal(err)
	}
	h, err := NewFromConfig(JWTConfig{
		PrivateKeyFile:       privPath,
		PublicKeyFile:        pubPath,
		AccessTokenLifetime:  time.Minute,
		RefreshTokenLifetime: time.Hour,
	})
	if err != nil {
		t.Fatal(err)
	}
	if h == nil {
		t.Fatal("nil helper")
	}
}

func TestNewFromConfig_PublicKeyOnly(t *testing.T) {
	_, pubPEM := rsaTestKeys(t)
	dir := t.TempDir()
	pubPath := filepath.Join(dir, "pub.pem")
	if err := os.WriteFile(pubPath, pubPEM, 0o600); err != nil {
		t.Fatal(err)
	}
	h, err := NewFromConfig(JWTConfig{
		PublicKeyFile:        pubPath,
		AccessTokenLifetime:  time.Minute,
		RefreshTokenLifetime: time.Hour,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := h.NewTokenPair(1); err == nil {
		t.Fatal("expected signing to fail without private key file")
	}
}

func TestValidateToken_Invalid(t *testing.T) {
	privPEM, pubPEM := rsaTestKeys(t)
	h, err := New(privPEM, pubPEM, time.Minute, time.Hour, "")
	if err != nil {
		t.Fatal(err)
	}
	_, err = h.ValidateToken("not-a-jwt")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestValidateToken_RejectsWrongIssuer(t *testing.T) {
	privPEM, pubPEM := rsaTestKeys(t)
	h, err := New(privPEM, pubPEM, time.Minute, time.Hour, "")
	if err != nil {
		t.Fatal(err)
	}
	priv, err := jwt.ParseRSAPrivateKeyFromPEM(privPEM)
	if err != nil {
		t.Fatal(err)
	}
	claims := TokenClaims{
		UserID: 1,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "other",
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		},
	}
	tok, err := jwt.NewWithClaims(jwt.SigningMethodRS256, claims).SignedString(priv)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := h.ValidateToken(tok); err == nil {
		t.Fatal("expected wrong issuer to be rejected")
	}
}

func TestValidateToken_RejectsNonRS256(t *testing.T) {
	privPEM, pubPEM := rsaTestKeys(t)
	h, err := New(privPEM, pubPEM, time.Minute, time.Hour, "")
	if err != nil {
		t.Fatal(err)
	}
	priv, err := jwt.ParseRSAPrivateKeyFromPEM(privPEM)
	if err != nil {
		t.Fatal(err)
	}
	claims := TokenClaims{
		UserID: 7,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    DefaultIssuer,
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		},
	}
	rs512Token, err := jwt.NewWithClaims(jwt.SigningMethodRS512, claims).SignedString(priv)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := h.ValidateToken(rs512Token); err == nil {
		t.Fatal("expected RS512 token to be rejected")
	}
}

func TestNew_WithAudience_ValidateRequiresMatch(t *testing.T) {
	privPEM, pubPEM := rsaTestKeys(t)
	sign, err := New(privPEM, pubPEM, time.Minute, time.Hour, "skeeper-api")
	if err != nil {
		t.Fatal(err)
	}
	pair, err := sign.NewTokenPair(1)
	if err != nil {
		t.Fatal(err)
	}
	wrongAud, err := New(nil, pubPEM, time.Minute, time.Hour, "other")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := wrongAud.ValidateToken(pair.AccessToken.Token); err == nil {
		t.Fatal("expected audience mismatch")
	}
	rightAud, err := New(nil, pubPEM, time.Minute, time.Hour, "skeeper-api")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := rightAud.ValidateToken(pair.AccessToken.Token); err != nil {
		t.Fatal(err)
	}
}
