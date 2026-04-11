package jwtuser

import (
	"testing"

	"github.com/golang-jwt/jwt/v4"
	"github.com/stretchr/testify/require"
)

func TestUserIDFromAccessTokenUnverified_UserIDClaim(t *testing.T) {
	tok := jwt.NewWithClaims(jwt.SigningMethodNone, jwt.MapClaims{
		"UserID": float64(7),
	})
	s, err := tok.SignedString(jwt.UnsafeAllowNoneSignatureType)
	require.NoError(t, err)
	id, err := UserIDFromAccessTokenUnverified(s)
	require.NoError(t, err)
	require.Equal(t, int64(7), id)
}

func TestUserIDFromAccessTokenUnverified_snakeClaim(t *testing.T) {
	tok := jwt.NewWithClaims(jwt.SigningMethodNone, jwt.MapClaims{
		"user_id": float64(42),
	})
	s, err := tok.SignedString(jwt.UnsafeAllowNoneSignatureType)
	require.NoError(t, err)
	id, err := UserIDFromAccessTokenUnverified(s)
	require.NoError(t, err)
	require.Equal(t, int64(42), id)
}
