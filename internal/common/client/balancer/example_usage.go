package balancer

import (
	"fmt"
	"log"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// ExampleNodeLevelCircuitBreaker 展示如何使用节点级别熔断器
func ExampleNodeLevelCircuitBreaker() {
	// 这个示例展示了如何配置 gRPC 客户端使用节点级别熔断器

	// gRPC 连接配置 - 使用我们的 node_breaker 负载均衡器
	conn, err := grpc.NewClient(
		"consul://localhost:8500/stock-service", // 假设使用 Consul 服务发现
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		// 关键配置：使用节点级别熔断器
		grpc.WithDefaultServiceConfig(`{
			"loadBalancingPolicy": "node_breaker"
		}`),
	)
	if err != nil {
		log.Fatalf("连接失败: %v", err)
	}
	defer conn.Close()

	// 现在每个下游服务实例都有自己独立的熔断器
	// 例如如果有三个 stock 服务实例：
	// - stock-A (192.168.1.10:9090) -> 独立的熔断器 A
	// - stock-B (192.168.1.11:9090) -> 独立的熔断器 B
	// - stock-C (192.168.1.12:9090) -> 独立的熔断器 C

	// 如果 stock-A 出现问题，只有该实例的熔断器会打开
	// stock-B 和 stock-C 仍然可以正常处理请求

	fmt.Println("节点级别熔断器配置完成")
	fmt.Println("特性:")
	fmt.Println("- 每个服务实例维护独立的熔断器状态")
	fmt.Println("- 连续失败 5 次后熔断器打开")
	fmt.Println("- 熔断器打开后等待 10 秒进入半开状态")
	fmt.Println("- 半开状态下连续成功 3 次后熔断器关闭")
}

// CircuitBreakerConfig 熔断器配置参数
type CircuitBreakerConfig struct {
	FailureThreshold int           // 失败阈值
	SuccessThreshold int           // 成功阈值
	OpenStateTimeout time.Duration // 打开状态超时时间
}

// GetDefaultConfig 获取默认配置
func GetDefaultConfig() CircuitBreakerConfig {
	return CircuitBreakerConfig{
		FailureThreshold: failureThreshold, // 5
		SuccessThreshold: successThreshold, // 3
		OpenStateTimeout: openStateTimeout, // 10秒
	}
}

// NodeLevelBreakerUsageNotes 节点级别熔断器使用说明
const NodeLevelBreakerUsageNotes = `
节点级别熔断器使用说明：

1. 自动注册：
   导入 balancer 包时会自动注册 "node_breaker" 负载均衡器

2. 配置客户端：
   使用 grpc.WithDefaultServiceConfig 配置负载均衡策略为 "node_breaker"

3. 熔断器行为：
   - 每个下游服务实例独立维护熔断器状态
   - 只有系统级错误才会触发熔断器（网络错误、超时、服务不可用等）
   - 业务错误（如参数错误、未找到资源）不会触发熔断器

4. 状态转换：
   关闭 -> (连续失败5次) -> 打开 -> (等待10秒) -> 半开 -> (连续成功3次) -> 关闭

5. 优势：
   - 故障隔离：单个实例故障不影响其他健康实例
   - 细粒度控制：针对每个实例的熔断状态
   - 快速恢复：健康实例立即可用，故障实例逐步恢复

6. 与方法级别熔断器的区别：
   - 方法级别：以 gRPC 方法为单位进行熔断
   - 节点级别：以服务实例为单位进行熔断
   - 可以同时使用两种熔断器实现多层保护
`
