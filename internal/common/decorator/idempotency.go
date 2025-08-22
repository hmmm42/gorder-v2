package decorator

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/hmmm42/gorder-v2/common/contextkeys"

	"github.com/pkg/errors"
)

// IdempotencyStore 存储幂等性结果的接口
type IdempotencyStore interface {
	Get(ctx context.Context, key string) ([]byte, error)
	Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
	Exists(ctx context.Context, key string) (bool, error)
}

// IdempotentRequest 需要幂等性检查的请求接口
type IdempotentRequest interface {
	GetIdempotencyKey() string
}

// IdempotentResult 幂等性结果包装
type IdempotentResult[R any] struct {
	Result R      `json:"result"`
	Error  string `json:"error,omitempty"`
}

type idempotencyDecorator[C, R any] struct {
	base      CommandHandler[C, R]
	store     IdempotencyStore
	ttl       time.Duration
	keyPrefix string
}

func (d idempotencyDecorator[C, R]) Handle(ctx context.Context, cmd C) (result R, err error) {
	key, exists := contextkeys.FromContext(ctx)
	if !exists || key == "" {
		return result, errors.New("idempotency key not found in context")
	}

	fullKey := fmt.Sprintf("%s:%s", d.keyPrefix, key)

	// 检查是否已经存在结果
	data, err := d.store.Get(ctx, fullKey)
	if err == nil && data != nil {
		// 反序列化已存在的结果
		var cachedResult IdempotentResult[R]
		if err := json.Unmarshal(data, &cachedResult); err == nil {
			if cachedResult.Error != "" {
				return result, errors.New(cachedResult.Error)
			}
			return cachedResult.Result, nil
		}
	}

	// 执行实际的业务逻辑
	result, err = d.base.Handle(ctx, cmd)

	// 缓存结果
	cachedResult := IdempotentResult[R]{
		Result: result,
	}
	if err != nil {
		cachedResult.Error = err.Error()
	}

	if data, marshalErr := json.Marshal(cachedResult); marshalErr == nil {
		_ = d.store.Set(ctx, fullKey, data, d.ttl)
	}

	return result, err
}

// IdempotencyOptions 幂等性配置选项
type IdempotencyOptions struct {
	Store     IdempotencyStore
	TTL       time.Duration
	KeyPrefix string
}

// WithCommandIdempotency 为Command添加幂等性装饰器
func WithCommandIdempotency[C, R any](handler CommandHandler[C, R], opts IdempotencyOptions) CommandHandler[C, R] {
	if opts.Store == nil {
		panic("IdempotencyStore cannot be nil")
	}
	if opts.TTL == 0 {
		opts.TTL = 5 * time.Minute // 默认TTL为5分钟
	}
	if opts.KeyPrefix == "" {
		opts.KeyPrefix = "idempotency"
	}
	return idempotencyDecorator[C, R]{
		base:      handler,
		store:     opts.Store,
		ttl:       opts.TTL,
		keyPrefix: opts.KeyPrefix,
	}
}

func WithQueryIdempotency[Q, R any](handler QueryHandler[Q, R], opts IdempotencyOptions) QueryHandler[Q, R] {
	if opts.Store == nil {
		panic("IdempotencyStore cannot be nil")
	}
	if opts.TTL == 0 {
		opts.TTL = 5 * time.Minute // 默认TTL为5分钟
	}
	if opts.KeyPrefix == "" {
		opts.KeyPrefix = "idempotency:query"
	}
	return idempotencyDecorator[Q, R]{
		base:      handler,
		store:     opts.Store,
		ttl:       opts.TTL,
		keyPrefix: opts.KeyPrefix,
	}
}
