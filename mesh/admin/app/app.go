// Package app 装配层：初始化基础设施、建表、种子数据与令牌签发器。
package app

import (
	"github.com/dobyte/due/v2/etc"
	"github.com/dobyte/due/v2/log"
	"github.com/dobyte/jwt"

	"server-cluster/internal/db"
	"server-cluster/internal/svc"
	"server-cluster/mesh/admin/app/repo"
	"server-cluster/share/model"
)

var jwtInst *jwt.JWT

// JWT 获取令牌签发/校验器。
func JWT() *jwt.JWT { return jwtInst }

// adminTables 管理后台全部表模型。
var adminTables = []any{
	&model.SysUser{},
	&model.SysRole{},
	&model.SysMenu{},
	&model.SysApi{},
	&model.SysDictType{},
	&model.SysDictData{},
	&model.SysConfig{},
	&model.SysLoginLog{},
	&model.SysOperaLog{},
	&model.CasbinRule{},
}

// Init 初始化应用：svc（DB/Redis）→ 自动建表 → 种子数据 → JWT。
func Init() error {
	if err := svc.Init(); err != nil {
		return err
	}
	gdb, err := svc.Ctx().DB()
	if err != nil {
		log.Warnf("admin: mysql not configured, admin apis disabled: %v", err)
		gdb = nil
	}
	if err := db.AutoMigrate(gdb, adminTables...); err != nil {
		return err
	}
	repo.Init(gdb)
	if gdb != nil {
		if err := seed(gdb); err != nil {
			return err
		}
	}
	inst, err := jwt.NewJWT(
		jwt.WithSecretKey(etc.Get("etc.jwt.secret").String()),
		jwt.WithSignAlgorithm(jwt.HS256),
		jwt.WithValidDuration(etc.Get("etc.jwt.valid_duration", 7200).Int()),
	)
	if err != nil {
		return err
	}
	jwtInst = inst
	return nil
}

// Destroy 进程退出后释放 DB / Redis 等资源。
func Destroy() {
	_ = svc.Close()
}
