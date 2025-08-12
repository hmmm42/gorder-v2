package client

import (
	"context"

	"github.com/hmmm42/gorder-v2/common/contextkeys"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

// idempotencyKeyToMetadataInterceptor 是一个客户端拦截器.
// 它从 context 中读取幂等键并将其放入传出的 gRPC metadata.
func idempotencyKeyToMetadataInterceptor() grpc.UnaryClientInterceptor {
	return func(
		ctx context.Context,
		method string,
		req, reply interface{},
		cc *grpc.ClientConn,
		invoker grpc.UnaryInvoker,
		opts ...grpc.CallOption,
	) error {
		// 使用共享包的函数从 context 中直接读取值
		key, ok := contextkeys.FromContext(ctx)
		if ok && key != "" {
			// 如果成功读取, 就把它加入到传出的元数据中
			ctx = metadata.AppendToOutgoingContext(ctx, contextkeys.IdempotencyMetadataKey, key)
		}
		// 继续调用链, 把包含了元数据的 ctx 传下去
		return invoker(ctx, method, req, reply, cc, opts...)
	}
}
