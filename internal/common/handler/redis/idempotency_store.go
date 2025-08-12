package redis

import (
	"context"
	"time"

	"github.com/hmmm42/gorder-v2/common/decorator"
	"github.com/pkg/errors"
	"github.com/redis/go-redis/v9"
)

// IdempotencyStore Redis实现的幂等性存储
type IdempotencyStore struct {
	client redis.UniversalClient
}

func NewRedisIdempotencyStore(client redis.UniversalClient) decorator.IdempotencyStore {
	return &IdempotencyStore{
		client: client,
	}
}

func (r *IdempotencyStore) Get(ctx context.Context, key string) ([]byte, error) {
	result, err := r.client.Get(ctx, key).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, nil // 键不存在
		}
		return nil, errors.Wrap(err, "failed to get from redis")
	}
	return []byte(result), nil
}

func (r *IdempotencyStore) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	err := r.client.Set(ctx, key, string(value), ttl).Err()
	if err != nil {
		return errors.Wrap(err, "failed to set to redis")
	}
	return nil
}

func (r *IdempotencyStore) Exists(ctx context.Context, key string) (bool, error) {
	result, err := r.client.Exists(ctx, key).Result()
	if err != nil {
		return false, errors.Wrap(err, "failed to check existence in redis")
	}
	return result > 0, nil
}
