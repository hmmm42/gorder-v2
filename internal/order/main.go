package main

import (
	"context"
	"flag"
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/hmmm42/gorder-v2/common/broker"
	_ "github.com/hmmm42/gorder-v2/common/config"
	"github.com/hmmm42/gorder-v2/common/discovery"
	"github.com/hmmm42/gorder-v2/common/genproto/orderpb"
	"github.com/hmmm42/gorder-v2/common/logging"
	"github.com/hmmm42/gorder-v2/common/server"
	"github.com/hmmm42/gorder-v2/common/tracing"
	"github.com/hmmm42/gorder-v2/order/infrastructure/consumer"
	"github.com/hmmm42/gorder-v2/order/ports"
	"github.com/hmmm42/gorder-v2/order/service"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
)

func init() {
	httpPort := flag.Int("http", 0, "HTTP server port, if not set, will use config value")
	grpcPort := flag.Int("grpc", 0, "gRPC server port, if not set, will use config value")
	flag.Parse()
	if *httpPort != 0 {
		newAddr := fmt.Sprintf("127.0.0.1:%d", *httpPort)
		viper.Set("order.http-addr", newAddr)
	}
	if *grpcPort != 0 {
		newAddr := fmt.Sprintf("127.0.0.1:%d", *grpcPort)
		viper.Set("order.grpc-addr", newAddr)
		metricsPort := *grpcPort + 10000
		newMetricsAddr := fmt.Sprintf("127.0.0.1:%d", metricsPort)
		viper.Set("order.metrics_export_addr", newMetricsAddr)
	}

	logging.Init()
}

func main() {
	serviceName := viper.GetString("order.service-name")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	shutdown, err := tracing.InitJaegerProvider(viper.GetString("jaeger.url"), serviceName)
	if err != nil {
		logrus.Fatal(err)
	}
	defer func() {
		_ = shutdown(ctx)
	}()

	application, cleanup := service.NewApplication(ctx)
	defer cleanup()

	deregisterFunc, err := discovery.RegisterToConsul(ctx, serviceName)
	if err != nil {
		logrus.Fatal(err)
	}
	defer func() {
		_ = deregisterFunc()
	}()

	ch, closeCh := broker.Connect(
		viper.GetString("rabbitmq.user"),
		viper.GetString("rabbitmq.password"),
		viper.GetString("rabbitmq.host"),
		viper.GetString("rabbitmq.port"),
	)
	defer func() {
		_ = ch.Close()
		_ = closeCh()
	}()
	go consumer.NewConsumer(application).Listen(ch)

	go server.RunGRPCServer(serviceName, func(server *grpc.Server) {
		svc := ports.NewGRPCServer(application)
		orderpb.RegisterOrderServiceServer(server, svc)
	})

	server.RunHTTPServer(serviceName, func(router *gin.Engine) {
		router.StaticFile("/success", "../../public/success.html")
		ports.RegisterHandlersWithOptions(router, HTTPServer{
			app: application,
		}, ports.GinServerOptions{
			BaseURL: "/api",
			// TODO: 增加鉴权中间件
			Middlewares:  nil,
			ErrorHandler: nil,
		})
	})
}
