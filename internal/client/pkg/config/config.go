// Package config loads the CLI client YAML via Viper (env and defaults apply without a file).
package config

import (
	"os"
	"strings"

	"github.com/georgg2003/skeeper/pkg/errors"
	"github.com/spf13/viper"
)

// ClientConfig holds remote service endpoints and local vault location.
type ClientConfig struct {
	AutherAddr  string `mapstructure:"auther_addr"`
	SkeeperAddr string `mapstructure:"skeeper_addr"`
	DataDir     string `mapstructure:"data_dir"`
}

// Load reads configPath with Viper when appropriate, then unmarshals.
//
// Env prefix is SKEEPERCLI (e.g. SKEEPERCLI_AUTHER_ADDR). Legacy vars
// SKEEPERCLI_AUTHER, SKEEPERCLI_SKEEPER, and SKEEPERCLI_DATA are also accepted.
//
// If configPathExplicit is false and the file is missing, loading continues
// using defaults and environment only. If configPathExplicit is true, the
// file must exist and be readable.
func Load(configPath string, configPathExplicit bool) (*ClientConfig, error) {
	v := viper.New()
	v.SetEnvPrefix("SKEEPERCLI")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	_ = v.BindEnv("auther_addr", "SKEEPERCLI_AUTHER")
	_ = v.BindEnv("skeeper_addr", "SKEEPERCLI_SKEEPER")
	_ = v.BindEnv("data_dir", "SKEEPERCLI_DATA")

	v.SetDefault("auther_addr", "127.0.0.1:50051")
	v.SetDefault("skeeper_addr", "127.0.0.1:50052")
	v.SetDefault("data_dir", "~/.skeepercli")

	if configPath != "" {
		if configPathExplicit {
			v.SetConfigFile(configPath)
			if err := v.ReadInConfig(); err != nil {
				return nil, errors.Wrap(err, "read config file")
			}
		} else {
			st, err := os.Stat(configPath)
			if err == nil && !st.IsDir() {
				v.SetConfigFile(configPath)
				if err := v.ReadInConfig(); err != nil {
					return nil, errors.Wrap(err, "read config file")
				}
			}
		}
	}

	var cfg ClientConfig
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, errors.Wrap(err, "unmarshal config")
	}
	return &cfg, nil
}
