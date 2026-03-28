package config

import (
	"github.com/georgg2003/skeeper/pkg/errors"
	"github.com/spf13/viper"
)

type PostgresConfig struct {
	Host     string `mapstructure:"host"`
	Port     uint16 `mapstructure:"port"`
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
	Database string `mapstructure:"database"`
}

type JWTConfig struct {
	PrivateKeyFile string `mapstructure:"private_key_file"`
	PublicKeyFile  string `mapstructure:"public_key_file"`
}

type AutherConfig struct {
	Postgres PostgresConfig `mapstructure:"postgres"`
	JWT      JWTConfig      `mapstructure:"jwt"`
}

func New() (*AutherConfig, error) {
	var cfg AutherConfig
	viper := viper.New()
	err := viper.Unmarshal(&cfg)
	if err != nil {
		return nil, errors.Wrap(err, "failed to unmarshall config")
	}
	return &cfg, nil
}
