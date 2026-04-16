// Package grpcclient returns grpc.DialOption slices—TLS (custom CA) or insecure when explicitly allowed.
package grpcclient

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
)

type TLSConfig struct {
	Enabled bool   `mapstructure:"enabled"`
	CAFile  string `mapstructure:"ca_file"`
}

func DialOptions(cfg TLSConfig) ([]grpc.DialOption, error) {
	if cfg.Enabled {
		if cfg.CAFile == "" {
			return nil, fmt.Errorf("grpc tls enabled but ca_file is empty")
		}
		pemData, err := os.ReadFile(cfg.CAFile)
		if err != nil {
			return nil, fmt.Errorf("read grpc ca file: %w", err)
		}
		pool := x509.NewCertPool()
		if !pool.AppendCertsFromPEM(pemData) {
			return nil, fmt.Errorf("grpc ca file: no certificates parsed")
		}
		tlsConf := &tls.Config{
			RootCAs:    pool,
			MinVersion: tls.VersionTLS12,
		}
		return []grpc.DialOption{grpc.WithTransportCredentials(credentials.NewTLS(tlsConf))}, nil
	}
	return []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}, nil
}
