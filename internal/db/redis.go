package db

import (
	"fmt"
	"sync"

	"github.com/dobyte/due/v2/core/value"
	"github.com/dobyte/due/v2/etc"
	"github.com/redis/go-redis/v9"
)

const keyRedisInstances = "etc.redis.instances"

// RedisConfig 单个 Redis 实例配置（对应 [[redis.instances]]）。
// 不同业务可按实例使用不同 db 号或不同集群，业务按 name 获取。
type RedisConfig struct {
	Name     string   `json:"name"`
	Addrs    []string `json:"addrs"` // 为空时回退 127.0.0.1:6379
	Username string   `json:"username"`
	Password string   `json:"password"`
	DB       int      `json:"db"`
}

// RedisManager Redis 命名实例管理器，进程内单次创建、并发安全。
type RedisManager struct {
	mu    sync.RWMutex
	insts map[string]redis.UniversalClient
}

// NewRedisManager 按 etc.redis.instances 创建全部客户端（懒连接，不在启动期 Ping）。
func NewRedisManager() (*RedisManager, error) {
	m := &RedisManager{insts: make(map[string]redis.UniversalClient)}
	for i, item := range etc.Get(keyRedisInstances).Slice() {
		var cfg RedisConfig
		if err := value.NewValue(item).Scan(&cfg); err != nil {
			_ = m.Close()
			return nil, fmt.Errorf("db: parse %s[%d] failed: %w", keyRedisInstances, i, err)
		}
		if cfg.Name == "" {
			_ = m.Close()
			return nil, fmt.Errorf("db: redis instance name is required")
		}
		if _, ok := m.insts[cfg.Name]; ok {
			_ = m.Close()
			return nil, fmt.Errorf("db: duplicate redis instance %q", cfg.Name)
		}
		addrs := cfg.Addrs
		if len(addrs) == 0 {
			addrs = []string{"127.0.0.1:6379"}
		}
		m.insts[cfg.Name] = redis.NewUniversalClient(&redis.UniversalOptions{
			Addrs:    addrs,
			Username: cfg.Username,
			Password: cfg.Password,
			DB:       cfg.DB,
		})
	}
	return m, nil
}

// Get 获取指定名称的 Redis 实例。
func (m *RedisManager) Get(name string) (redis.UniversalClient, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	client, ok := m.insts[name]
	if !ok {
		return nil, fmt.Errorf("db: redis instance %q is not configured", name)
	}
	return client, nil
}

// Default 获取默认 Redis 实例。
func (m *RedisManager) Default() (redis.UniversalClient, error) { return m.Get(DefaultInstance) }

// Close 关闭全部客户端。
func (m *RedisManager) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	var err error
	for name, client := range m.insts {
		if e := client.Close(); e != nil && err == nil {
			err = fmt.Errorf("db: close redis instance %q failed: %w", name, e)
		}
	}
	m.insts = make(map[string]redis.UniversalClient)
	return err
}
