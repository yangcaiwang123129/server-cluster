// Package svc 服务上下文：集中持有 MySQL / Redis 命名实例管理器、JWT 签发器与雪花 ID 生成器等基础设施句柄。
package svc

import (
	"github.com/dobyte/due/v2/etc"
	"github.com/dobyte/due/v2/log"
	"github.com/dobyte/jwt"
	"gorm.io/gorm"
)

// Context 基础设施句柄集合，以后扩展 etcd、kafka 等都挂在这里。
type Context struct {
	MySQL     *MySQLManager // 命名 MySQL 实例
	Redis     *RedisManager // 命名 Redis 实例（不同业务可用不同 db 号）
	JWT       *jwt.JWT      // 令牌签发/校验器；etc.jwt.secret 未配置的服务为 nil
	Snowflake *Snowflake    // 雪花 ID 生成器
}

var ctx *Context

// Init 初始化服务上下文，须在 global.Init 最前调用。
// 初始化顺序：MySQL → Redis → 雪花 ID → JWT（secret 未配置则跳过）。
func Init() error {
	mysqlMgr, err := NewMySQLManager()
	if err != nil {
		return err
	}
	redisMgr, err := NewRedisManager()
	if err != nil {
		_ = mysqlMgr.Close()
		return err
	}
	c := &Context{MySQL: mysqlMgr, Redis: redisMgr}
	// 雪花 ID 生成器：节点号由 etc.snowflake.node_id 配置（∈[0,1023]），单机默认 1
	sf, err := newSnowflake(etc.Get("etc.snowflake.node_id", 1).Int64())
	if err != nil {
		_ = mysqlMgr.Close()
		_ = redisMgr.Close()
		return err
	}
	c.Snowflake = sf
	// JWT 签发器：按需初始化，不签发令牌的服务（gate/battle/social 等）无需配置
	if secret := etc.Get("etc.jwt.secret").String(); secret != "" {
		inst, err := jwt.NewJWT(
			jwt.WithSecretKey(secret),
			jwt.WithSignAlgorithm(jwt.HS256),
			jwt.WithValidDuration(etc.Get("etc.jwt.valid_duration", 7200).Int()),
		)
		if err != nil {
			_ = mysqlMgr.Close()
			_ = redisMgr.Close()
			return err
		}
		c.JWT = inst
	} else {
		log.Warnf("svc: etc.jwt.secret not configured, jwt disabled")
	}
	ctx = c
	return nil
}

// Ctx 获取服务上下文；须先 Init。
func Ctx() *Context {
	if ctx == nil {
		panic("svc context is not initialized, call svc.Init() first")
	}
	return ctx
}

// DB 默认 MySQL 实例的 gorm 句柄；实例未配置时返回错误。
func (c *Context) DB() (*gorm.DB, error) {
	inst, err := c.MySQL.Default()
	if err != nil {
		return nil, err
	}
	return inst.Gorm(), nil
}

// NextID 生成下一个雪花 ID。
func NextID() int64 { return Ctx().Snowflake.Next() }

// Close 释放资源，进程退出前调用。
func Close() error {
	if ctx == nil {
		return nil
	}
	err := ctx.MySQL.Close()
	if e := ctx.Redis.Close(); err == nil {
		err = e
	}
	ctx = nil
	return err
}
