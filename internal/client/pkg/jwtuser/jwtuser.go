// Package jwtuser pulls user_id out of the access JWT without verifying the signature—we only
// need it for local row scoping; the server still enforces auth on every RPC.
package jwtuser

import (
	"fmt"

	"github.com/golang-jwt/jwt/v4"
)

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
