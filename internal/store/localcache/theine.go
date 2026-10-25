// Package localcache 本地 L1 缓存：统一 theine-go 封装。
// Mesh 用 loading 模式（loader 回源 + 内置 singleflight 防击穿），Node 用 simple 模式。
package localcache

import (
	"context"
	"fmt"
	"time"

	"github.com/Yiling-J/theine-go"
)

// 缓存模式。
const (
	ModeLoading = "loading"
	ModeSimple  = "simple"
)

// Loader 回源函数；缓存 miss 时调用。返回 store.ErrNotFound 表示数据不存在。
type Loader[V any] func(ctx context.Context, key string) (V, error)

// Config 本地缓存配置。
type Config struct {
	Mode    string        // loading | simple
	MaxSize int64         // 最大条目数
	TTL     time.Duration // 条目 TTL
}

// Cache L1 缓存统一接口，业务层不感知底层模式差异。
type Cache[V any] interface {
	// Get 命中直接返回；miss 时回源 loader 并回填。
	Get(ctx context.Context, key string) (V, error)
	// Set 写路径成功后更新本地缓存。
	Set(key string, val V)
	// Delete 失效广播 / 删除时剔除。
	Delete(key string)
	// Close 释放缓存资源。
	Close()
}

// New 按模式构造缓存；loader 为 nil 时 simple 模式 miss 直接返回错误。
func New[V any](cfg Config, loader Loader[V]) (Cache[V], error) {
	if cfg.MaxSize <= 0 {
		cfg.MaxSize = 100000
	}
	if cfg.TTL <= 0 {
		cfg.TTL = time.Minute
	}
	if cfg.Mode == ModeSimple {
		return newSimple[V](cfg, loader)
	}
	return newLoading[V](cfg, loader)
}

// loadingCache Mesh 模式：theine LoadingCache，内置 singleflight 合并回源。
type loadingCache[V any] struct {
	c   *theine.LoadingCache[string, V]
	ttl time.Duration
}

func newLoading[V any](cfg Config, loader Loader[V]) (Cache[V], error) {
	ttl := cfg.TTL
	c, err := theine.NewBuilder[string, V](cfg.MaxSize).BuildWithLoader(
		func(ctx context.Context, key string) (theine.Loaded[V], error) {
			v, err := loader(ctx, key)
			if err != nil {
				return theine.Loaded[V]{}, err
			}
			return theine.Loaded[V]{Value: v, Cost: 1, TTL: ttl}, nil
		})
	if err != nil {
		return nil, fmt.Errorf("localcache: build loading cache failed: %w", err)
	}
	return &loadingCache[V]{c: c, ttl: ttl}, nil
}

func (c *loadingCache[V]) Get(ctx context.Context, key string) (V, error) {
	return c.c.Get(ctx, key)
}

func (c *loadingCache[V]) Set(key string, val V) { c.c.SetWithTTL(key, val, 1, c.ttl) }

func (c *loadingCache[V]) Delete(key string) { c.c.Delete(key) }

func (c *loadingCache[V]) Close() { c.c.Close() }

// simpleCache Node 模式：无 loader 的纯缓存，miss 时由封装层调用回源函数。
type simpleCache[V any] struct {
	c      *theine.Cache[string, V]
	ttl    time.Duration
	loader Loader[V]
}

func newSimple[V any](cfg Config, loader Loader[V]) (Cache[V], error) {
	c, err := theine.NewBuilder[string, V](cfg.MaxSize).Build()
	if err != nil {
		return nil, fmt.Errorf("localcache: build simple cache failed: %w", err)
	}
	return &simpleCache[V]{c: c, ttl: cfg.TTL, loader: loader}, nil
}

func (c *simpleCache[V]) Get(ctx context.Context, key string) (V, error) {
	if v, ok := c.c.Get(key); ok {
		return v, nil
	}
	var zero V
	if c.loader == nil {
		return zero, fmt.Errorf("localcache: miss and no loader, key=%s", key)
	}
	v, err := c.loader(ctx, key)
	if err != nil {
		return zero, err
	}
	c.c.SetWithTTL(key, v, 1, c.ttl)
	return v, nil
}

func (c *simpleCache[V]) Set(key string, val V) { c.c.SetWithTTL(key, val, 1, c.ttl) }

func (c *simpleCache[V]) Delete(key string) { c.c.Delete(key) }

func (c *simpleCache[V]) Close() { c.c.Close() }
