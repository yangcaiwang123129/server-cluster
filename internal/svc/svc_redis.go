package svc

import (
	"fmt"
	"sync"

	"github.com/dobyte/due/v2/core/value"
	"github.com/dobyte/due/v2/etc"
	"github.com/redis/go-redis/v9"

	"server-cluster/share/consts"
)

const keyRedisInstances = "etc.redis.instances"

// RedisConfig 单个 Redis 实例配置（对应 [[redis.instances]]）。
// 不同业务可按实例使用不同 db 号或不同集群，业务按 name 获取。
type RedisConfig struct {
	Name     string   `json:"name"`
	Addrs    []string `json:"addrs"` // 为空时回退 consts.DefaultRedisAddr
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
			return nil, fmt.Errorf("svc: parse %s[%d] failed: %w", keyRedisInstances, i, err)
		}
		if cfg.Name == "" {
			_ = m.Close()
			return nil, fmt.Errorf("svc: redis instance name is required")
		}
		if _, ok := m.insts[cfg.Name]; ok {
			_ = m.Close()
			return nil, fmt.Errorf("svc: duplicate redis instance %q", cfg.Name)
		}
		addrs := cfg.Addrs
		if len(addrs) == 0 {
			addrs = []string{consts.DefaultRedisAddr}
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
func (this *RedisManager) Get(name string) (redis.UniversalClient, error) {
	this.mu.RLock()
	defer this.mu.RUnlock()
	client, ok := this.insts[name]
	if !ok {
		return nil, fmt.Errorf("svc: redis instance %q is not configured", name)
	}
	return client, nil
}

// Default 获取默认 Redis 实例。
func (this *RedisManager) Default() (redis.UniversalClient, error) { return this.Get(DefaultInstance) }

// Len 返回已建立的 Redis 实例数（停服日志展示用）。
func (this *RedisManager) Len() int {
	this.mu.RLock()
	defer this.mu.RUnlock()
	return len(this.insts)
}

// Close 关闭全部客户端。
func (this *RedisManager) Close() error {
	this.mu.Lock()
	defer this.mu.Unlock()
	var err error
	for name, client := range this.insts {
		if e := client.Close(); e != nil && err == nil {
			err = fmt.Errorf("svc: close redis instance %q failed: %w", name, e)
		}
	}
	this.insts = make(map[string]redis.UniversalClient)
	return err
}
