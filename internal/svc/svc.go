// Package svc 服务上下文：集中持有 MySQL / Redis 命名实例管理器等基础设施句柄。
package svc

import (
	"gorm.io/gorm"

	"server-cluster/internal/db"
)

// Context 基础设施句柄集合，以后扩展 etcd、kafka 等都挂在这里。
type Context struct {
	MySQL *db.MySQLManager // 命名 MySQL 实例
	Redis *db.RedisManager // 命名 Redis 实例（不同业务可用不同 db 号）
}

var ctx *Context

// Init 初始化服务上下文，须在 app.Init 最前调用。
func Init() error {
	mysqlMgr, err := db.NewMySQLManager()
	if err != nil {
		return err
	}
	redisMgr, err := db.NewRedisManager()
	if err != nil {
		_ = mysqlMgr.Close()
		return err
	}
	ctx = &Context{MySQL: mysqlMgr, Redis: redisMgr}
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
