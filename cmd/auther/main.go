package auther

import (
	"os"

	"google.golang.org/grpc"

	"github.com/georgg2003/skeeper/api"
	delivery "github.com/georgg2003/skeeper/internal/auther/delivery"
	"github.com/georgg2003/skeeper/internal/auther/pkg/jwthelper"
	"github.com/georgg2003/skeeper/internal/auther/usecase"
	"github.com/georgg2003/skeeper/internal/pkg/config"
	"github.com/georgg2003/skeeper/pkg/errors"
)

func initJWTHelper(cfg config.JWTConfig) (*jwthelper.JWTHelper, error) {
	privBytes, err := os.ReadFile(cfg.PrivateKeyFile)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read private key file")
	}
	pubBytes, err := os.ReadFile(cfg.PublicKeyFile)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read public key file")
	}
	jwthelper.New(privBytes, pubBytes)
}

func main() {
	cfg, err := config.New()
	if err != nil {
		// add log
		os.Exit(1)
	}

	jwtHelper, err := initJWTHelper(cfg.JWT)
	if err != nil {
		// add log
		os.Exit(1)
	}

	uc := usecase.New(jwtHelper)
	service := delivery.New(uc)

	server := grpc.NewServer()
	api.RegisterAutherServer(server, service)
}
