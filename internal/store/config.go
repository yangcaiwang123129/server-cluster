package store

import (
	"time"

	"github.com/dobyte/due/v2/etc"

	"server-cluster/internal/store/localcache"
)

// Config 存储层配置，对应 etc.yaml 的 store 段（键前缀 etc.store.*）。
type Config struct {
	WALID                string        // 实例级 WAL 标识，每实例唯一且需持久化
	WALDir               string        // WAL 根目录，实际路径 {WALDir}/{WALID}
	WALSyncMode          string        // strict（每条 fsync）| batch（批量 Sync）
	WALBatchSyncInterval time.Duration // batch 模式 Sync 间隔
	Local                LocalConfig
	Redis                RedisConfig
	MySQL                MySQLConfig
	Version              VersionConfig
}

// LocalConfig L1 本地缓存配置。
type LocalConfig struct {
	Driver  string        // 固定 theine（预留）
	Mode    string        // loading（Mesh，带 loader+singleflight）| simple（Node）
	MaxSize int64         // 容量 ≈ 在线玩家数 × 1.2
	TTL     time.Duration // L1 TTL，须短于 Redis TTL
}

// RedisConfig 共享读缓存配置；Node 置 Enabled=false。
// 连接实例引用 [[redis.instances]] 中的命名实例，本段不再配置连接参数。
type RedisConfig struct {
	Enabled           bool
	Instance          string        // Redis 命名实例名，默认 default
	KeyPrefix         string        // 键前缀，默认 player
	TTL               time.Duration // 默认 15m
	TTLJitter         time.Duration // TTL 随机抖动上限，默认 300s
	InvalidateChannel string        // 失效广播频道
}

// MySQLConfig 真相源配置。
// 分片引用 [[db.mysql.instances]] 中的命名实例名，本段不再配置连接参数。
type MySQLConfig struct {
	Shards         []string      // 分片实例名列表；当前单分片，路由经 ShardRouter
	FlushBatchSize int           // flush worker 批量阈值，默认 200
	FlushInterval  time.Duration // flush worker 刷盘间隔，默认 200ms
}

// VersionConfig 版本号生成器配置。
type VersionConfig struct {
	Driver string // redis_incr（默认，INCR 优先降级 MySQL）| mysql_fallback | local
}

// 配置键（etc.yaml 的 store 段）。
const (
	keyStore           = "etc.store"
	keyWALID           = "etc.store.wal_id"
	keyWALDir          = "etc.store.wal_dir"
	keyWALSyncMode     = "etc.store.wal_sync_mode"
	keyWALBatchSync    = "etc.store.wal_batch_sync_interval"
	keyLocalMode       = "etc.store.local.mode"
	keyLocalMaxSize    = "etc.store.local.max_size"
	keyLocalTTL        = "etc.store.local.ttl"
	keyRedisEnabled    = "etc.store.redis.enabled"
	keyRedisInstance   = "etc.store.redis.instance"
	keyRedisPrefix     = "etc.store.redis.key_prefix"
	keyRedisTTL        = "etc.store.redis.ttl"
	keyRedisTTLJitter  = "etc.store.redis.ttl_jitter"
	keyRedisInvalidate = "etc.store.redis.invalidate_channel"
	keyMySQLShards     = "etc.store.mysql.shards"
	keyFlushBatchSize  = "etc.store.mysql.flush_batch_size"
	keyFlushInterval   = "etc.store.mysql.flush_interval"
	keyVersionDriver   = "etc.store.version.driver"
)

// Load 从 etc 加载存储配置；etc.store 段缺席时返回 (nil, nil)，调用方应跳过存储初始化。
func Load() (*Config, error) {
	if !etc.Has(keyStore) {
		return nil, nil
	}
	cfg := &Config{
		WALID:                etc.Get(keyWALID).String(),
		WALDir:               etc.Get(keyWALDir, "./data/wal").String(),
		WALSyncMode:          etc.Get(keyWALSyncMode, "strict").String(),
		WALBatchSyncInterval: etc.Get(keyWALBatchSync, 200*time.Millisecond).Duration(),
		Local: LocalConfig{
			Mode:    etc.Get(keyLocalMode, localcache.ModeLoading).String(),
			MaxSize: etc.Get(keyLocalMaxSize, 100000).Int64(),
			TTL:     etc.Get(keyLocalTTL, time.Minute).Duration(),
		},
		Redis: RedisConfig{
			Enabled:           etc.Get(keyRedisEnabled, true).Bool(),
			Instance:          etc.Get(keyRedisInstance, "default").String(),
			KeyPrefix:         etc.Get(keyRedisPrefix, "player").String(),
			TTL:               etc.Get(keyRedisTTL, 15*time.Minute).Duration(),
			TTLJitter:         etc.Get(keyRedisTTLJitter, 300*time.Second).Duration(),
			InvalidateChannel: etc.Get(keyRedisInvalidate, "player:invalidate").String(),
		},
		MySQL: MySQLConfig{
			Shards:         etc.Get(keyMySQLShards, []string{"default"}).Strings(),
			FlushBatchSize: etc.Get(keyFlushBatchSize, 200).Int(),
			FlushInterval:  etc.Get(keyFlushInterval, 200*time.Millisecond).Duration(),
		},
		Version: VersionConfig{
			Driver: etc.Get(keyVersionDriver, "redis_incr").String(),
		},
	}
	return cfg, nil
}
