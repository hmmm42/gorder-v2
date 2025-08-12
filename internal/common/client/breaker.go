package client

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/sony/gobreaker"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// BreakerInterceptor 包含熔断器实例
type BreakerInterceptor struct {
	breakers map[string]*gobreaker.CircuitBreaker
	mutex    sync.Mutex
}

// NewBreakerInterceptor 创建一个新的熔断拦截器
func NewBreakerInterceptor() *BreakerInterceptor {
	return &BreakerInterceptor{
		breakers: make(map[string]*gobreaker.CircuitBreaker),
	}
}

// UnaryClientInterceptor 是 gRPC 客户端的一元拦截器实现
func (bi *BreakerInterceptor) UnaryClientInterceptor(
	ctx context.Context,
	method string,
	req, reply interface{},
	cc *grpc.ClientConn,
	invoker grpc.UnaryInvoker,
	opts ...grpc.CallOption,
) error {
	cb := bi.getBreaker(method)

	var originalErr error
	_, err := cb.Execute(func() (interface{}, error) {
		originalErr = invoker(ctx, method, req, reply, cc, opts...)
		if isSystemError(originalErr) {
			return nil, originalErr // 告诉熔断器这是一个失败
		}
		return nil, nil // 告诉熔断器这是一个成功
	})

	// 如果 err 不是 nil，意味着熔断器跳闸了 (例如状态为 Open 或 Half-Open 时拒绝请求)
	if err != nil {
		// 将 gobreaker 的错误转换为 gRPC 错误，以便客户端能够理解
		if errors.Is(err, gobreaker.ErrOpenState) || errors.Is(err, gobreaker.ErrTooManyRequests) {
			return status.Errorf(codes.Unavailable, "circuit breaker is open for method %s", method)
		}
		// 在其他情况下，返回 gobreaker 可能产生的其他错误
		return status.Errorf(codes.Internal, "circuit breaker execution error: %v", err)
	}

	// 返回原始的 gRPC 调用错误
	return originalErr
}

// getBreaker 获取或创建一个新的熔断器实例
func (bi *BreakerInterceptor) getBreaker(method string) *gobreaker.CircuitBreaker {
	bi.mutex.Lock()
	defer bi.mutex.Unlock()

	if cb, ok := bi.breakers[method]; ok {
		return cb
	}

	st := gobreaker.Settings{
		Name:        method,
		MaxRequests: 3,                // 半开状态下允许的请求数
		Interval:    5 * time.Second,  // 从关闭状态到打开状态的重置周期
		Timeout:     10 * time.Second, // 从打开状态到半开状态的冷却时间
		ReadyToTrip: func(counts gobreaker.Counts) bool { // 定义何时跳闸
			// 连续失败5次，或者在超过10个请求的情况下失败率达到60%
			return counts.ConsecutiveFailures > 5 ||
				(counts.Requests >= 10 && float64(counts.TotalFailures)/float64(counts.Requests) >= 0.6)
		},
	}

	cb := gobreaker.NewCircuitBreaker(st)
	bi.breakers[method] = cb
	return cb
}

// isSystemError 判断一个错误是否应被视为系统级故障
func isSystemError(err error) bool {
	if err == nil {
		return false
	}
	st, ok := status.FromError(err)
	if !ok {
		// 如果不是一个标准的 gRPC 状态错误，可能是一个网络错误等，我们将其视为系统错误
		return true
	}
	switch st.Code() {
	// 这些错误码通常表示服务端或网络问题，而不是客户端请求本身的问题
	case codes.Unavailable, codes.DeadlineExceeded, codes.Internal, codes.ResourceExhausted, codes.DataLoss:
		return true
	default:
		return false
	}
}
