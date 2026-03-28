package auther

import (
	"os"

	"google.golang.org/grpc"

	"github.com/georgg2003/skeeper/api"
	delivery "github.com/georgg2003/skeeper/internal/auther/delivery"
	"github.com/georgg2003/skeeper/internal/auther/pkg/config"
	"github.com/georgg2003/skeeper/internal/auther/usecase"
	"github.com/georgg2003/skeeper/internal/pkg/log"
	"github.com/georgg2003/skeeper/pkg/errors"
	"github.com/georgg2003/skeeper/pkg/jwthelper"
)

func initJWTHelper(cfg config.JWTConfig) (jwthelper.JWTHelper, error) {
	privBytes, err := os.ReadFile(cfg.PrivateKeyFile)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read private key file")
	}
	pubBytes, err := os.ReadFile(cfg.PublicKeyFile)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read public key file")
	}
	return jwthelper.New(privBytes, pubBytes)
}

func main() {
	l := log.New()
	cfg, err := config.New()
	if err != nil {
		l.Error("failed to init config", "err", err)
		os.Exit(1)
	}

	jwtHelper, err := initJWTHelper(cfg.JWT)
	if err != nil {
		l.Error("failed to init jwt helper", "err", err)
		os.Exit(1)
	}

	uc := usecase.New(l, jwtHelper)
	service := delivery.New(l, uc)

	server := grpc.NewServer()
	api.RegisterAutherServer(server, service)
}
