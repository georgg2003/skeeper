// Package jwtuser reads Auther access-token claims without signature verification.
// The token is only used to record user_id in the local session; API calls are still authorized by the server.
package jwtuser

import (
	"fmt"

	"github.com/golang-jwt/jwt/v4"
)

// UserIDFromAccessTokenUnverified returns the numeric user id embedded in the JWT payload.
func UserIDFromAccessTokenUnverified(token string) (int64, error) {
	p := jwt.NewParser()
	t, _, err := p.ParseUnverified(token, jwt.MapClaims{})
	if err != nil {
		return 0, fmt.Errorf("parse jwt: %w", err)
	}
	m, ok := t.Claims.(jwt.MapClaims)
	if !ok {
		return 0, fmt.Errorf("invalid jwt claims type")
	}
	if v, ok := m["UserID"].(float64); ok {
		return int64(v), nil
	}
	if v, ok := m["user_id"].(float64); ok {
		return int64(v), nil
	}
	return 0, fmt.Errorf("jwt missing user id claim")
}
