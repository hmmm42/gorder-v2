package main

import (
	"context"

	"github.com/hmmm42/gorder-v2/common/broker"
	_ "github.com/hmmm42/gorder-v2/common/config"
	"github.com/hmmm42/gorder-v2/common/logging"
	"github.com/hmmm42/gorder-v2/common/server"
	"github.com/hmmm42/gorder-v2/common/tracing"
	"github.com/hmmm42/gorder-v2/payment/infrastructure/consumer"
	"github.com/hmmm42/gorder-v2/payment/service"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

func init() {
	logging.Init()
}

func main() {
	serviceName := viper.GetString("payment.service-name")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	serverType := viper.GetString("payment.server-to-run")

	shutdown, err := tracing.InitJaegerProvider(viper.GetString("jaeger.url"), serviceName)
	if err != nil {
		logrus.Fatal(err)
	}
	defer shutdown(ctx)

	application, cleanUp := service.NewApplication(ctx)
	defer cleanUp()

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

	paymentHandler := NewPaymentHandler(ch)
	switch serverType {
	case "http":
		server.RunHTTPServer(serviceName, paymentHandler.RegisterRoutes)
	case "grpc":
		logrus.Panic("unsupported server type: grpc")
	default:
		panic("unreachable code")
	}
}
