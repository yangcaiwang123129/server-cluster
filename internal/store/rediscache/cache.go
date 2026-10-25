// Package rediscache 共享读缓存（Redis Hash）：Mesh 侧 L2，Node 默认关闭。
// 防穿透：miss 后写入 _nil 空值标记（短 TTL）；防雪崩：TTL 加随机抖动。
package rediscache

import (
	"context"
	"fmt"
	"math/rand/v2"
	"time"

	"github.com/redis/go-redis/v9"
)

// 缓存数据类别（键尾段）。
const (
	KindCore    = "core"
	KindAssets  = "assets"
	KindProfile = "profile"
)

// nullField 空值标记字段，防穿透。
const nullField = "_nil"

// nullTTL 空值标记 TTL（短）。
const nullTTL = 30 * time.Second

// Config 缓存配置；连接实例由 db 包命名实例管理，此处不再持有连接参数。
type Config struct {
	KeyPrefix         string        // 键前缀，默认 player
	TTL               time.Duration // 默认 15m
	TTLJitter         time.Duration // TTL 随机抖动上限，默认 300s
	InvalidateChannel string        // 失效广播频道
}

// Cache 共享读缓存。
type Cache struct {
	rdb     redis.UniversalClient
	prefix  string
	ttl     time.Duration
	jitter  time.Duration
	channel string
}

// New 构造缓存；client 由调用方（db.RedisManager）注入，生命周期由调用方管理。
func New(client redis.UniversalClient, cfg Config) *Cache {
	if cfg.KeyPrefix == "" {
		cfg.KeyPrefix = "player"
	}
	if cfg.TTL <= 0 {
		cfg.TTL = 15 * time.Minute
	}
	if cfg.TTLJitter <= 0 {
		cfg.TTLJitter = 300 * time.Second
	}
	return &Cache{
		rdb:     client,
		prefix:  cfg.KeyPrefix,
		ttl:     cfg.TTL,
		jitter:  cfg.TTLJitter,
		channel: cfg.InvalidateChannel,
	}
}

// Key 缓存键：{prefix}:{playerID}:{kind}。
func (c *Cache) Key(playerID int64, kind string) string {
	return fmt.Sprintf("%s:%d:%s", c.prefix, playerID, kind)
}

// HashResult HGETALL 结果。
type HashResult struct {
	Fields map[string]string // 命中时的全部字段
	Found  bool              // 命中真实数据
	Null   bool              // 命中空值标记（数据不存在，防穿透）
}

// GetHash 读取整个 Hash。
func (c *Cache) GetHash(ctx context.Context, playerID int64, kind string) (*HashResult, error) {
	fields, err := c.rdb.HGetAll(ctx, c.Key(playerID, kind)).Result()
	if err != nil {
		return nil, err
	}
	if len(fields) == 0 {
		return &HashResult{}, nil
	}
	if _, ok := fields[nullField]; ok {
		return &HashResult{Null: true}, nil
	}
	return &HashResult{Fields: fields, Found: true}, nil
}

// SetHash 写入 Hash 并设置带抖动的 TTL。
func (c *Cache) SetHash(ctx context.Context, playerID int64, kind string, fields map[string]string) error {
	key := c.Key(playerID, kind)
	ttl := c.ttl
	if c.jitter > 0 {
		ttl += time.Duration(rand.Int64N(int64(c.jitter)))
	}
	pipe := c.rdb.Pipeline()
	pipe.HSet(ctx, key, fields)
	pipe.Expire(ctx, key, ttl)
	_, err := pipe.Exec(ctx)
	return err
}

// SetNull 写入空值标记（短 TTL），防止缓存穿透。
func (c *Cache) SetNull(ctx context.Context, playerID int64, kind string) error {
	key := c.Key(playerID, kind)
	pipe := c.rdb.Pipeline()
	pipe.HSet(ctx, key, nullField, "1")
	pipe.Expire(ctx, key, nullTTL)
	_, err := pipe.Exec(ctx)
	return err
}

// Delete 删除指定类别的缓存。
func (c *Cache) Delete(ctx context.Context, playerID int64, kinds ...string) error {
	keys := make([]string, 0, len(kinds))
	for _, kind := range kinds {
		keys = append(keys, c.Key(playerID, kind))
	}
	return c.rdb.Del(ctx, keys...).Err()
}
