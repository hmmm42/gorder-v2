package server

import (
	"context"
	"fmt"
	"net"

	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/recovery"
	"github.com/hmmm42/gorder-v2/common/contextkeys"
	"github.com/hmmm42/gorder-v2/common/logging"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	log "github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/logging"
)

// v2 需要一个适配器将 logrus.Entry 转换为其期望的 Logger 接口
func interceptorLogger(l *logrus.Entry) log.Logger {
	return log.LoggerFunc(func(ctx context.Context, lvl log.Level, msg string, fields ...any) {
		f := make(map[string]any, len(fields)/2)
		i := log.Fields(fields).Iterator()
		for i.Next() {
			k, v := i.At()
			f[k] = v
		}
		l := l.WithFields(f)

		switch lvl {
		case log.LevelDebug:
			l.Debug(msg)
		case log.LevelInfo:
			l.Info(msg)
		case log.LevelWarn:
			l.Warn(msg)
		case log.LevelError:
			l.Error(msg)
		default:
			panic(fmt.Sprintf("unknown level %v", lvl))
		}
	})
}

func RunGRPCServer(serviceName string, registerServer func(server *grpc.Server)) {
	addr := viper.Sub(serviceName).GetString("grpc-addr")
	if addr == "" {
		addr = viper.GetString("fallback-grpc-addr")
	}
	RunGRPCServerOnAddr(addr, registerServer)
}

func RunGRPCServerOnAddr(addr string, registerServer func(server *grpc.Server)) {
	logrusEntry := logrus.NewEntry(logrus.StandardLogger())
	logEvents := []log.LoggableEvent{
		log.StartCall, log.FinishCall,
	}

	grpcServer := grpc.NewServer(
		grpc.StatsHandler(otelgrpc.NewServerHandler()),
		grpc.ChainUnaryInterceptor(
			recovery.UnaryServerInterceptor(),
			log.UnaryServerInterceptor(interceptorLogger(logrusEntry), log.WithLogOnEvents(logEvents...)),
			logging.GRPCUnaryInterceptor,
			IdempotencyExtractorInterceptor(),
		),
		grpc.ChainStreamInterceptor(
			recovery.StreamServerInterceptor(),
			log.StreamServerInterceptor(interceptorLogger(logrusEntry), log.WithLogOnEvents(logEvents...)),
		),
	)
	registerServer(grpcServer)

	listen, err := net.Listen("tcp", addr)
	if err != nil {
		logrus.Panic(err)
	}
	logrus.Infof("Starting gRPC server, Listening: %s", addr)
	if err := grpcServer.Serve(listen); err != nil {
		logrus.Panic(err)
	}
}

func IdempotencyExtractorInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		md, ok := metadata.FromIncomingContext(ctx)
		if ok {
			keys := md.Get(contextkeys.IdempotencyMetadataKey)
			if len(keys) > 0 {
				// 从 gRPC 的传入元数据中读出值,
				// 并把它写入一个新的 context, 这个 context 将被传递给业务代码.
				ctx = contextkeys.NewContext(ctx, keys[0])
			}
		}
		// 把包含了新值的 ctx 传递下去
		return handler(ctx, req)
	}
}
