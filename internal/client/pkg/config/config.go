// Package config reads the CLI's YAML (or env only if the default file isn't there).
package config

import (
	"os"
	"strings"

	"github.com/spf13/viper"

	"github.com/georgg2003/skeeper/pkg/errors"
)

type GRPCClientTLS struct {
	Enabled bool   `mapstructure:"enabled"`
	CAFile  string `mapstructure:"ca_file"`
}

type ClientConfig struct {
	AutherAddr        string        `mapstructure:"auther_addr"`
	SkeeperAddr       string        `mapstructure:"skeeper_addr"`
	DataDir           string        `mapstructure:"data_dir"`
	MaxFileBytes      int64         `mapstructure:"max_file_bytes"`
	AllowInsecureGRPC bool          `mapstructure:"allow_insecure_grpc"`
	GRPCTLS           GRPCClientTLS `mapstructure:"grpc_tls"`
}

// Load merges file (if present), defaults, and env. Prefix is SKEEPERCLI_*; older names
// SKEEPERCLI_AUTHER / SKEEPERCLI_SKEEPER / SKEEPERCLI_DATA still work. If configPathExplicit
// is true, the file must exist; otherwise a missing path just means “env + defaults only”.
func Load(configPath string, configPathExplicit bool) (*ClientConfig, error) {
	v := viper.New()
	v.SetEnvPrefix("SKEEPERCLI")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	_ = v.BindEnv("auther_addr", "SKEEPERCLI_AUTHER")
	_ = v.BindEnv("skeeper_addr", "SKEEPERCLI_SKEEPER")
	_ = v.BindEnv("data_dir", "SKEEPERCLI_DATA")
	_ = v.BindEnv("max_file_bytes", "SKEEPERCLI_MAX_FILE_BYTES")
	_ = v.BindEnv("grpc_tls.enabled", "SKEEPERCLI_GRPC_TLS_ENABLED")
	_ = v.BindEnv("grpc_tls.ca_file", "SKEEPERCLI_GRPC_CA_FILE")
	_ = v.BindEnv("allow_insecure_grpc", "SKEEPERCLI_ALLOW_INSECURE_GRPC")

	v.SetDefault("auther_addr", "127.0.0.1:50051")
	v.SetDefault("skeeper_addr", "127.0.0.1:50052")
	v.SetDefault("data_dir", "~/.skeepercli")
	v.SetDefault("max_file_bytes", int64(10<<20)) // 10 MiB cap for FILE entry payload
	v.SetDefault("allow_insecure_grpc", false)
	v.SetDefault("grpc_tls.enabled", false)
	v.SetDefault("grpc_tls.ca_file", "config/keys/grpc_server.crt")

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
