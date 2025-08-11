package redis

import (
	"context"
	"errors"
	"time"

	"github.com/hmmm42/gorder-v2/common/logging"
	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
)

func SetNX(ctx context.Context, client *redis.Client, key, value string, ttl time.Duration) (err error) {
	now := time.Now()
	defer func() {
		l := logrus.WithContext(ctx).WithFields(logrus.Fields{
			"start":       now,
			"key":         key,
			"value":       value,
			logging.Error: err,
			logging.Cost:  time.Since(now).Milliseconds(),
		})
		if err == nil {
			l.Info("_redis_setnx_success")
		} else {
			l.Warn("_redis_setnx_error")
		}
	}()
	if client == nil {
		return errors.New("redis client is nil")
	}
	_, err = client.SetNX(ctx, key, value, ttl).Result()
	return err
}

func Del(ctx context.Context, client *redis.Client, key string) (err error) {
	now := time.Now()
	defer func() {
		l := logrus.WithContext(ctx).WithFields(logrus.Fields{
			"start":       now,
			"key":         key,
			logging.Error: err,
			logging.Cost:  time.Since(now).Milliseconds(),
		})
		if err == nil {
			l.Info("_redis_del_success")
		} else {
			l.Warn("_redis_del_error")
		}
	}()
	if client == nil {
		return errors.New("redis client is nil")
	}
	_, err = client.Del(ctx, key).Result()
	return err
}

func Incr(ctx context.Context, client *redis.Client, key string) (int64, error) {
	now := time.Now()
	var err error
	defer func() {
		l := logrus.WithContext(ctx).WithFields(logrus.Fields{
			"start":       now,
			"key":         key,
			logging.Error: err,
			logging.Cost:  time.Since(now).Milliseconds(),
		})
		if err == nil {
			l.Info("_redis_incr_success")
		} else {
			l.Warn("_redis_incr_error")
		}
	}()
	if client == nil {
		return 0, errors.New("redis client is nil")
	}
	return client.Incr(ctx, key).Result()
}

func Decr(ctx context.Context, client *redis.Client, key string) {
	now := time.Now()
	var err error
	defer func() {
		l := logrus.WithContext(ctx).WithFields(logrus.Fields{
			"start":       now,
			"key":         key,
			logging.Error: err,
			logging.Cost:  time.Since(now).Milliseconds(),
		})
		if err == nil {
			l.Info("_redis_decr_success")
		} else {
			l.Warn("_redis_decr_error")
		}
	}()
	//if client == nil {
	//	return 0, errors.New("redis client is nil")
	//}
	client.Decr(ctx, key)
	//return client.Decr(ctx, key).Result()
}

func Expire(ctx context.Context, client *redis.Client, key string, ttl time.Duration) (bool, error) {
	now := time.Now()
	var err error
	defer func() {
		l := logrus.WithContext(ctx).WithFields(logrus.Fields{
			"start":      now,
			"key":        key,
			"ttl":        ttl,
			logging.Cost: time.Since(now).Milliseconds(),
		})
		if err == nil {
			l.Info("_redis_expire_success")
		} else {
			l.Warn("_redis_expire_error")
		}
	}()
	if client == nil {
		return false, errors.New("redis client is nil")
	}
	return client.Expire(ctx, key, ttl).Result()
}
