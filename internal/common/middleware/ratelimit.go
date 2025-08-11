package middleware

import (
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"golang.org/x/time/rate"
)

// IPRateLimiter 包含一个IP地址到速率限制器的映射以及一个互斥锁
type IPRateLimiter struct {
	mu       sync.Mutex
	limiters map[string]*limiterEntry
	r        rate.Limit // 每秒允许的事件速率
	b        int        // 桶的大小
}

// limiterEntry 包含一个限流器和最后一次被看到的时间
type limiterEntry struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// NewIPRateLimiter 创建一个新的IP速率限制器
func NewIPRateLimiter(r rate.Limit, b int) *IPRateLimiter {
	ipl := &IPRateLimiter{
		limiters: make(map[string]*limiterEntry),
		r:        r,
		b:        b,
	}

	// 启动一个后台goroutine来清理旧的条目
	go ipl.cleanupLimiters()

	return ipl
}

// getLimiter 为给定的IP地址返回限流器，如果不存在则创建一个
func (i *IPRateLimiter) getLimiter(ip string) *rate.Limiter {
	i.mu.Lock()
	defer i.mu.Unlock()

	entry, exists := i.limiters[ip]
	if !exists {
		limiter := rate.NewLimiter(i.r, i.b)
		entry = &limiterEntry{limiter: limiter}
		i.limiters[ip] = entry
	}

	entry.lastSeen = time.Now()
	return entry.limiter
}

// cleanupLimiters 定期检查并删除超过3分钟未使用的限流器
func (i *IPRateLimiter) cleanupLimiters() {
	for {
		time.Sleep(time.Minute)

		i.mu.Lock()
		for ip, entry := range i.limiters {
			if time.Since(entry.lastSeen) > 3*time.Minute {
				delete(i.limiters, ip)
			}
		}
		i.mu.Unlock()
	}
}

// CreateRateLimitMiddleware 创建一个HTTP中间件，用于实现IP限流
func CreateRateLimitMiddleware(r rate.Limit, b int) func(http.Handler) http.Handler {
	ipLimiter := NewIPRateLimiter(r, b)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip, _, err := net.SplitHostPort(r.RemoteAddr)
			if err != nil {
				logrus.Infof("could not get ip from remote addr %s: %v", r.RemoteAddr, err)
				ip = r.RemoteAddr
			}

			limiter := ipLimiter.getLimiter(ip)
			if !limiter.Allow() {
				http.Error(w, http.StatusText(http.StatusTooManyRequests), http.StatusTooManyRequests)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
