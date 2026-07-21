package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type Cache struct {
	client *redis.Client
}

func New(addr string) *Cache {
	rdb := redis.NewClient(&redis.Options{
		Addr: addr,
	})
	return &Cache{client: rdb}
}

func (c *Cache) Close() error {
	return c.client.Close()
}

func (c *Cache) Ping(ctx context.Context) error {
	return c.client.Ping(ctx).Err()
}

func (c *Cache) Get(ctx context.Context, key string, dest any) error {
	val, err := c.client.Get(ctx, key).Bytes()
	if err != nil {
		return err
	}
	return json.Unmarshal(val, dest)
}

func (c *Cache) Set(ctx context.Context, key string, val any, ttl time.Duration) error {
	data, err := json.Marshal(val)
	if err != nil {
		return fmt.Errorf("marshal cache: %w", err)
	}
	return c.client.Set(ctx, key, data, ttl).Err()
}

func (c *Cache) Del(ctx context.Context, keys ...string) error {
	return c.client.Del(ctx, keys...).Err()
}

// Key helpers
func AccountKey(id int64) string  { return fmt.Sprintf("account:%d", id) }
func CustomerKey(id int64) string { return fmt.Sprintf("customer:%d", id) }
