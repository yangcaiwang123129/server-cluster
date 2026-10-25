// Package version 跨实例单调版本号生成器：redis_incr（默认，INCR 优先降级 MySQL）
// | mysql_fallback（事务 version+1）| local（时间戳+序列，仅降级兜底，不保证跨实例语义）。
package version

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/dobyte/due/v2/log"
	"server-cluster/internal/store/mysql"
)

// 生成器驱动名（etc.store.version.driver）。
const (
	DriverRedisIncr     = "redis_incr"
	DriverMySQLFallback = "mysql_fallback"
	DriverLocal         = "local"
)

// Generator 版本号生成器。
type Generator interface {
	Next(ctx context.Context, playerID int64) (uint64, error)
}

// New 按驱动组装降级链：redis_incr → mysql_fallback → local。
// rdb 为 nil 时 redis_incr 直接落到 MySQL 层。
func New(driver string, rdb redis.UniversalClient, prefix string, repo *mysql.PlayerRepo) Generator {
	local := &LocalGenerator{}
	mysqlGen := &MySQLGenerator{repo: repo, fallback: local}
	switch driver {
	case DriverLocal:
		return local
	case DriverMySQLFallback:
		return mysqlGen
	default: // redis_incr
		if rdb == nil {
			log.Warnf("version: redis unavailable, driver redis_incr falls back to mysql")
			return mysqlGen
		}
		return &RedisGenerator{rdb: rdb, prefix: prefix, fallback: mysqlGen}
	}
}

// RedisGenerator INCR `{prefix}:{playerID}:version`；失败降级到 fallback 并告警。
type RedisGenerator struct {
	rdb      redis.UniversalClient
	prefix   string
	fallback Generator
}

// Next 递增并返回版本号。
func (g *RedisGenerator) Next(ctx context.Context, playerID int64) (uint64, error) {
	key := fmt.Sprintf("%s:%d:version", g.prefix, playerID)
	v, err := g.rdb.Incr(ctx, key).Result()
	if err != nil {
		log.Warnf("version: redis incr key=%s failed: %v, fallback to mysql", key, err)
		return g.fallback.Next(ctx, playerID)
	}
	return uint64(v), nil
}

// MySQLGenerator 事务内 player_core.version+1；失败降级到 fallback。
type MySQLGenerator struct {
	repo     *mysql.PlayerRepo
	fallback Generator
}

// Next 事务递增并返回版本号。
func (g *MySQLGenerator) Next(ctx context.Context, playerID int64) (uint64, error) {
	v, err := g.repo.NextVersion(ctx, playerID)
	if err != nil {
		log.Warnf("version: mysql next_version player=%d failed: %v, fallback to local", playerID, err)
		return g.fallback.Next(ctx, playerID)
	}
	return v, nil
}

// LocalGenerator 本地降级兜底：毫秒时间戳<<20 | 自增序列（20bit）。
// 不保证跨实例单调，仅在 Redis/MySQL 均不可用时避免写路径完全不可用。
type LocalGenerator struct {
	seq atomic.Uint64
}

// Next 生成本地版本号。
func (g *LocalGenerator) Next(_ context.Context, _ int64) (uint64, error) {
	ms := uint64(time.Now().UnixMilli())
	s := g.seq.Add(1) & 0xFFFFF
	return ms<<20 | s, nil
}
