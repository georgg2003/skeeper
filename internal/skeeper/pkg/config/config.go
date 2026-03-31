package config

import (
	"flag"

	"github.com/georgg2003/skeeper/internal/pkg/config"
	"github.com/georgg2003/skeeper/internal/pkg/server"
	"github.com/georgg2003/skeeper/internal/skeeper/repository/postgres"
	"github.com/georgg2003/skeeper/pkg/jwthelper"
)

type SkeeperConfig struct {
	Postgres postgres.PostgresConfig `mapstructure:"postgres"`
	JWT      jwthelper.JWTConfig     `mapstructure:"jwt"`
	Service  server.ServerConfig     `mapstructure:"service"`
}

func New() (*SkeeperConfig, error) {
	configPath := flag.String("config", "config/skeeper.yaml", "config file in yaml/json format")
	flag.Parse()

	return config.New[SkeeperConfig](*configPath, "SKEEPER")
}
