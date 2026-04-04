// Package config loads the CLI client YAML via Viper.
package config

import (
	pkgconfig "github.com/georgg2003/skeeper/internal/pkg/config"
)

// ClientConfig holds remote service endpoints and local vault location.
type ClientConfig struct {
	AutherAddr  string `mapstructure:"auther_addr"`
	SkeeperAddr string `mapstructure:"skeeper_addr"`
	DataDir     string `mapstructure:"data_dir"`
}

// New reads and unmarshals the client config file. Viper uses env prefix SKEEPERCLI
// (e.g. SKEEPERCLI_AUTHER_ADDR) and also binds legacy SKEEPERCLI_AUTHER,
// SKEEPERCLI_SKEEPER, and SKEEPERCLI_DATA.
func New(configPath string) (*ClientConfig, error) {
	return pkgconfig.New[ClientConfig](configPath, "SKEEPERCLI")
}
