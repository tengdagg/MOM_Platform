package data

import (
	"context"
	"encoding/json"
	"time"

	"github.com/redis/go-redis/v9"
)

// Cache Redis 缓存工具，封装常用操作
type Cache struct {
	client *redis.Client
}

// NewCache 创建缓存工具实例
func NewCache(client *redis.Client) *Cache {
	if client == nil {
		return nil
	}
	return &Cache{client: client}
}

// Get 从缓存读取并反序列化到目标类型
func (c *Cache) Get(ctx context.Context, key string, dest interface{}) bool {
	if c == nil || c.client == nil {
		return false
	}
	val, err := c.client.Get(ctx, key).Result()
	if err != nil {
		return false
	}
	if err := json.Unmarshal([]byte(val), dest); err != nil {
		return false
	}
	return true
}

// Set 将值序列化后写入缓存
func (c *Cache) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) {
	if c == nil || c.client == nil {
		return
	}
	data, err := json.Marshal(value)
	if err != nil {
		return
	}
	c.client.Set(ctx, key, string(data), ttl)
}

// Del 删除指定的缓存键
func (c *Cache) Del(ctx context.Context, keys ...string) {
	if c == nil || c.client == nil || len(keys) == 0 {
		return
	}
	c.client.Del(ctx, keys...)
}

// DelByPrefix 按前缀批量删除缓存键（使用 SCAN 避免阻塞）
func (c *Cache) DelByPrefix(ctx context.Context, prefix string) {
	if c == nil || c.client == nil || prefix == "" {
		return
	}
	var cursor uint64
	for {
		keys, nextCursor, err := c.client.Scan(ctx, cursor, prefix+"*", 100).Result()
		if err != nil {
			return
		}
		if len(keys) > 0 {
			c.client.Del(ctx, keys...)
		}
		cursor = nextCursor
		if cursor == 0 {
			break
		}
	}
}

// Available 返回缓存是否可用
func (c *Cache) Available() bool {
	return c != nil && c.client != nil
}
