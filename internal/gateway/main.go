package main

import (
	"context"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"

	_ "github.com/hmmm42/gorder-v2/common/config"
	"github.com/hmmm42/gorder-v2/common/discovery"
	"github.com/hmmm42/gorder-v2/common/logging"
	"github.com/hmmm42/gorder-v2/common/middleware"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"golang.org/x/time/rate"
)

// ServiceDiscoverer 定义了一个服务发现器的接口
type ServiceDiscoverer interface {
	// Pick 方法从指定服务的所有健康实例中选择一个
	Pick(serviceName string) (string, error)
}

// consulDiscoverer 是一个基于 Consul 的服务发现器实现
type consulDiscoverer struct{}

func (c *consulDiscoverer) Pick(serviceName string) (string, error) {
	// 复用您现有的 GetServiceAddr 函数, 它已经实现了从 Consul 发现并随机选择一个实例的逻辑
	return discovery.GetServiceAddr(context.Background(), serviceName)
}

func init() {
	logging.Init()
}

func main() {
	discoverer := &consulDiscoverer{}

	// 创建一个反向代理, 并为其配置一个动态的 Director
	proxy := &httputil.ReverseProxy{
		Director: func(req *http.Request) {
			targetService := "order-http"

			// 从服务发现器中挑选一个健康的目标实例地址
			targetAddr, err := discoverer.Pick(targetService)
			if err != nil {
				log.Printf("Failed to discover service %s: %v", targetService, err)
				return
			}

			log.Printf("Forwarding request for %s to %s", req.URL.Path, targetAddr)

			// 重写请求的目标地址
			targetUrl, _ := url.Parse("http://" + targetAddr)
			req.URL.Scheme = targetUrl.Scheme
			req.URL.Host = targetUrl.Host
			req.Host = targetUrl.Host // 关键: 必须设置 req.Host
		},
	}

	// 创建一个限流中间件：每个IP每秒2个请求，桶容量为5
	rateLimitMiddleware := middleware.CreateRateLimitMiddleware(rate.Limit(2), 5)

	// 使用限流中间件包装代理处理器
	http.Handle("/", rateLimitMiddleware(proxy))

	gatewayAddr := viper.Sub("gateway").GetString("http-addr")
	logrus.Infof("Starting gateway server on %s", gatewayAddr)
	if err := http.ListenAndServe(gatewayAddr, nil); err != nil {
		log.Fatal(err)
	}
}
