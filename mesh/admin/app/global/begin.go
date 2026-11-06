package global

import (
	"context"
	"server-cluster/share/consts"
	"server-cluster/share/model"
	"server-cluster/share/tpl"

	"github.com/dobyte/due/v2/log"

	"server-cluster/internal/svc"
)

// 活动模块启停由 logic 包在 init 中注入，避免 global 直接依赖 logic 形成导入环。
var (
	activityStarter func(ctx context.Context) error
	activityStopper func()
)

// SetActivityLifecycle 注入活动模块启停（logic 包初始化时自动调用）。
func SetActivityLifecycle(start func(ctx context.Context) error, stop func()) {
	activityStarter = start
	activityStopper = stop
}

// Init 初始化应用：svc（DB/Redis/雪花ID/JWT）→ 自动建表 → 种子数据，须在注册路由前调用。
func Init() error {
	if err := svc.Init(); err != nil {
		return err
	}
	gdb, err := svc.Ctx().DB()
	if err != nil {
		log.Warnf("admin: mysql not configured, admin apis disabled: %v", err)
		gdb = nil
	}
	// 自动建表（系统相关表）
	if inst, err := svc.Ctx().MySQL.Default(); err == nil {
		if err = svc.AutoMigrate(inst.Gorm(),
			&model.SysUser{},
			&model.SysRole{},
			&model.SysMenu{},
			&model.SysApi{},
			&model.SysDictType{},
			&model.SysDictData{},
			&model.SysConfig{},
			&model.SysLoginLog{},
			&model.SysOperaLog{},
			&model.SysCasbinRule{},
			&model.ClientVersion{},
			&model.ClusterNode{},
			&model.Activity{},
			&model.PlayerActivity{},
			&model.PlayerTask{},
		); err != nil {
			return err
		}
	}
	// 载入 Luban 配置表（物品配置等 GM 展示数据）
	if err = tpl.Load(); err != nil {
		return err
	}
	if gdb != nil {
		if err := seed(gdb); err != nil {
			return err
		}
		if err := syncMenus(gdb); err != nil {
			return err
		}
		// 启动内嵌活动模块（活动状态机 + Redis 快照发布）
		if activityStarter != nil {
			if err := activityStarter(context.Background()); err != nil {
				return err
			}
		}
	}
	return nil
}

// Destroy 进程退出后释放 DB / Redis 等资源，并按统一停服日志输出。
func Destroy() {
	if activityStopper != nil {
		activityStopper()
	}
	svc.CloseWithLog(consts.MeshInstanceID())
}
