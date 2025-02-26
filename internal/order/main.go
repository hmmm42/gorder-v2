package main

import (
	"github.com/gin-gonic/gin"
	"github.com/hmmm42/gorder-v2/common/config"
	"github.com/hmmm42/gorder-v2/common/genproto/orderpb"
	"github.com/hmmm42/gorder-v2/common/server"
	"github.com/hmmm42/gorder-v2/order/ports"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
	"log"
)

func init() {
	if err := config.NewViperConfig(); err != nil {
		log.Fatal()
	}
}

func main() {
	serviceName := viper.GetString("order.service-name")
	go server.RunGRPCServer(serviceName, func(server *grpc.Server) {
		svc := ports.NewGRPCServer()
		orderpb.RegisterOrderServiceServer(server, svc)
	})

	server.RunHTTPServer(serviceName, func(router *gin.Engine) {
		ports.RegisterHandlersWithOptions(router, HTTPServer{}, ports.GinServerOptions{
			BaseURL:      "/api",
			Middlewares:  nil,
			ErrorHandler: nil,
		})
	})
}
