// Package jwthelper signs access JWTs (RSA) and mints random refresh tokens. Auther issues them;
// Skeeper validates the access token on each call.
package jwthelper

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/hex"
	"fmt"
	"os"
	"time"

	"github.com/georgg2003/skeeper/pkg/errors"

	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
)

// DefaultIssuer is set on minted access tokens and required when validating.
const DefaultIssuer = "skeeper"

// Token is a bearer string with wall-clock expiry metadata.
type Token struct {
	Token     string
	ExpiresAt time.Time
}

// TokenPair is issued on login: signed access JWT plus opaque refresh token.
type TokenPair struct {
	AccessToken  Token
	RefreshToken Token
}

// TokenClaims is the signed payload of an access token (RS256).
type TokenClaims struct {
	jwt.RegisteredClaims
	UserID int64
}

// JWTHelper mints and validates access JWTs and generates refresh token strings.
type JWTHelper struct {
	privateKey *rsa.PrivateKey
	publicKey  *rsa.PublicKey

	atLifetime time.Duration
	rtLifetime time.Duration
	// audience is optional; when set it is embedded in access tokens and required on validation.
	audience string
}

func (h *JWTHelper) newClaims(userID int64, expiresAt time.Time) TokenClaims {
	rc := jwt.RegisteredClaims{
		Issuer:    DefaultIssuer,
		ExpiresAt: jwt.NewNumericDate(expiresAt),
		IssuedAt:  jwt.NewNumericDate(time.Now()),
		ID:        uuid.NewString(),
	}
	if h.audience != "" {
		rc.Audience = jwt.ClaimStrings{h.audience}
	}
	return TokenClaims{
		UserID:           userID,
		RegisteredClaims: rc,
	}
}

func generateRandomString() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// NewTokenPair creates a fresh access JWT and refresh secret for the given user id.
func (h *JWTHelper) NewTokenPair(userID int64) (TokenPair, error) {
	if h.privateKey == nil {
		return TokenPair{}, errors.New("jwt: signing key not configured")
	}
	accessExpiresAt := time.Now().Add(h.atLifetime)
	aClaims := h.newClaims(userID, accessExpiresAt)
	accessToken, err := jwt.NewWithClaims(jwt.SigningMethodRS256, aClaims).SignedString(h.privateKey)
	if err != nil {
		return TokenPair{}, err
	}

	refreshExpiresAt := time.Now().Add(h.rtLifetime)
	refreshToken, err := generateRandomString()
	if err != nil {
		return TokenPair{}, errors.Wrap(err, "failed to generate refresh token")
	}

	return TokenPair{
		AccessToken: Token{
			Token:     accessToken,
			ExpiresAt: accessExpiresAt,
		},
		RefreshToken: Token{
			Token:     refreshToken,
			ExpiresAt: refreshExpiresAt,
		},
	}, nil
}

// ValidateToken parses and verifies an access JWT and returns its claims.
func (h *JWTHelper) ValidateToken(encodedToken string) (TokenClaims, error) {
	claims := TokenClaims{}

	token, err := jwt.ParseWithClaims(encodedToken, &claims, func(token *jwt.Token) (interface{}, error) {
		if token.Method != jwt.SigningMethodRS256 {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return h.publicKey, nil
	})

	if err != nil || !token.Valid {
		return claims, errors.New("invalid token")
	}

	if claims.Issuer != DefaultIssuer {
		return claims, errors.New("invalid token")
	}

	if h.audience != "" && !claims.VerifyAudience(h.audience, true) {
		return claims, errors.New("invalid token")
	}

	return claims, nil
}

// New builds a helper from PEM-encoded RSA keys. Empty privByte enables verify-only mode.
func New(
	privByte,
	pubByte []byte,
	atLifetime time.Duration,
	rtLifetime time.Duration,
	audience string,
) (*JWTHelper, error) {
	pubKey, err := jwt.ParseRSAPublicKeyFromPEM(pubByte)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse rsa public key")
	}
	var privKey *rsa.PrivateKey
	if len(privByte) > 0 {
		privKey, err = jwt.ParseRSAPrivateKeyFromPEM(privByte)
		if err != nil {
			return nil, errors.Wrap(err, "failed to parse rsa private key")
		}
	}
	if atLifetime <= 0 {
		atLifetime = time.Minute * 15
	}
	if rtLifetime <= 0 {
		rtLifetime = time.Hour * 24
	}
	return &JWTHelper{
		privateKey: privKey,
		publicKey:  pubKey,
		atLifetime: atLifetime,
		rtLifetime: rtLifetime,
		audience:   audience,
	}, nil
}

// JWTConfig maps YAML/mapstructure fields to key paths and token lifetimes.
type JWTConfig struct {
	// PrivateKeyFile is required for signing (Auther). Leave empty for validate-only consumers (e.g. Skeeper).
	PrivateKeyFile       string        `mapstructure:"private_key_file"`
	PublicKeyFile        string        `mapstructure:"public_key_file"`
	AccessTokenLifetime  time.Duration `mapstructure:"access_token_lifetime"`
	RefreshTokenLifetime time.Duration `mapstructure:"refresh_token_lifetime"`
	// Audience is optional; when non-empty, access tokens include aud and validators require it.
	Audience string `mapstructure:"audience"`
}

// NewFromConfig loads PEM files from disk and calls New with defaults for lifetimes.
func NewFromConfig(cfg JWTConfig) (*JWTHelper, error) {
	pubBytes, err := os.ReadFile(cfg.PublicKeyFile)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read public key file")
	}
	var privBytes []byte
	if cfg.PrivateKeyFile != "" {
		privBytes, err = os.ReadFile(cfg.PrivateKeyFile)
		if err != nil {
			return nil, errors.Wrap(err, "failed to read private key file")
		}
	}
	return New(privBytes, pubBytes, cfg.AccessTokenLifetime, cfg.RefreshTokenLifetime, cfg.Audience)
}
