package config

import (
	"fmt"
	"strings"

	"github.com/georgg2003/skeeper/pkg/errors"
	"github.com/spf13/viper"
)

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
	err := v.Unmarshal(&cfg)
	if err != nil {
		return nil, errors.Wrap(err, "failed to unmarshall config")
	}
	return &cfg, nil
}
