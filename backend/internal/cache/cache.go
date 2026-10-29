// Package cache provides short-lived cache, distributed locks, and rate limiting.
package cache

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"
	"github.com/tafakorpardazan/huginn/internal/config"
)

// Client wraps go-redis.
type Client struct {
	c *goredis.Client
}

func New(ctx context.Context, cfg config.RedisConfig) (*Client, error) {
	c := goredis.NewClient(&goredis.Options{
		Addr:     cfg.Addr,
		Password: cfg.Password,
		DB:       cfg.DB,
	})
	if err := c.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("redis ping: %w", err)
	}
	return &Client{c: c}, nil
}

func (r *Client) Ping(ctx context.Context) error { return r.c.Ping(ctx).Err() }

func (r *Client) Close() error { return r.c.Close() }

func (r *Client) Raw() *goredis.Client { return r.c }

// Get/Set/Delete short-lived cache.
func (r *Client) Get(ctx context.Context, key string) (string, error) {
	return r.c.Get(ctx, key).Result()
}

func (r *Client) Set(ctx context.Context, key, value string, ttl time.Duration) error {
	return r.c.Set(ctx, key, value, ttl).Err()
}

func (r *Client) Delete(ctx context.Context, keys ...string) error {
	return r.c.Del(ctx, keys...).Err()
}

// Lock is a distributed lock with TTL. Returns false if already held.
func (r *Client) Lock(ctx context.Context, key string, ttl time.Duration) (bool, error) {
	return r.c.SetNX(ctx, "lock:"+key, uuid.NewString(), ttl).Result()
}

// Unlock releases a lock. Safe to call even if not held.
func (r *Client) Unlock(ctx context.Context, key string) error {
	return r.c.Del(ctx, "lock:"+key).Err()
}

// WithLock runs fn while holding the distributed lock.
func (r *Client) WithLock(ctx context.Context, key string, ttl time.Duration, fn func() error) error {
	ok, err := r.Lock(ctx, key, ttl)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("lock held: %s", key)
	}
	defer r.Unlock(ctx, key)
	return fn()
}

// Allow reports whether a request is permitted under a fixed-window rate limit.
func (r *Client) Allow(ctx context.Context, key string, limit int, window time.Duration) (bool, error) {
	now := time.Now()
	windowKey := now.Truncate(window).Unix()
	full := fmt.Sprintf("rl:%s:%d", key, windowKey)
	n, err := r.c.Incr(ctx, full).Result()
	if err != nil {
		return false, err
	}
	if n == 1 {
		r.c.Expire(ctx, full, window*2)
	}
	return n <= int64(limit), nil
}
