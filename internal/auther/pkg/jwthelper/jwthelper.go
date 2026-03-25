package jwthelper

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/georgg2003/skeeper/pkg/errors"

	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
)

type TokenClaims struct {
	jwt.RegisteredClaims
	UserID int64
}

//go:generate TODO add mockgen generation
type JWTHelper interface {
	NewTokenPair(userID int64) (string, string, time.Time, error)
	ValidateToken(encodedToken string) (int64, error)
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

func (h *helper) NewTokenPair(userID int64) (string, string, time.Time, error) {
	accessExpiry := time.Now().Add(time.Minute * 15)
	aClaims := newClaims(userID, accessExpiry)
	accessToken, err := jwt.NewWithClaims(jwt.SigningMethodRS256, aClaims).SignedString(h.privateKey)
	if err != nil {
		return "", "", time.Time{}, err
	}

	refreshExpiry := time.Now().Add(time.Hour * 24 * 30)
	refreshToken, err := generateRandomString()
	if err != nil {
		return "", "", time.Time{}, errors.Wrap(err, "failed to generate refresh token")
	}

	return accessToken, refreshToken, refreshExpiry, err
}

func (h *helper) ValidateToken(encodedToken string) (int64, error) {
	claims := TokenClaims{}

	token, err := jwt.ParseWithClaims(encodedToken, &claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return h.publicKey, nil
	})

	if err != nil || !token.Valid {
		return 0, errors.New("invalid token")
	}

	return claims.UserID, nil
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
