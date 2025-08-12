package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/hmmm42/gorder-v2/common/contextkeys"
)

// IdempotencyMiddleware 是一个 Gin 中间件.
// 它从请求头中提取 "Idempotency-Key", 并将其注入到请求的 context 中.
func IdempotencyMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		idempotencyKey := c.GetHeader("Idempotency-Key")

		if idempotencyKey != "" {
			// 使用共享包的函数来创建包含新值的 context
			ctx := contextkeys.NewContext(c.Request.Context(), idempotencyKey)
			// 用这个新的 context 替换掉请求中原有的 context
			c.Request = c.Request.WithContext(ctx)
		}

		// 继续处理请求链
		c.Next()
	}
}
