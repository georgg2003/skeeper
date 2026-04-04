package postgres

import (
	"strings"
	"testing"
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
	if err != nil {
		t.Fatal(err)
	}
	conn := pc.ConnString()
	if !strings.Contains(conn, "sslmode=disable") {
		t.Fatalf("missing sslmode: %q", conn)
	}
	if !strings.Contains(conn, "p%40ss%3Aword") && !strings.Contains(conn, "p@ss") {
		// url.UserPassword percent-encodes @ and :
		t.Fatalf("password not encoded in conn string: %q", conn)
	}
}

func TestConfig_PoolConfig_ExplicitSSLMode(t *testing.T) {
	cfg := Config{
		Host: "h", Port: 1, User: "u", Password: "p", Database: "d",
		SSLMode: "require",
	}
	pc, err := cfg.PoolConfig()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(pc.ConnString(), "sslmode=require") {
		t.Fatalf("got %q", pc.ConnString())
	}
}

func TestNewPoolFromConnString_Invalid(t *testing.T) {
	_, err := NewPoolFromConnString(t.Context(), "not a postgres url \x00")
	if err == nil {
		t.Fatal("expected error")
	}
}

