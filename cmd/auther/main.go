package auther

import (
	"google.golang.org/grpc"

	"github.com/georgg2003/skeeper/api"
	delivery "github.com/georgg2003/skeeper/internal/delivery/auther"
)

func main() {
	service := delivery.New()

	server := grpc.NewServer()
	api.RegisterAutherServer(server, service)
}
