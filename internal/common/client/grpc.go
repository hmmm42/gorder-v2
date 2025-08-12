package client

import (
	"context"
	"fmt"
	"time"

	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/retry"
	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/timeout"
	"github.com/hmmm42/gorder-v2/common/genproto/orderpb"
	"github.com/hmmm42/gorder-v2/common/genproto/stockpb"
	"github.com/spf13/viper"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"

	_ "github.com/mbobakov/grpc-consul-resolver"
)

func NewStockGRPCClient(ctx context.Context) (client stockpb.StockServiceClient, close func() error, err error) {
	grpcAddr := fmt.Sprintf("consul://%s/%s", viper.GetString("consul.addr"), viper.GetString("stock.service-name"))
	opts := grpcDialOpts(grpcAddr)

	conn, err := grpc.NewClient(grpcAddr, opts...)
	if err != nil {
		return nil, func() error {
			return nil
		}, err
	}
	return stockpb.NewStockServiceClient(conn), conn.Close, nil
}

func NewOrderGRPCClient(ctx context.Context) (client orderpb.OrderServiceClient, close func() error, err error) {
	grpcAddr := fmt.Sprintf("consul://%s/%s", viper.GetString("consul.addr"), viper.GetString("order.service-name"))
	opts := grpcDialOpts(grpcAddr)
	conn, err := grpc.NewClient(grpcAddr, opts...)
	if err != nil {
		return nil, func() error { return nil }, err
	}
	return orderpb.NewOrderServiceClient(conn), conn.Close, nil
}

func grpcDialOpts(_ string) []grpc.DialOption {
	// 默认超时配置
	defaultTimeout := 5 * time.Second

	// 默认重试配置
	retryOpts := []retry.CallOption{
		retry.WithBackoff(retry.BackoffExponential(100 * time.Millisecond)),
		retry.WithMax(3),
		retry.WithPerRetryTimeout(defaultTimeout),
		retry.WithCodes(codes.Unavailable, codes.ResourceExhausted),
	}

	return []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithStatsHandler(otelgrpc.NewClientHandler()),
		grpc.WithDefaultServiceConfig(`{"loadBalancingPolicy": "round_robin"}`),
		grpc.WithChainUnaryInterceptor(
			idempotencyKeyToMetadataInterceptor(),
			timeout.UnaryClientInterceptor(defaultTimeout),
			NewBreakerInterceptor().UnaryClientInterceptor,
			retry.UnaryClientInterceptor(retryOpts...),
		),
	}
}
