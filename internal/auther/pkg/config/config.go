package config

import (
	"flag"

	"github.com/georgg2003/skeeper/internal/auther/repository/db/postgres"
	"github.com/georgg2003/skeeper/internal/pkg/config"
	"github.com/georgg2003/skeeper/internal/pkg/server"
	"github.com/georgg2003/skeeper/pkg/jwthelper"
)

type AutherConfig struct {
	Postgres postgres.PostgresConfig `mapstructure:"postgres"`
	JWT      jwthelper.JWTConfig     `mapstructure:"jwt"`
	Service  server.ServerConfig     `mapstructure:"service"`
}

func New() (*AutherConfig, error) {
	configPath := flag.String("config", "config/auther.yaml", "config file in yaml/json format")
	flag.Parse()

	return config.New[AutherConfig](*configPath, "AUTHER")
}
