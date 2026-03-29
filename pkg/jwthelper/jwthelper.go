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

type Token struct {
	Token     string
	ExpiresAt time.Time
}

type TokenPair struct {
	AccessToken  Token
	RefreshToken Token
}

type TokenClaims struct {
	jwt.RegisteredClaims
	UserID int64
}

type JWTHelper interface {
	NewTokenPair(userID int64) (TokenPair, error)
	ValidateToken(encodedToken string) (TokenClaims, error)
}

type helper struct {
	privateKey *rsa.PrivateKey
	publicKey  *rsa.PublicKey
}

func newClaims(userID int64, expiresAt time.Time) TokenClaims {
	return TokenClaims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ID:        uuid.NewString(),
		},
	}
}

func generateRandomString() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func (h *helper) NewTokenPair(userID int64) (TokenPair, error) {
	accessExpiresAt := time.Now().Add(time.Minute * 15)
	aClaims := newClaims(userID, accessExpiresAt)
	accessToken, err := jwt.NewWithClaims(jwt.SigningMethodRS256, aClaims).SignedString(h.privateKey)
	if err != nil {
		return TokenPair{}, err
	}

	refreshExpiresAt := time.Now().Add(time.Hour * 24 * 30)
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
	}, err
}

func (h *helper) ValidateToken(encodedToken string) (TokenClaims, error) {
	claims := TokenClaims{}

	token, err := jwt.ParseWithClaims(encodedToken, &claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return h.publicKey, nil
	})

	if err != nil || !token.Valid {
		return claims, errors.New("invalid token")
	}

	return claims, nil
}

func New(privByte, pubByte []byte) (JWTHelper, error) {
	privKey, err := jwt.ParseRSAPrivateKeyFromPEM(privByte)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse rsa private key")
	}
	pubKey, err := jwt.ParseRSAPublicKeyFromPEM(pubByte)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse rsa public key")
	}
	return &helper{
		privateKey: privKey,
		publicKey:  pubKey,
	}, nil
}

type JWTConfig struct {
	PrivateKeyFile string `mapstructure:"private_key_file"`
	PublicKeyFile  string `mapstructure:"public_key_file"`
}

func NewFromFiles(cfg JWTConfig) (JWTHelper, error) {
	privBytes, err := os.ReadFile(cfg.PrivateKeyFile)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read private key file")
	}
	pubBytes, err := os.ReadFile(cfg.PublicKeyFile)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read public key file")
	}
	return New(privBytes, pubBytes)
}
