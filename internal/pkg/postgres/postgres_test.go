package postgres

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfig_PoolConfig_URLAndSSLMode(t *testing.T) {
	cfg := Config{
		Host:     "db.example.com",
		Port:     5432,
		User:     "app",
		Password: "p@ss:word",
		Database: "my db",
	}
	pc, err := cfg.PoolConfig()
	require.NoError(t, err)
	conn := pc.ConnString()
	assert.Contains(t, conn, "sslmode=disable", "missing sslmode")
	assert.True(t, strings.Contains(conn, "p%40ss%3Aword") || strings.Contains(conn, "p@ss"),
		"password not encoded in conn string: %q", conn)
}

func TestConfig_PoolConfig_ExplicitSSLMode(t *testing.T) {
	cfg := Config{
		Host: "h", Port: 1, User: "u", Password: "p", Database: "d",
		SSLMode: "require",
	}
	pc, err := cfg.PoolConfig()
	require.NoError(t, err)
	assert.Contains(t, pc.ConnString(), "sslmode=require")
}

func TestNewPoolFromConnString_Invalid(t *testing.T) {
	_, err := NewPoolFromConnString(t.Context(), "not a postgres url \x00")
	require.Error(t, err, "expected error")
}

