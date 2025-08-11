package main

import (
	"context"
	"flag"
	"fmt"

	_ "github.com/hmmm42/gorder-v2/common/config"
	"github.com/hmmm42/gorder-v2/common/discovery"
	"github.com/hmmm42/gorder-v2/common/genproto/stockpb"
	"github.com/hmmm42/gorder-v2/common/logging"
	"github.com/hmmm42/gorder-v2/common/server"
	"github.com/hmmm42/gorder-v2/common/tracing"
	"github.com/hmmm42/gorder-v2/stock/ports"
	"github.com/hmmm42/gorder-v2/stock/service"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
)

func init() {
	var port = flag.Int("port", 0, "gRPC server port, if not set, will use config value")
	flag.Parse()
	if *port != 0 {
		newAddr := fmt.Sprintf("127.0.0.1:%d", *port)
		metricsPort := *port + 10000
		newMetricsAddr := fmt.Sprintf("127.0.0.1:%d", metricsPort)
		viper.Set("stock.grpc-addr", newAddr)
		viper.Set("stock.metrics_export_addr", newMetricsAddr)
	}

	logging.Init()
}

func main() {
	serviceName := viper.GetString("stock.service-name")
	serverType := viper.GetString("stock.server-to-run")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	shutdown, err := tracing.InitJaegerProvider(viper.GetString("jaeger.url"), serviceName)
	if err != nil {
		logrus.Fatal(err)
	}
	defer func() {
		_ = shutdown(ctx)
	}()

	application := service.NewApplication(ctx)

	deregisterFunc, err := discovery.RegisterToConsul(ctx, serviceName)
	if err != nil {
		logrus.Fatal(err)
	}
	defer func() {
		_ = deregisterFunc()
	}()

	switch serverType {
	case "grpc":
		server.RunGRPCServer(serviceName, func(server *grpc.Server) {
			svc := ports.NewGRPCServer(application)
			stockpb.RegisterStockServiceServer(server, svc)
		})
	case "http":
	default:
		panic("unexpected server type")
	}

}
