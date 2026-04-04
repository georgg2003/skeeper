package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"

	"github.com/georgg2003/skeeper/pkg/errors"
)

// New loads configPath with Viper, applies envPrefix for AutomaticEnv, unmarshals into T.
func New[T any](configPath string, envPrefix string) (*T, error) {
	if configPath == "" {
		return nil, fmt.Errorf("config path is empty")
	}

	v := viper.New()
	v.SetEnvPrefix(envPrefix)
	v.SetConfigFile(configPath)
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	if err := v.ReadInConfig(); err != nil {
		return nil, errors.Wrap(err, "failed to read config file")
	}

	var cfg T
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshall config")
	}
	return &cfg, nil
}
