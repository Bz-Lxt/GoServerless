package cache

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/gogo/goserverless/internal/logger"
)

type Redis struct {
	c *redis.Client
}

func Connect(addr string) (*Redis, error) {
	c := redis.NewClient(&redis.Options{
		Addr:         addr,
		DialTimeout:  3 * time.Second,
		ReadTimeout:  2 * time.Second,
		WriteTimeout: 2 * time.Second,
	})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := c.Ping(ctx).Err(); err != nil {
		return nil, err
	}
	return &Redis{c: c}, nil
}

func (r *Redis) Close() error {
	if r == nil || r.c == nil {
		return nil
	}
	return r.c.Close()
}

func (r *Redis) Incr(ctx context.Context, key string) {
	if r == nil {
		return
	}
	if err := r.c.Incr(ctx, key).Err(); err != nil {
		logger.Debug(ctx, "redis incr failed", "key", key, "err", err)
	}
}

func (r *Redis) GetInt(ctx context.Context, key string) int64 {
	if r == nil {
		return 0
	}
	n, err := r.c.Get(ctx, key).Int64()
	if err != nil {
		return 0
	}
	return n
}

func (r *Redis) SetRoute(ctx context.Context, name, status string) {
	if r == nil {
		return
	}
	_ = r.c.Set(ctx, "route:"+name, status, 10*time.Minute).Err()
}

func (r *Redis) DelRoute(ctx context.Context, name string) {
	if r == nil {
		return
	}
	_ = r.c.Del(ctx, "route:"+name).Err()
}
