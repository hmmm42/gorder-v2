package contextkeys

import "context"

// 定义一个私有的 key 类型, 这是 Go 语言的最佳实践, 防止键名冲突
type idempotencyKey struct{}

const IdempotencyMetadataKey = "idempotency-key"

// NewContext 将幂等键的值存入一个新的 context 中
func NewContext(ctx context.Context, key string) context.Context {
	return context.WithValue(ctx, idempotencyKey{}, key)
}

// FromContext 从 context 中安全地提取幂等键的值
func FromContext(ctx context.Context) (string, bool) {
	key, ok := ctx.Value(idempotencyKey{}).(string)
	return key, ok
}
