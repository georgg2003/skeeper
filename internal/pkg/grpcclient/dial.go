// Package grpcclient builds standard gRPC client dial options (TLS or insecure).
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

// TLSConfig configures TLS for outbound gRPC. When Enabled is false, insecure credentials are used.
type TLSConfig struct {
	Enabled bool   `mapstructure:"enabled"`
	CAFile  string `mapstructure:"ca_file"`
}

// DialOptions returns transport credentials for grpc.NewClient.
func DialOptions(cfg TLSConfig) ([]grpc.DialOption, error) {
	if !cfg.Enabled {
		return []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}, nil
	}
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
