// Package config reads the CLI's YAML (or env only if the default file isn't there).
package config

import (
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/georgg2003/skeeper/pkg/errors"
)

type GRPCClientTLS struct {
	Enabled bool   `mapstructure:"enabled"`
	CAFile  string `mapstructure:"ca_file"`
}

// ClientLogging configures slog for the CLI process (use cases and repositories).
// If File is empty, logs go to the process stdout (wired in cmd/client).
type ClientLogging struct {
	// File is a path to append logs to; "~/" is expanded. Empty means stdout only.
	File string `mapstructure:"file"`
	// Level is one of: debug, info, warn, error (case-insensitive). Default info.
	Level string `mapstructure:"level"`
	// Format is json or text (case-insensitive). Default json.
	Format string `mapstructure:"format"`
}

type ClientConfig struct {
	AutherAddr   string        `mapstructure:"auther_addr"`
	SkeeperAddr  string        `mapstructure:"skeeper_addr"`
	DataDir      string        `mapstructure:"data_dir"`
	MaxFileBytes int64         `mapstructure:"max_file_bytes"`
	GRPCTLS      GRPCClientTLS `mapstructure:"grpc_tls"`
	Logging      ClientLogging `mapstructure:"logging"`
}

// Load builds a single Viper instance: defaults, env (SKEEPERCLI_*), optional YAML file, and optional Cobra --config.
// All runtime settings (addresses, data dir, TLS, limits) come only from that merge — no duplicate CLI flags.
func Load(cmd *cobra.Command) (*ClientConfig, error) {
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
	_ = v.BindEnv("client_config_file", "SKEEPERCLI_CONFIG")
	_ = v.BindEnv("logging.file", "SKEEPERCLI_LOG_FILE")
	_ = v.BindEnv("logging.level", "SKEEPERCLI_LOG_LEVEL")
	_ = v.BindEnv("logging.format", "SKEEPERCLI_LOG_FORMAT")

	v.SetDefault("auther_addr", "127.0.0.1:50051")
	v.SetDefault("skeeper_addr", "127.0.0.1:50052")
	v.SetDefault("data_dir", "~/.skeepercli")
	v.SetDefault("max_file_bytes", int64(10<<20)) // 10 MiB cap for FILE entry payload
	v.SetDefault("grpc_tls.enabled", false)
	v.SetDefault("grpc_tls.ca_file", "config/keys/grpc_server.crt")
	v.SetDefault("client_config_file", "config/client.yaml")
	v.SetDefault("logging.level", "info")
	v.SetDefault("logging.format", "json")

	if cmd != nil {
		if f := cmd.Root().PersistentFlags().Lookup("config"); f != nil {
			if err := v.BindPFlag("client_config_file", f); err != nil {
				return nil, errors.Wrap(err, "bind config flag to viper")
			}
		}
	}

	cfgPath := v.GetString("client_config_file")
	cfgExplicit := cmd != nil && cmd.Root().PersistentFlags().Changed("config")

	if cfgPath != "" {
		if cfgExplicit {
			v.SetConfigFile(cfgPath)
			if err := v.ReadInConfig(); err != nil {
				return nil, errors.Wrap(err, "read config file")
			}
		} else {
			st, err := os.Stat(cfgPath)
			if err == nil && !st.IsDir() {
				v.SetConfigFile(cfgPath)
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
