package balancer

import (
	"errors"
	"testing"
	"time"

	"google.golang.org/grpc/balancer"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestNodeCircuitBreaker(t *testing.T) {
	t.Run("初始状态为关闭", func(t *testing.T) {
		breaker := newNodeCircuitBreaker()
		if !breaker.Allow() {
			t.Error("新创建的熔断器应该允许请求")
		}
	})

	t.Run("连续失败后熔断器打开", func(t *testing.T) {
		breaker := newNodeCircuitBreaker()

		// 模拟连续失败
		for i := 0; i < failureThreshold; i++ {
			breaker.RecordFailure()
		}

		// 熔断器应该打开
		if breaker.Allow() {
			t.Error("连续失败后熔断器应该打开，拒绝请求")
		}
	})

	t.Run("半开状态下成功请求关闭熔断器", func(t *testing.T) {
		breaker := newNodeCircuitBreaker()

		// 触发熔断器打开
		for i := 0; i < failureThreshold; i++ {
			breaker.RecordFailure()
		}

		// 等待超时，应该进入半开状态
		time.Sleep(openStateTimeout + 1*time.Second)

		// 现在应该允许请求（半开状态）
		if !breaker.Allow() {
			t.Error("超时后熔断器应该进入半开状态，允许请求")
		}

		// 模拟成功请求
		for i := 0; i < successThreshold; i++ {
			breaker.RecordSuccess()
		}

		// 熔断器应该关闭
		if !breaker.Allow() {
			t.Error("连续成功后熔断器应该关闭，允许请求")
		}
	})
}

func TestIsSystemError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{"nil错误", nil, false},
		{"Unavailable错误", status.Error(codes.Unavailable, "service unavailable"), true},
		{"DeadlineExceeded错误", status.Error(codes.DeadlineExceeded, "timeout"), true},
		{"Internal错误", status.Error(codes.Internal, "internal error"), true},
		{"ResourceExhausted错误", status.Error(codes.ResourceExhausted, "rate limit"), true},
		{"DataLoss错误", status.Error(codes.DataLoss, "data loss"), true},
		{"InvalidArgument错误", status.Error(codes.InvalidArgument, "bad request"), false},
		{"NotFound错误", status.Error(codes.NotFound, "not found"), false},
		{"非gRPC错误", errors.New("network error"), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isSystemError(tt.err); got != tt.expected {
				t.Errorf("isSystemError() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestBreakerBalancerRegistration(t *testing.T) {
	t.Run("balancer注册验证", func(t *testing.T) {
		// 验证我们的 balancer 已经注册
		builder := balancer.Get(Name)
		if builder == nil {
			t.Errorf("balancer %s 没有注册", Name)
		}

		// 验证名称正确
		if builder.Name() != Name {
			t.Errorf("balancer 名称错误，期望 %s，得到 %s", Name, builder.Name())
		}
	})
}

func TestBreakerStates(t *testing.T) {
	t.Run("状态转换测试", func(t *testing.T) {
		breaker := newNodeCircuitBreaker()

		// 初始状态：关闭
		if breaker.state != StateClosed {
			t.Error("初始状态应该是关闭")
		}

		// 连续失败导致打开
		for i := 0; i < failureThreshold; i++ {
			breaker.RecordFailure()
		}
		if breaker.state != StateOpen {
			t.Error("连续失败后状态应该是打开")
		}

		// 超时后进入半开状态
		time.Sleep(openStateTimeout + 100*time.Millisecond)
		breaker.Allow() // 这会触发状态检查
		if breaker.state != StateHalfOpen {
			t.Error("超时后状态应该是半开")
		}
	})
}
